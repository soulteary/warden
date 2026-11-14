package middleware

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// The Redis guard is shared by every client of the same Redis, remembers a
// nonce for the given time, and falls back to one minute when given none.
func TestRedisReplayGuardAgainstRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	newGuard := func() ReplayGuard {
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { require.NoError(t, client.Close()) })
		return NewRedisReplayGuard(client)
	}
	replicaA, replicaB := newGuard(), newGuard()

	require.False(t, replaySeen(t, replicaA, "gateway:nonce-1", 30*time.Second))
	require.True(t, replaySeen(t, replicaB, "gateway:nonce-1", 30*time.Second), "a second replica must see the nonce")
	mr.FastForward(31 * time.Second)
	require.False(t, replaySeen(t, replicaB, "gateway:nonce-1", 30*time.Second), "the nonce is forgotten after its time")

	// The stored key does not contain the key ID or the nonce.
	for _, key := range mr.Keys() {
		require.NotContains(t, key, "gateway")
		require.NotContains(t, key, "nonce-1")
	}

	require.False(t, replaySeen(t, replicaA, "gateway:nonce-2", 0))
	mr.FastForward(59 * time.Second)
	require.True(t, replaySeen(t, replicaA, "gateway:nonce-2", 0))
	mr.FastForward(2 * time.Second)
	require.False(t, replaySeen(t, replicaA, "gateway:nonce-2", 0))
}

// The in-memory guard applies the same one-minute fallback.
func TestMemoryReplayGuardDefaultTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	guard := NewMemoryReplayGuard(0)
	guard.nowFn = func() time.Time { return now }

	require.False(t, replaySeen(t, guard, "gateway:nonce", 0))
	now = now.Add(59 * time.Second)
	require.True(t, replaySeen(t, guard, "gateway:nonce", -time.Second))
	now = now.Add(2 * time.Minute)
	require.False(t, replaySeen(t, guard, "gateway:nonce", 0))
}
