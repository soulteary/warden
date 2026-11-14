package main

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// dataKeyWrites counts SET commands on the shared rule-set key, whether sent
// alone or inside a transaction.
type dataKeyWrites struct{ sets atomic.Int64 }

func (d *dataKeyWrites) count(cmd redis.Cmder) {
	args := cmd.Args()
	if len(args) >= 2 && strings.EqualFold(cmd.Name(), "set") && args[1] == cache.REDIS_CACHE_KEY {
		d.sets.Add(1)
	}
}

func (d *dataKeyWrites) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return next(ctx, network, addr) }
}

func (d *dataKeyWrites) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		d.count(cmd)
		return next(ctx, cmd)
	}
}

func (d *dataKeyWrites) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			d.count(cmd)
		}
		return next(ctx, cmds)
	}
}

// Publishing a rule set that Redis already holds renews its TTL and does not
// send the list again. A missing or different value is written in full.
func TestUnchangedRuleSetIsNotRewrittenToRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck // test cleanup
	writes := &dataKeyWrites{}
	client.AddHook(writes)
	app := &App{redisClient: client, redisUserCache: cache.NewRedisUserCache(client), log: logger.GetLoggerKit()}
	users := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Status: "active", Scope: []string{}}}

	require.NoError(t, app.updateRedisCacheWithRetry(users))
	require.EqualValues(t, 1, writes.sets.Load(), "the first publication writes the list")
	for i := 0; i < 5; i++ {
		require.NoError(t, app.updateRedisCacheWithRetry(users))
	}
	require.EqualValues(t, 1, writes.sets.Load(), "an unchanged list is not written again")
	require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(cache.REDIS_CACHE_KEY))

	// Another writer replaced the value: the next publication restores it.
	require.NoError(t, mr.Set(cache.REDIS_CACHE_KEY, `[]`))
	require.NoError(t, app.updateRedisCacheWithRetry(users))
	require.EqualValues(t, 2, writes.sets.Load())
	stored, err := app.redisUserCache.Get()
	require.NoError(t, err)
	require.Equal(t, users, stored)

	// The key expired: written again.
	mr.Del(cache.REDIS_CACHE_KEY)
	require.NoError(t, app.updateRedisCacheWithRetry(users))
	require.EqualValues(t, 3, writes.sets.Load())

	// Changed data: written.
	users[0].Status = "suspended"
	require.NoError(t, app.updateRedisCacheWithRetry(users))
	require.EqualValues(t, 4, writes.sets.Load())
}
