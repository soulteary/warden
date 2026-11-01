package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	health "github.com/soulteary/health-kit/v4"
	rediskitclient "github.com/soulteary/redis-kit/client"
	rediskitlock "github.com/soulteary/redis-kit/lock"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestRedisBackendRetainsDependenciesAfterInitialFailure(t *testing.T) {
	mr := miniredis.RunT(t)
	var available atomic.Bool
	cfg := rediskitclient.DefaultConfig().WithAddr(mr.Addr()).WithMinIdleConns(0).WithMaxRetries(-1).WithDialTimeout(100 * time.Millisecond)
	cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !available.Load() {
			return nil, errors.New("test Redis unavailable")
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	client, err := newRecoverableRedisClient(&cfg)
	require.Error(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	app := &App{redisClient: client, redisEnabled: true, redisUserCache: cache.NewRedisUserCache(client), redisRefreshLocker: rediskitlock.NewRedisLockerWithLockTime(client, redisRefreshLockTTL), userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: logger.GetLoggerKit()}
	app.initRedisSnapshots(&cmd.Config{RedisSnapshotMode: "v2", RedisSnapshotNamespace: "recovery-test"})
	require.NotNil(t, app.redisSnapshots)
	originalStore := app.redisSnapshots
	// Create a health checker and Redis replay guard before the backend recovers.
	checker := setupHealthChecker(client, app.userCache, app.snapshots, time.Minute, "DEFAULT", "development", true, true, "")
	require.Equal(t, health.StatusUnhealthy, checker.Check(context.Background()).Checks["redis"].Status)
	guard := middleware.NewRedisReplayGuard(client)
	_, guardErr := guard.SeenBefore("recovery:before", time.Minute)
	require.Error(t, guardErr)
	available.Store(true)
	require.NoError(t, client.Ping(context.Background()).Err())
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Checks["redis"].Status)
	require.Same(t, originalStore, app.redisSnapshots)
	seen, guardErr := guard.SeenBefore("recovery:after", time.Minute)
	require.NoError(t, guardErr)
	require.False(t, seen)
	seen, guardErr = guard.SeenBefore("recovery:after", time.Minute)
	require.NoError(t, guardErr)
	require.True(t, seen)
	snap := publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	require.NotEmpty(t, snap.RedisRevision)
	app.snapshots = newSnapshotStore()
	app.userCache = cache.NewSafeUserCache()
	require.True(t, app.bootstrapFromRedis())
	require.Equal(t, snap.SourceSucceededAt, app.snapshots.Load().SourceSucceededAt)
	// Legacy cache and locker also share the original client and recover without replacement.
	require.NoError(t, app.redisUserCache.Set(app.userCache.Get()))
	got, err := app.redisUserCache.Get()
	require.NoError(t, err)
	require.Len(t, got, 1)
	app.redisSnapshotMode = "legacy"
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	release()
}

func TestRedisBackendFallbackRetainsConnectionOptions(t *testing.T) {
	cfg := rediskitclient.DefaultConfig().WithAddr("test.invalid:6379").WithMinIdleConns(0).WithMaxRetries(-1).WithDialTimeout(50 * time.Millisecond)
	cfg.Username = "service"
	cfg.Password = "test-password"
	cfg.DB = 2
	cfg.PoolSize = 7
	cfg.ContextTimeoutEnabled = true
	cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "redis.example"}
	cfg.Dialer = func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("offline") }
	client, err := newRecoverableRedisClient(&cfg)
	require.Error(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	opts := client.Options()
	require.Equal(t, cfg.Username, opts.Username)
	require.Equal(t, cfg.Password, opts.Password)
	require.Equal(t, cfg.DB, opts.DB)
	require.Equal(t, cfg.PoolSize, opts.PoolSize)
	require.Equal(t, cfg.DialTimeout, opts.DialTimeout)
	require.Equal(t, cfg.ReadTimeout, opts.ReadTimeout)
	require.Equal(t, cfg.WriteTimeout, opts.WriteTimeout)
	require.Equal(t, cfg.PoolTimeout, opts.PoolTimeout)
	require.True(t, opts.ContextTimeoutEnabled)
	require.NotSame(t, cfg.TLSConfig, opts.TLSConfig)
	require.Equal(t, cfg.TLSConfig.ServerName, opts.TLSConfig.ServerName)
}

func TestRedisBackendStartupSkipsRepeatedRedisAttempts(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	app.redisStartupFailed = true
	require.False(t, app.bootstrapFromRedis())
	writer, release := app.acquireRedisRefreshWriter()
	require.False(t, writer)
	release()
	app.redisStartupFailed = false
	writer, release = app.acquireRedisRefreshWriter()
	require.True(t, writer)
	release()
}

func TestRedisBackendRejectsEmptyAddress(t *testing.T) {
	client, err := newRecoverableRedisClient(&rediskitclient.Config{})
	require.Error(t, err)
	require.Nil(t, client)
}

func TestRedisBackendNewAppRecoversWithoutReinitialization(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	file := filepath.Join(t.TempDir(), "users.json")
	require.NoError(t, os.WriteFile(file, []byte(`[{"mail":"member@example.com","user_id":"member"}]`), 0o600))
	app := NewApp(&cmd.Config{Port: "8081", Redis: addr, RedisEnabled: true, Mode: "ONLY_LOCAL", Environment: "development", TaskInterval: 60, DataFile: file, RedisSnapshotMode: "v2", RedisSnapshotNamespace: "startup-recovery"})
	require.NotNil(t, app.redisClient)
	require.NotNil(t, app.redisUserCache)
	require.NotNil(t, app.redisSnapshots)
	t.Cleanup(func() { require.NoError(t, app.redisClient.Close()); app.rateLimiter.Stop() })
	require.False(t, app.redisStartupFailed)
	require.Len(t, app.userCache.Get(), 1, "initial Redis failure must not prevent source loading")
	originalClient := app.redisClient
	originalStore := app.redisSnapshots
	require.NoError(t, mr.Restart())
	require.NoError(t, app.redisClient.Ping(context.Background()).Err())
	app.backgroundTask(file, "")
	require.Same(t, originalClient, app.redisClient)
	require.Same(t, originalStore, app.redisSnapshots)
	envelope, err := app.redisSnapshots.Read(context.Background())
	require.NoError(t, err)
	require.NotNil(t, envelope)
	require.Equal(t, app.snapshots.Load().SourceSucceededAt, envelope.SourceSucceededAt)
	require.Equal(t, "local", envelope.Origin)
}
