package main

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func followLocal(t *testing.T, app *App, at time.Time) *Snapshot {
	t.Helper()
	prepared, snap := preparedTestSnapshot(t, []define.AllowListUser{{UserID: "old", Mail: "old@example.com"}})
	snap.SourceSucceededAt = at
	snap.LoadedAt = at
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.Store(snap)
	return app.snapshots.Load()
}

func TestRedisFollowRecoveryEligibility(t *testing.T) {
	for _, tc := range []struct {
		change func(*App)
		name   string
		want   bool
	}{
		{name: "enabled", change: func(a *App) {}, want: true},
		{name: "default", change: func(a *App) { a.redisSnapshotRecovery = "" }, want: false},
		{name: "bootstrap", change: func(a *App) { a.redisSnapshotRecovery = "bootstrap" }, want: false},
		{name: "local_only", change: func(a *App) { a.appMode = "ONLY_LOCAL" }, want: false},
		{name: "shadow", change: func(a *App) { a.redisSnapshotMode = "shadow" }, want: false},
		{name: "no_failure", change: func(a *App) { snap := a.snapshots.Load(); a.snapshots.Store(snap) }, want: false},
		{name: "local_fresh", change: func(a *App) {
			s := *a.snapshots.Load()
			s.SourceSucceededAt = time.Now().Add(-time.Second)
			a.snapshots.StorePreservingFailures(&s)
		}, want: false},
		{name: "local_unknown", change: func(a *App) {
			s := *a.snapshots.Load()
			s.SourceSucceededAt = time.Time{}
			a.snapshots.StorePreservingFailures(&s)
		}, want: false},
		{name: "local_future", change: func(a *App) {
			s := *a.snapshots.Load()
			s.SourceSucceededAt = time.Now().Add(time.Hour)
			a.snapshots.StorePreservingFailures(&s)
		}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			now := time.Now().UTC()
			published := publishMigrationSource(t, app, now.Add(-time.Second), []define.AllowListUser{{UserID: "new", Mail: "new@example.com"}})
			raw, err := mr.Get("warden:{migration-test}:snapshot:v2")
			require.NoError(t, err)
			followLocal(t, app, now.Add(-time.Hour))
			app.snapshotNow = func() time.Time { return now }
			app.snapshotMaxAge = time.Minute
			app.redisSnapshotRecovery = "stale"
			app.snapshots.RecordRefreshFailure("timeout")
			tc.change(app)
			before := app.snapshots.Load()
			require.Equal(t, tc.want, app.recoverVersionedBaseline())
			if tc.want {
				restored := app.snapshots.Load()
				require.Equal(t, "new", restored.view.Get()[0].UserID)
				require.Equal(t, published.SourceSucceededAt, restored.SourceSucceededAt)
				require.Equal(t, published.PublishedAt, restored.PublishedAt)
				require.Equal(t, published.RedisRevision, restored.RedisRevision)
				require.Equal(t, now, restored.RestoredAt)
				failures, reason := app.snapshots.RefreshFailure()
				require.EqualValues(t, 1, failures)
				require.Equal(t, "timeout", reason)
				require.Equal(t, "redis_follow_recovery", restored.DegradedReason)
			} else {
				require.Same(t, before, app.snapshots.Load())
			}
			after, err := mr.Get("warden:{migration-test}:snapshot:v2")
			require.NoError(t, err)
			require.Equal(t, raw, after)
		})
	}
}

func TestRedisFollowRejectsIneligibleCandidates(t *testing.T) {
	for _, state := range []string{"expired", "future", "same", "missing", "invalid", "policy"} {
		t.Run(state, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			now := time.Now().UTC()
			at := now.Add(-time.Second)
			switch state {
			case "expired":
				at = now.Add(-2 * time.Minute)
			case "same":
				at = now.Add(-time.Hour)
			case "future":
				at = now.Add(3 * time.Second)
			}
			publishMigrationSource(t, app, at, []define.AllowListUser{{UserID: "new", Mail: "new@example.com"}})
			key := "warden:{migration-test}:snapshot:v2"
			switch state {
			case "missing":
				mr.Del(key)
			case "invalid":
				require.NoError(t, mr.Set(key, "bad"))
			case "policy":
				raw, err := mr.Get(key)
				require.NoError(t, err)
				require.NoError(t, mr.Set(key, strings.Replace(raw, strings.Repeat("a", 64), strings.Repeat("b", 64), 1)))
			}
			old := followLocal(t, app, now.Add(-time.Hour))
			app.snapshotNow = func() time.Time { return now }
			app.snapshotMaxAge = time.Minute
			app.redisSnapshotRecovery = "stale"
			app.snapshots.RecordRefreshFailure("timeout")
			require.False(t, app.recoverVersionedBaseline())
			require.Same(t, old, app.snapshots.Load())
		})
	}
}

func TestRedisFollowConditionalPublication(t *testing.T) {
	app := responseMetadataApp(t)
	old := app.snapshots.Load()
	candidate := *old
	candidate.RestoredAt = time.Now()
	app.snapshots.RecordRefreshFailure("timeout")
	app.snapshots.RecordRefreshFailure("identity_conflict")
	require.True(t, app.snapshots.restoreIfCurrent(old, &candidate))
	failures, reason := app.snapshots.RefreshFailure()
	require.EqualValues(t, 2, failures)
	require.Equal(t, "identity_conflict", reason)
	newer := *old
	newer.SourceSucceededAt = time.Now()
	app.snapshots.Store(&newer)
	require.False(t, app.snapshots.restoreIfCurrent(old, &candidate))
	require.Equal(t, newer.SourceSucceededAt, app.snapshots.Load().SourceSucceededAt)
}

func TestRedisFollowBackgroundFailure(t *testing.T) {
	for _, content := range []string{"invalid json", rejectedRuleSet} {
		t.Run(content[:7], func(t *testing.T) {
			app, file, _ := policyTestApp(t, "availability-first")
			backing, mr := migrationFixture(t, "v2")
			app.appMode = "DEFAULT"
			var err error
			app.rulesLoader, err = loader.NewRulesLoader(&cmd.Config{}, "DEFAULT")
			require.NoError(t, err)
			app.redisSnapshotMode = "v2"
			app.redisSnapshots = backing.redisSnapshots
			app.redisSnapshotRecovery = "stale"
			app.snapshotMaxAge = time.Minute
			now := time.Now().UTC()
			publishMigrationSource(t, app, now.Add(-time.Second), []define.AllowListUser{})
			raw, err := mr.Get("warden:{migration-test}:snapshot:v2")
			require.NoError(t, err)
			followLocal(t, app, now.Add(-time.Hour))
			app.snapshotNow = func() time.Time { return now }
			writeRuleSet(t, file, content)
			app.backgroundTask(file, "")
			require.Empty(t, app.snapshots.Load().view.Get())
			failures, reason := app.snapshots.RefreshFailure()
			require.EqualValues(t, 1, failures)
			require.NotEmpty(t, reason)
			after, err := mr.Get("warden:{migration-test}:snapshot:v2")
			require.NoError(t, err)
			require.Equal(t, raw, after)
		})
	}
}

func TestRedisFollowRealRedis(t *testing.T) {
	addr := os.Getenv("WARDEN_REDIS_INTEGRATION_ADDR")
	if addr == "" {
		t.Skip("set WARDEN_REDIS_INTEGRATION_ADDR for a disposable Redis 8 server")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	namespace := "follow-" + strings.ReplaceAll(t.Name(), "_", "") + strconv.FormatInt(time.Now().UnixNano(), 10)
	store, err := cache.NewRedisSnapshotStore(client, namespace, strings.Repeat("a", 64))
	require.NoError(t, err)
	key := "warden:{" + namespace + "}:snapshot:v2"
	t.Cleanup(func() { require.NoError(t, client.Del(context.Background(), key, key+":lease").Err()) })
	newApp := func() *App {
		return &App{redisClient: client, redisSnapshots: store, redisSnapshotMode: "v2", redisSnapshotRecovery: "stale", snapshots: newSnapshotStore(), userCache: cache.NewSafeUserCache(), snapshotMaxAge: time.Minute, log: logger.GetLoggerKit()}
	}
	writer, replica := newApp(), newApp()
	now := time.Now().UTC()
	old := followLocal(t, replica, now.Add(-time.Hour))
	replica.snapshots.RecordRefreshFailure("timeout")
	published := publishMigrationSource(t, writer, now.Add(-time.Second), []define.AllowListUser{})
	raw, err := client.Get(context.Background(), key).Result()
	require.NoError(t, err)
	ttlBefore, err := client.PTTL(context.Background(), key).Result()
	require.NoError(t, err)
	require.True(t, replica.recoverVersionedBaseline())
	require.NotSame(t, old, replica.snapshots.Load())
	require.Empty(t, replica.snapshots.Load().view.Get())
	require.Equal(t, published.SourceSucceededAt, replica.snapshots.Load().SourceSucceededAt)
	require.Equal(t, published.PublishedAt, replica.snapshots.Load().PublishedAt)
	failures, reason := replica.snapshots.RefreshFailure()
	require.EqualValues(t, 1, failures)
	require.Equal(t, "timeout", reason)
	after, err := client.Get(context.Background(), key).Result()
	require.NoError(t, err)
	require.Equal(t, raw, after)
	ttlAfter, err := client.PTTL(context.Background(), key).Result()
	require.NoError(t, err)
	require.Positive(t, ttlAfter)
	require.LessOrEqual(t, ttlAfter, ttlBefore)
}

func TestRedisFollowRejectsCandidateThatExpiresDuringRecovery(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	now := time.Now().UTC()
	publishMigrationSource(t, app, now.Add(-time.Second), []define.AllowListUser{{UserID: "new", Mail: "new@example.com"}})
	old := followLocal(t, app, now.Add(-time.Hour))
	app.redisSnapshotRecovery = "stale"
	app.snapshotMaxAge = time.Minute
	app.snapshots.RecordRefreshFailure("timeout")
	calls := 0
	app.snapshotNow = func() time.Time {
		calls++
		if calls == 1 {
			return now
		}
		return now.Add(2 * time.Minute)
	}
	require.False(t, app.recoverVersionedBaseline())
	require.Same(t, old, app.snapshots.Load())
}

func TestRedisFollowSupersededAfterEligibility(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	now := time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)
	expected := followLocal(t, app, now.Add(-time.Hour))
	app.snapshots.RecordRefreshFailure("timeout")
	app.snapshotNow = func() time.Time { return now }
	app.snapshotMaxAge = time.Minute
	prepared, candidate := preparedTestSnapshot(t, []define.AllowListUser{{UserID: "redis", Mail: "redis@example.com"}})
	candidate.SourceSucceededAt = now.Add(-time.Second)
	candidate.LoadedAt = candidate.SourceSucceededAt
	require.True(t, app.eligibleRedisRecovery(expected, candidate))

	// Publish a successful source generation after eligibility but before adoption.
	newPrepared, newer := preparedTestSnapshot(t, []define.AllowListUser{{UserID: "source", Mail: "source@example.com"}})
	newer.SourceSucceededAt = now
	newer.LoadedAt = now
	app.userCache.ApplyPrepared(newPrepared)
	app.snapshots.Store(newer)
	app.snapshots.RecordRefreshFailure("identity_conflict")
	pinned := app.snapshots.capture()
	view := app.userCache.ReadView()
	require.False(t, app.adoptFollowSnapshot(expected, candidate, prepared))
	require.Same(t, pinned, app.snapshots.capture())
	require.Same(t, view, app.userCache.ReadView())
	require.Equal(t, "source", app.snapshots.Load().view.Get()[0].UserID)
	failures, reason := app.snapshots.RefreshFailure()
	require.EqualValues(t, 1, failures)
	require.Equal(t, "identity_conflict", reason)
}
