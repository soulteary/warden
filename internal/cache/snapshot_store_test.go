package cache

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

func snapshotStoreFixture(t *testing.T) (*RedisSnapshotStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store, err := NewRedisSnapshotStore(client, "test-dataset", strings.Repeat("a", 64))
	require.NoError(t, err)
	return store, mr
}

func sourceSnapshot(at time.Time) *SnapshotEnvelope {
	users := []define.AllowListUser{{Mail: "member@example.com", UserID: "member", Status: "active", Scope: []string{"read"}}}
	return &SnapshotEnvelope{Users: users, EffectiveVersion: HashUserList(users), InputVersion: HashUserList(users), InputCount: 1, Origin: "remote", SourceSucceededAt: at}
}

func TestVersionedSnapshotTimesSurviveRenewalAndRecovery(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	missing, err := store.Read(ctx)
	require.NoError(t, err)
	require.Nil(t, missing)
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	input := sourceSnapshot(now.Add(-2 * time.Minute))
	published, err := lease.Publish(ctx, input)
	require.NoError(t, err)
	require.Equal(t, input.SourceSucceededAt, published.SourceSucceededAt)
	require.Equal(t, now, published.PublishedAt)
	original, err := store.client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	mr.FastForward(30 * time.Second)
	now = now.Add(30 * time.Second)
	require.NoError(t, lease.Renew(ctx, published.Revision))
	renewed, err := store.client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.Equal(t, original, renewed, "renewal changes only expiry")
	recovered, err := NewRedisSnapshotStore(store.client, "test-dataset", store.policy)
	require.NoError(t, err)
	recovered.now = store.now
	got, err := recovered.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, published, got)
	require.Equal(t, 150*time.Second, now.Sub(got.SourceSucceededAt))
	require.NoError(t, lease.Release(ctx))
}

func TestVersionedSnapshotDistinguishesEmptyFromMissing(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	input := sourceSnapshot(time.Now())
	input.Users = []define.AllowListUser{}
	input.EffectiveVersion = HashUserList(nil)
	input.InputVersion = HashUserList(nil)
	input.InputCount = 0
	_, err = lease.Publish(ctx, input)
	require.NoError(t, err)
	got, err := store.Read(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Users)
	require.Empty(t, got.Users)
}

func TestVersionedSnapshotExpiredWriterCannotPublishOrReleaseSuccessor(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	first, err := store.Acquire(ctx, time.Second)
	require.NoError(t, err)
	busy, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.Nil(t, busy)
	mr.FastForward(2 * time.Second)
	second, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, second)
	newSnapshot, err := second.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	_, err = first.Publish(ctx, sourceSnapshot(time.Now().Add(-time.Minute)))
	require.ErrorIs(t, err, ErrSnapshotLeaseLost)
	require.ErrorIs(t, first.Renew(ctx, newSnapshot.Revision), ErrSnapshotLeaseLost)
	require.NoError(t, first.Release(ctx))
	require.NoError(t, second.Renew(ctx, newSnapshot.Revision))
	got, err := store.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, newSnapshot.Revision, got.Revision)
}

func TestVersionedSnapshotRenewCannotReplaceNewPublication(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	old, err := lease.Publish(ctx, sourceSnapshot(time.Now().Add(-time.Minute)))
	require.NoError(t, err)
	latest, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	require.NotEqual(t, old.Revision, latest.Revision)
	require.ErrorIs(t, lease.Renew(ctx, old.Revision), ErrSnapshotConflict)
	got, err := store.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, latest, got)
}

func TestVersionedSnapshotRejectsInvalidContractsAndPayloads(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	valid, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	for name, mutate := range map[string]func(*SnapshotEnvelope){
		"schema":              func(e *SnapshotEnvelope) { e.SchemaVersion = 3 },
		"dataset":             func(e *SnapshotEnvelope) { e.Dataset = "other" },
		"policy":              func(e *SnapshotEnvelope) { e.Policy = strings.Repeat("b", 64) },
		"null_users":          func(e *SnapshotEnvelope) { e.Users = nil },
		"version":             func(e *SnapshotEnvelope) { e.EffectiveVersion = strings.Repeat("b", 64) },
		"missing_source_time": func(e *SnapshotEnvelope) { e.SourceSucceededAt = time.Time{} },
		"future_time":         func(e *SnapshotEnvelope) { e.SourceSucceededAt = time.Now().Add(time.Hour) },
		"future_publication":  func(e *SnapshotEnvelope) { e.PublishedAt = time.Now().Add(time.Hour) },
		"origin":              func(e *SnapshotEnvelope) { e.Origin = "redis" },
		"input_count":         func(e *SnapshotEnvelope) { e.InputCount = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := *valid
			mutate(&invalid)
			body, marshalErr := json.Marshal(invalid)
			require.NoError(t, marshalErr)
			require.NoError(t, store.client.Set(ctx, store.key, body, time.Hour).Err())
			_, err = store.Read(ctx)
			require.Error(t, err)
		})
	}
	for _, raw := range []string{"", "[]", "{}", "invalid", `{"schema_version":2,"unexpected":true}`} {
		require.NoError(t, store.client.Set(ctx, store.key, raw, time.Hour).Err())
		_, err = store.Read(ctx)
		require.Error(t, err)
	}
	_, err = NewRedisSnapshotStore(store.client, "bad{namespace}", store.policy)
	require.ErrorIs(t, err, ErrSnapshotContract)
	_, err = NewRedisSnapshotStore(store.client, "", store.policy)
	require.ErrorIs(t, err, ErrSnapshotContract)
}
