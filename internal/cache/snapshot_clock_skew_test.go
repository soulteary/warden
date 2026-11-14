package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The store's clock is fixed, so the bound is exact and does not depend on
// how long the test takes: a source or publication time at the allowance is
// accepted on publish and on read, one nanosecond beyond it is refused.
func TestSnapshotFutureAllowanceBoundaryWithInjectedClock(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)

	// Publish stamps the publication time with the store's clock, so the
	// source time is the one that can be ahead of it.
	_, err = lease.Publish(ctx, sourceSnapshot(now.Add(SNAPSHOT_FUTURE_ALLOWANCE+time.Nanosecond)))
	require.ErrorIs(t, err, ErrSnapshotTimestamp)
	require.False(t, mr.Exists(store.key), "a refused publication stores nothing")
	published, err := lease.Publish(ctx, sourceSnapshot(now.Add(SNAPSHOT_FUTURE_ALLOWANCE)))
	require.NoError(t, err)
	require.Equal(t, now.Add(SNAPSHOT_FUTURE_ALLOWANCE), published.SourceSucceededAt)

	// A reader whose clock is behind the writer's sees both times ahead.
	for _, tc := range []struct {
		err    error
		behind time.Duration
	}{
		{nil, 0},
		{ErrSnapshotTimestamp, time.Nanosecond},
		{ErrSnapshotTimestamp, time.Hour},
	} {
		store.now = func() time.Time { return now.Add(-tc.behind) }
		read, readErr := store.Read(ctx)
		if tc.err == nil {
			require.NoError(t, readErr)
			require.Equal(t, published, read)
			continue
		}
		require.ErrorIs(t, readErr, tc.err)
		require.Nil(t, read)
	}

	// A publication time alone beyond the allowance is refused as well.
	store.now = func() time.Time { return now }
	late := *published
	late.SourceSucceededAt = now
	late.PublishedAt = now.Add(SNAPSHOT_FUTURE_ALLOWANCE + time.Nanosecond)
	body, err := json.Marshal(late)
	require.NoError(t, err)
	require.NoError(t, mr.Set(store.key, string(body)))
	_, err = store.Read(ctx)
	require.ErrorIs(t, err, ErrSnapshotTimestamp)
	late.PublishedAt = now.Add(SNAPSHOT_FUTURE_ALLOWANCE)
	body, err = json.Marshal(late)
	require.NoError(t, err)
	require.NoError(t, mr.Set(store.key, string(body)))
	_, err = store.Read(ctx)
	require.NoError(t, err)
}
