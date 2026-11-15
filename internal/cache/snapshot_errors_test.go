package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

func TestSnapshotReadFailureReasonsPreserveBroadErrors(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	valid, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		change func(*SnapshotEnvelope)
		broad  error
		want   string
	}{
		{"schema", func(e *SnapshotEnvelope) { e.SchemaVersion = 99 }, ErrSnapshotContract, "schema_rejected"},
		{"dataset", func(e *SnapshotEnvelope) { e.Dataset = "other" }, ErrSnapshotContract, "dataset_rejected"},
		{"policy", func(e *SnapshotEnvelope) { e.Policy = strings.Repeat("b", 64) }, ErrSnapshotContract, "policy_rejected"},
		{"time", func(e *SnapshotEnvelope) { e.SourceSucceededAt = time.Time{} }, ErrSnapshotInvalid, "timestamp_rejected"},
		{"content", func(e *SnapshotEnvelope) { e.Users = nil }, ErrSnapshotInvalid, "invalid_envelope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := *valid
			tc.change(&invalid)
			body, marshalErr := json.Marshal(invalid)
			require.NoError(t, marshalErr)
			require.NoError(t, store.client.Set(ctx, store.key, body, time.Minute).Err())
			_, readErr := store.Read(ctx)
			require.ErrorIs(t, readErr, tc.broad)
			require.Equal(t, tc.want, SnapshotFailureReason(readErr))
		})
	}
	require.NoError(t, store.client.Set(ctx, store.key, strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1), time.Minute).Err())
	_, err = store.Read(ctx)
	require.ErrorIs(t, err, ErrSnapshotInvalid)
	require.Equal(t, "too_large", SnapshotFailureReason(err))
	require.NoError(t, store.client.Set(ctx, store.key, "broken", time.Minute).Err())
	_, err = store.Read(ctx)
	require.Equal(t, "invalid_envelope", SnapshotFailureReason(err))
	require.Equal(t, "io_error", SnapshotFailureReason(errors.New("payload-not-a-label")))
}

func TestSnapshotRenewFailureReasons(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	valid, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	for _, tc := range []struct {
		value   string
		want    string
		missing bool
	}{
		{missing: true, want: "missing"}, {value: "broken", want: "invalid_envelope"}, {value: `{"revision":"different"}`, want: "invalid_envelope"},
		{value: store.head + strings.Repeat("0", 32) + `","users":[]}`, want: "revision_conflict"},
		{value: `{"schema_version":3}`, want: "revision_conflict"},
		{value: " " + envelopeHead("other", store.policy) + valid.Revision + `"}`, want: "dataset_rejected"},
		{value: " " + envelopeHead(store.dataset, strings.Repeat("b", 64)) + valid.Revision + `"}`, want: "policy_rejected"}, {value: strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1), want: "too_large"},
	} {
		if tc.missing {
			require.NoError(t, store.client.Del(ctx, store.key).Err())
		} else {
			require.NoError(t, store.client.Set(ctx, store.key, tc.value, time.Minute).Err())
		}
		renewErr := lease.Renew(ctx, valid.Revision)
		require.ErrorIs(t, renewErr, ErrSnapshotConflict)
		require.Equal(t, tc.want, SnapshotFailureReason(renewErr))
	}
	require.NoError(t, lease.Release(ctx))
	require.Equal(t, "lease_lost", SnapshotFailureReason(lease.Renew(ctx, valid.Revision)))
}

func TestSnapshotRenewalFoundNoPublication(t *testing.T) {
	for _, err := range []error{
		ErrSnapshotMissing,
		fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotTooLarge),
		fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotInvalid),
		fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotDataset),
		fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotPolicy),
	} {
		require.True(t, SnapshotRenewalFoundNoPublication(err), err.Error())
	}
	for _, err := range []error{
		nil, ErrSnapshotConflict, ErrSnapshotLeaseLost, ErrSnapshotInvalid, ErrSnapshotTooLarge,
		ErrSnapshotSchema, ErrSnapshotDataset, ErrSnapshotPolicy, fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotSchema), errors.New("renew versioned snapshot: connection refused"),
	} {
		require.False(t, SnapshotRenewalFoundNoPublication(err), "%v", err)
	}
}
