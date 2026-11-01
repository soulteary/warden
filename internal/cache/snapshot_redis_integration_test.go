package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// Integration tests require an explicitly selected disposable Redis 8 server.
// Each test owns its keys and ACL user; none flushes the database or script cache.
func realSnapshotStore(t *testing.T) (*RedisSnapshotStore, *redis.Client) {
	t.Helper()
	addr := os.Getenv("WARDEN_REDIS_INTEGRATION_ADDR")
	if addr == "" {
		t.Skip("set WARDEN_REDIS_INTEGRATION_ADDR to run Redis 8 snapshot integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, ContextTimeoutEnabled: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err(), "an explicitly configured integration server must be reachable")
	info, err := client.Info(ctx, "server").Result()
	require.NoError(t, err)
	require.Contains(t, info, "redis_version:8.")
	token, err := newSnapshotToken()
	require.NoError(t, err)
	store, err := NewRedisSnapshotStore(client, "snapshot-it-"+token, strings.Repeat("a", 64))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		require.NoError(t, client.Del(cleanup, store.key, store.lock).Err())
		require.NoError(t, client.Close())
	})
	return store, client
}

// Keep this first: CI uses a fresh Redis service and explicitly requires cold scripts.
func TestRedisSnapshotIntegrationColdScriptsAndLifecycle(t *testing.T) {
	store, client := realSnapshotStore(t)
	ctx := context.Background()
	scripts := []string{readSnapshot.Hash(), publishSnapshot.Hash(), renewSnapshot.Hash(), releaseSnapshot.Hash()}
	if os.Getenv("WARDEN_REDIS_INTEGRATION_REQUIRE_COLD") == "true" {
		exists, err := client.ScriptExists(ctx, scripts...).Result()
		require.NoError(t, err)
		for _, loaded := range exists {
			require.False(t, loaded, "CI must exercise NOSCRIPT fallback against a fresh service")
		}
	}
	missing, err := store.Read(ctx)
	require.NoError(t, err)
	require.Nil(t, missing)
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	sourceTime := time.Now().Add(-10 * time.Minute).UTC()
	published, err := lease.Publish(ctx, sourceSnapshot(sourceTime))
	require.NoError(t, err)
	before, err := client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.NoError(t, client.PExpire(ctx, store.key, 5*time.Second).Err())
	require.NoError(t, lease.Renew(ctx, published.Revision))
	after, err := client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.Equal(t, before, after)
	ttl, err := client.PTTL(ctx, store.key).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, REDIS_CACHE_TTL-5*time.Second)
	recovered, err := NewRedisSnapshotStore(client, store.dataset, store.policy)
	require.NoError(t, err)
	read, err := recovered.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, published, read)
	require.Equal(t, sourceTime, read.SourceSucceededAt)
	empty := sourceSnapshot(sourceTime)
	empty.Users = []define.AllowListUser{}
	empty.InputCount = 0
	empty.InputVersion = HashUserList(nil)
	empty.EffectiveVersion = HashUserList(nil)
	_, err = lease.Publish(ctx, empty)
	require.NoError(t, err)
	read, err = store.Read(ctx)
	require.NoError(t, err)
	require.NotNil(t, read)
	require.NotNil(t, read.Users)
	require.Empty(t, read.Users)
	require.NoError(t, lease.Release(ctx))
	exists, err := client.ScriptExists(ctx, scripts...).Result()
	require.NoError(t, err)
	for _, loaded := range exists {
		require.True(t, loaded)
	}
}

func TestRedisSnapshotIntegrationExpiredAndStaleWriters(t *testing.T) {
	store, client := realSnapshotStore(t)
	ctx := context.Background()
	old, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	busy, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.Nil(t, busy)
	previous, err := old.Publish(ctx, sourceSnapshot(time.Now().Add(-time.Minute)))
	require.NoError(t, err)
	require.NoError(t, client.PExpire(ctx, store.lock, 20*time.Millisecond).Err())
	require.Eventually(t, func() bool { n, e := client.Exists(ctx, store.lock).Result(); return e == nil && n == 0 }, time.Second, 5*time.Millisecond)
	next, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, next)
	latest, err := next.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	_, err = old.Publish(ctx, sourceSnapshot(time.Now()))
	require.ErrorIs(t, err, ErrSnapshotLeaseLost)
	require.ErrorIs(t, old.Renew(ctx, latest.Revision), ErrSnapshotLeaseLost)
	require.NoError(t, old.Release(ctx))
	require.NoError(t, next.Renew(ctx, latest.Revision))
	require.ErrorIs(t, next.Renew(ctx, previous.Revision), ErrSnapshotConflict)
	read, err := store.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, latest, read)
	require.NoError(t, client.Del(ctx, store.key).Err())
	require.ErrorIs(t, next.Renew(ctx, latest.Revision), ErrSnapshotConflict)
	require.NoError(t, next.Release(ctx))
}

func TestRedisSnapshotIntegrationRejectsInvalidAndOversizedValues(t *testing.T) {
	store, client := realSnapshotStore(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	valid, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	for _, change := range []func(*SnapshotEnvelope){
		func(e *SnapshotEnvelope) { e.SchemaVersion = 99 }, func(e *SnapshotEnvelope) { e.Dataset = "other" }, func(e *SnapshotEnvelope) { e.Policy = strings.Repeat("b", 64) },
		func(e *SnapshotEnvelope) { e.EffectiveVersion = strings.Repeat("b", 64) }, func(e *SnapshotEnvelope) { e.Users = nil }, func(e *SnapshotEnvelope) { e.SourceSucceededAt = time.Now().Add(time.Hour) },
	} {
		invalid := *valid
		change(&invalid)
		body, marshalErr := json.Marshal(invalid)
		require.NoError(t, marshalErr)
		require.NoError(t, client.Set(ctx, store.key, body, time.Minute).Err())
		read, readErr := store.Read(ctx)
		require.Error(t, readErr)
		require.Nil(t, read)
	}
	body, err := json.Marshal(valid)
	require.NoError(t, err)
	for _, raw := range []string{"invalid", "[]", string(body) + " {}", strings.TrimSuffix(string(body), "}") + `,"unexpected":true}`} {
		require.NoError(t, client.Set(ctx, store.key, raw, time.Minute).Err())
		_, readErr := store.Read(ctx)
		require.Error(t, readErr)
	}
	oversized := "private-fixture-marker" + strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE)
	require.NoError(t, client.Set(ctx, store.key, oversized, time.Minute).Err())
	read, err := store.Read(ctx)
	require.Error(t, err)
	require.Nil(t, read)
	require.NotContains(t, err.Error(), "private-fixture-marker")
	require.NoError(t, client.Del(ctx, store.key).Err())
	require.NoError(t, client.RPush(ctx, store.key, "wrong-type").Err())
	_, err = store.Read(ctx)
	require.Error(t, err)
}

func TestRedisSnapshotIntegrationScriptACLFailureAndCancellation(t *testing.T) {
	store, client := realSnapshotStore(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	published, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	require.NoError(t, lease.Release(ctx))
	before, err := client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	user := "snapshot-it-" + published.Revision
	require.NoError(t, client.Do(ctx, "ACL", "SETUSER", user, "reset", "on", ">integration-password", "~"+store.key, "~"+store.lock, "+ping", "+get", "+strlen", "+set", "+del", "+pexpire").Err())
	t.Cleanup(func() { require.NoError(t, client.Do(context.Background(), "ACL", "DELUSER", user).Err()) })
	restricted := redis.NewClient(&redis.Options{Addr: client.Options().Addr, Username: user, Password: "integration-password", MaxRetries: -1, ContextTimeoutEnabled: true})
	t.Cleanup(func() { require.NoError(t, restricted.Close()) })
	denied, err := NewRedisSnapshotStore(restricted, store.dataset, store.policy)
	require.NoError(t, err)
	_, err = denied.Read(ctx)
	require.Error(t, err)
	writer, err := denied.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, writer)
	_, err = writer.Publish(ctx, sourceSnapshot(time.Now()))
	require.Error(t, err)
	require.Error(t, writer.Renew(ctx, published.Revision))
	require.Error(t, writer.Release(ctx))
	after, err := client.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.Equal(t, before, after)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = store.Read(canceled)
	require.True(t, errors.Is(err, context.Canceled), "canceled reads must report cancellation")
}
