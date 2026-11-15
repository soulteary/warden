package main

import (
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// envelopeChange returns a change to the envelope a replica published.
func envelopeChange(old, replacement string) func(string) string {
	return func(published string) string { return strings.Replace(published, old, replacement, 1) }
}

func storedValue(value string) func(string) string {
	return func(string) string { return value }
}

// A stored value that is not a publication of this dataset and policy (over
// the size limit, not JSON, or JSON that is not such an envelope) is replaced
// by the lease holder even when it holds a revision, as a replica without a
// revision already does. Otherwise the key keeps a value no replica can
// restore for as long as the source keeps the writer on its retained set.
func TestV2UnusableSnapshotIsReplacedByRevisionHolder(t *testing.T) {
	for name, tc := range map[string]struct {
		stored  func(published string) string
		outcome string
	}{
		"not_json":           {storedValue("{"), "invalid_envelope"},
		"empty_value":        {storedValue(""), "invalid_envelope"},
		"json_scalar":        {storedValue("7"), "invalid_envelope"},
		"over_the_limit":     {storedValue(strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1)), "too_large"},
		"empty_object":       {storedValue(`{}`), "invalid_envelope"},
		"empty_array":        {storedValue(`[]`), "invalid_envelope"},
		"revision_only":      {storedValue(`{"revision":"` + strings.Repeat("c", 32) + `"}`), "invalid_envelope"},
		"no_revision":        {envelopeChange(`"revision":`, `"was_revision":`), "invalid_envelope"},
		"revision_not_token": {envelopeChange(`"revision":"`, `"revision":"not-a-revision","was_revision":"`), "invalid_envelope"},
		"revision_not_text":  {envelopeChange(`"revision":"`, `"revision":7,"was_revision":"`), "invalid_envelope"},
		"earlier_schema":     {envelopeChange(`"schema_version":2`, `"schema_version":1`), "invalid_envelope"},
		"no_schema_version":  {envelopeChange(`"schema_version":2`, `"was_schema_version":2`), "invalid_envelope"},
		"other_dataset":      {envelopeChange(`"dataset":"migration-test"`, `"dataset":"another-dataset"`), "dataset_rejected"},
		"other_policy":       {envelopeChange(strings.Repeat("a", 64), strings.Repeat("b", 64)), "policy_rejected"},
	} {
		t.Run(name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			sourceTime := time.Now().Add(-time.Minute).UTC()
			published := publishMigrationSource(t, app, sourceTime, []define.AllowListUser{{Mail: "kept@example.com", UserID: "kept"}})
			rejectedCycle(t, app)
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)

			own, err := mr.Get(migrationSnapshotKey)
			require.NoError(t, err)
			unusable := tc.stored(own)
			require.NotEqual(t, own, unusable)
			require.NoError(t, mr.Set(migrationSnapshotKey, unusable))
			renewals := snapshotOutcomeCount(t, "v2", "renew", tc.outcome)
			publications := snapshotOutcomeCount(t, "v2", "publish", "published")
			rejectedCycle(t, app)
			require.Equal(t, renewals+1, snapshotOutcomeCount(t, "v2", "renew", tc.outcome))
			require.Equal(t, publications+1, snapshotOutcomeCount(t, "v2", "publish", "published"))

			stored, err := app.redisSnapshots.Read(t.Context())
			require.NoError(t, err, "the key is usable again after the next cycle")
			require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, app))
			require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(migrationSnapshotKey))
			require.Equal(t, sourceTime, stored.SourceSucceededAt, "the replacement does not renew the source evidence")
			current := app.snapshots.Load()
			require.NotEqual(t, published.RedisRevision, current.RedisRevision)
			require.Equal(t, stored.Revision, current.RedisRevision)

			// The following cycle renews the replacement instead of writing again.
			rejectedCycle(t, app)
			require.Equal(t, stored.Revision, app.snapshots.Load().RedisRevision)

			restarted := &App{redisClient: app.redisClient, redisSnapshotMode: "v2", redisSnapshots: app.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: app.log}
			require.True(t, restarted.bootstrapFromRedis())
		})
	}
}

// A publication of this dataset and policy with another revision stays a
// conflict: another writer may have published since. So does a value that
// states a later schema version, which a newer release may be writing under
// the same key and this release cannot judge.
func TestV2PublicationWithAnotherRevisionIsLeftAlone(t *testing.T) {
	other := strings.Repeat("b", 32)
	for name, stored := range map[string]func(published, revision string) string{
		"other_revision":          func(published, revision string) string { return strings.Replace(published, revision, other, 1) },
		"other_revision_indented": func(published, revision string) string { return "\n" + strings.Replace(published, revision, other, 1) },
		"later_schema": func(published, _ string) string {
			return strings.Replace(published, `"schema_version":2`, `"schema_version":3`, 1)
		},
		"later_schema_other_shape": func(string, string) string { return `{"schema_version":3,"revision":"` + other + `"}` },
		"later_schema_not_decoded": func(string, string) string { return `{"schema_version":3,"records":[` },
	} {
		t.Run(name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			published := publishMigrationSource(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
			own, err := mr.Get(migrationSnapshotKey)
			require.NoError(t, err)
			value := stored(own, published.RedisRevision)
			require.NotEqual(t, own, value)
			require.NoError(t, mr.Set(migrationSnapshotKey, value))
			conflicts := snapshotOutcomeCount(t, "v2", "renew", "revision_conflict")
			publications := snapshotOutcomeCount(t, "v2", "publish", "published")
			rejectedCycle(t, app)
			rejectedCycle(t, app)
			require.Equal(t, conflicts+2, snapshotOutcomeCount(t, "v2", "renew", "revision_conflict"))
			require.Equal(t, publications, snapshotOutcomeCount(t, "v2", "publish", "published"))
			after, err := mr.Get(migrationSnapshotKey)
			require.NoError(t, err)
			require.Equal(t, value, after)
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)
		})
	}
}

// The replacement of an unusable value follows the rule for a retained set: a
// lease is still required.
func TestV2UnusableSnapshotIsNotReplacedWithoutTheLease(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	published := publishMigrationSource(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
	app.snapshots.RecordRefreshFailure(reasonAllRecordsRejected)
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	require.NoError(t, mr.Set(migrationSnapshotKey, "{"))
	mr.Del(migrationSnapshotKey + ":lease")
	require.ErrorIs(t, app.publishRedisSnapshots(app.userCache.Get()), cache.ErrSnapshotLeaseLost)
	after, err := mr.Get(migrationSnapshotKey)
	require.NoError(t, err)
	require.Equal(t, "{", after)
	require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)
}
