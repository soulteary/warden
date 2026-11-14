package main

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// refreshLockReplica builds the Redis side of an App the way startup does.
func refreshLockReplica(t *testing.T, addr string, httpTimeout time.Duration, snapshotMode string) *App {
	t.Helper()
	cfg := &cmd.Config{RedisEnabled: true, Redis: addr, RedisSnapshotMode: snapshotMode, RedisSnapshotNamespace: "lock-test"}
	app := &App{httpTimeout: httpTimeout, log: logger.GetLoggerKit(), snapshots: newSnapshotStore()}
	app.initRedisClient(cfg)
	require.NotNil(t, app.redisClient)
	t.Cleanup(func() { _ = app.redisClient.Close() }) //nolint:errcheck // test cleanup
	app.initRedisSnapshots(cfg)
	return app
}

// The writer lock is taken before the sources are loaded so that an older,
// slower load cannot write to Redis after a newer one. That holds only while
// the lock outlives the longest load: with a fixed lifetime of one minute and
// a refresh deadline derived from HTTP_TIMEOUT, a second replica could take
// the lock while the first was still loading.
func TestRefreshLockOutlivesTheLongestRefresh(t *testing.T) {
	for _, mode := range []string{"legacy", "v2"} {
		for _, httpTimeout := range []time.Duration{5 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute} {
			mr := miniredis.RunT(t)
			slow := refreshLockReplica(t, mr.Addr(), httpTimeout, mode)
			other := refreshLockReplica(t, mr.Addr(), httpTimeout, mode)

			writer, release := slow.acquireRedisRefreshWriter()
			require.True(t, writer, "%s %s", mode, httpTimeout)

			// The longest a load may run: the first load has the larger deadline.
			longest := max(define.DEFAULT_LOAD_DATA_TIMEOUT, refreshDeadline(httpTimeout))
			mr.FastForward(longest)

			second, releaseSecond := other.acquireRedisRefreshWriter()
			releaseSecond()
			require.False(t, second, "%s, HTTP timeout %s: the lock expired while a load of %s was still allowed to run", mode, httpTimeout, longest)
			release()
		}
	}
}

func TestRefreshLockTTL(t *testing.T) {
	require.Equal(t, time.Minute, refreshLockTTL(0), "the default keeps the previous lifetime")
	require.Equal(t, time.Minute, refreshLockTTL(5*time.Second))
	require.Equal(t, 67*time.Second, refreshLockTTL(30*time.Second))
	require.Equal(t, 97*time.Second, refreshLockTTL(time.Minute))
}
