package main

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
)

func TestRedisHoldsForeignKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck // test cleanup
	ctx := context.Background()

	shared, err := redisHoldsForeignKeys(ctx, client)
	require.NoError(t, err)
	require.False(t, shared, "an empty database")

	for _, key := range []string{"warden:users:cache", "warden:users:cache:version", "warden:{prod}:snapshot:v2", "warden:hmac:replay:abc", "lock:warden:rules:refresh"} {
		require.NoError(t, mr.Set(key, "1"))
	}
	shared, err = redisHoldsForeignKeys(ctx, client)
	require.NoError(t, err)
	require.False(t, shared, "only Warden's own keys")

	require.NoError(t, mr.Set("herald:otp:13800138000", "123456"))
	shared, err = redisHoldsForeignKeys(ctx, client)
	require.NoError(t, err)
	require.True(t, shared)

	mr.Close()
	_, err = redisHoldsForeignKeys(ctx, client)
	require.Error(t, err)
}

// Startup runs the check against the connected instance and is not affected by
// its result.
func TestStartupChecksWhetherRedisIsShared(t *testing.T) {
	for _, key := range []string{"IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS", "REDIS_PASSWORD", "REDIS_PASSWORD_FILE"} {
		t.Setenv(key, "")
	}
	mr := miniredis.RunT(t)
	require.NoError(t, mr.Set("session:other-application", "1"))

	app := NewApp(&cmd.Config{Port: "8081", Mode: "ONLY_LOCAL", Environment: "production", TaskInterval: 60, APIKey: strings.Repeat("k", 32), Redis: mr.Addr(), RedisEnabled: true})
	t.Cleanup(app.rateLimiter.Stop)
	require.NotNil(t, app.redisClient)
	t.Cleanup(func() { _ = app.redisClient.Close() }) //nolint:errcheck // test cleanup

	shared, err := redisHoldsForeignKeys(context.Background(), app.redisClient)
	require.NoError(t, err)
	require.True(t, shared)

	// No client: nothing to check.
	(&App{}).warnIfRedisIsShared()
}
