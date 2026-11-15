package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// The renewal's classification of a stored value decides whether the lease
// holder may replace it, and it depends on Redis' own cjson, which accepts and
// rejects some inputs differently from the Lua JSON library miniredis uses.
func TestRedisSnapshotIntegrationRenewClassifiesStoredValues(t *testing.T) {
	store, client := realSnapshotStore(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	revision := strings.Repeat("a", 32)

	require.True(t, SnapshotRenewalFoundNoPublication(lease.Renew(ctx, revision)), "missing key")
	for name, raw := range map[string]string{
		"empty":            "",
		"truncated":        "{",
		"scalar":           "7",
		"string":           `"text"`,
		"null":             "null",
		"trailing_garbage": `{"revision":"` + revision + `"} x`,
		"nested_too_deep":  strings.Repeat("[", 1200) + strings.Repeat("]", 1200),
		"over_the_limit":   strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1),
	} {
		require.NoError(t, client.Set(ctx, store.key, raw, time.Minute).Err())
		renewErr := lease.Renew(ctx, revision)
		require.ErrorIs(t, renewErr, ErrSnapshotConflict, name)
		require.True(t, SnapshotRenewalFoundNoPublication(renewErr), name)
	}
	// Values Redis' cjson decodes, including some that Go's JSON decoder and
	// miniredis reject, are still not envelopes.
	for name, raw := range map[string]string{
		"revision_only":       `{"revision":"` + revision + `"}`,
		"object":              `{}`,
		"array":               `[]`,
		"not_a_number":        `{"revision":NaN}`,
		"schema_not_a_number": `{"schema_version":NaN,"dataset":"` + store.dataset + `","policy":"` + store.policy + `","revision":"` + revision + `"}`,
		"schema_hexadecimal":  `{"schema_version":0x2,"dataset":"` + store.dataset + `","policy":"` + store.policy + `","revision":"` + strings.Repeat("G", 32) + `"}`,
		"hexadecimal":         `{"a":0x10}`,
		"raw_control_char":    "{\"a\":\"line\nbreak\"}",
	} {
		require.NoError(t, client.Set(ctx, store.key, raw, time.Minute).Err())
		renewErr := lease.Renew(ctx, revision)
		require.ErrorIs(t, renewErr, ErrSnapshotConflict, name)
		require.ErrorIs(t, renewErr, ErrSnapshotInvalid, name)
		require.True(t, SnapshotRenewalFoundNoPublication(renewErr), name)
	}
	// A value that states a later schema version stays a conflict in every
	// spelling of the number that Redis' cjson accepts.
	for name, raw := range map[string]string{
		"later_schema":             `{"schema_version":3}`,
		"later_schema_exponent":    `{"schema_version":1e3,"revision":"` + revision + `"}`,
		"later_schema_hexadecimal": `{"schema_version":0x10}`,
		"later_schema_infinity":    `{"schema_version":Infinity}`,
	} {
		require.NoError(t, client.Set(ctx, store.key, raw, time.Minute).Err())
		renewErr := lease.Renew(ctx, revision)
		require.ErrorIs(t, renewErr, ErrSnapshotConflict, name)
		require.False(t, SnapshotRenewalFoundNoPublication(renewErr), name)
	}
	require.NoError(t, lease.Release(ctx))
}
