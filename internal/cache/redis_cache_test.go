package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeRedisClient returns a client backed by miniredis.
//
// This used to be a hand-rolled RESP server understanding SET/GET/INCR and a
// handful more. cache-kit v1.6.0 moved Set and Clear onto MULTI/EXEC and reads
// the value through a Lua script that enforces MaxValueBytes, and emulating
// those by hand -- Lua in particular -- would go stale the next time the kit
// changes its commands. miniredis runs them for real.
func newFakeRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close redis client: %v", err)
		}
	})
	return client
}

func TestRedisUserCache_BasicFlow(t *testing.T) {
	client := newFakeRedisClient(t)
	cache := NewRedisUserCache(client)

	exists, err := cache.Exists()
	require.NoError(t, err)
	assert.False(t, exists)

	version, err := cache.GetVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(0), version)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
		{Phone: "13900139000", Mail: "user2@example.com"},
	}
	require.NoError(t, cache.Set(users))

	exists, err = cache.Exists()
	require.NoError(t, err)
	assert.True(t, exists)

	got, err := cache.Get()
	require.NoError(t, err)
	assert.Equal(t, users, got)

	version, err = cache.GetVersion()
	require.NoError(t, err)
	assert.Greater(t, version, int64(0))

	require.NoError(t, cache.Clear())

	exists, err = cache.Exists()
	require.NoError(t, err)
	assert.False(t, exists)

	got, err = cache.Get()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRedisUserCache_GetInvalidJSON(t *testing.T) {
	client := newFakeRedisClient(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, REDIS_CACHE_KEY, "invalid-json", REDIS_CACHE_TTL).Err())

	cache := NewRedisUserCache(client)
	users, err := cache.Get()
	assert.Error(t, err)
	assert.Nil(t, users)
}
