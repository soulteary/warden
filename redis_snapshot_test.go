package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	health "github.com/soulteary/health-kit/v4"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
)

func migrationFixture(t *testing.T, mode string) (*App, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store, err := cache.NewRedisSnapshotStore(client, "migration-test", strings.Repeat("a", 64))
	require.NoError(t, err)
	return &App{redisClient: client, redisSnapshotMode: mode, redisSnapshots: store, redisUserCache: cache.NewRedisUserCache(client), userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: logger.GetLoggerKit()}, mr
}

func publishMigrationSource(t *testing.T, app *App, at time.Time, users []define.AllowListUser) *Snapshot {
	t.Helper()
	prepared, err := app.prepareUsers(users)
	require.NoError(t, err)
	app.snapshotNow = func() time.Time { return at }
	res := &loader.LoadResult{Users: users, Version: cache.HashUserList(users), Source: loader.SourceRemote, LoadedAt: at.Add(-time.Second)}
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.Store(app.sourceSnapshot(res, prepared.ReadView()))
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	return app.snapshots.Load()
}

func TestV2RecoveryAndRenewalPreserveDataAge(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	sourceTime := time.Now().Add(-10 * time.Minute).UTC()
	users := []define.AllowListUser{{Mail: "member@example.com", UserID: "member"}}
	original := publishMigrationSource(t, app, sourceTime, users)
	require.False(t, original.PublishedAt.IsZero())
	require.Equal(t, sourceTime, original.LoadedAt)
	raw, err := mr.Get("warden:{migration-test}:snapshot:v2")
	require.NoError(t, err)
	app.snapshots = newSnapshotStore()
	app.userCache = cache.NewSafeUserCache()
	restoredAt := time.Now().UTC()
	app.snapshotNow = func() time.Time { return restoredAt }
	require.True(t, app.bootstrapFromRedis())
	restored := app.snapshots.Load()
	require.Equal(t, sourceTime, restored.SourceSucceededAt)
	require.Equal(t, sourceTime, restored.LoadedAt)
	require.Equal(t, original.PublishedAt, restored.PublishedAt)
	require.Equal(t, restoredAt, restored.RestoredAt)
	require.Equal(t, loader.SourceRemote, restored.Origin)
	require.Equal(t, original.EffectiveVersion, restored.view.GetHash())
	app.snapshots.RecordRefreshFailure("all_records_rejected")
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	release()
	renewed, err := mr.Get("warden:{migration-test}:snapshot:v2")
	require.NoError(t, err)
	require.Equal(t, raw, renewed)
	checker := setupHealthChecker(nil, app.userCache, app.snapshots, time.Minute, "ONLY_REMOTE", "development", false, false, "", false)
	result := checker.Check(context.Background())
	require.Equal(t, health.StatusUnhealthy, result.Status)
	require.Equal(t, sourceTime.Format(time.RFC3339Nano), result.Checks["snapshot"].Metadata["source_succeeded_at"])
	require.Len(t, app.userCache.Get(), 1, "stale health does not revoke cached users")
	refreshed := publishMigrationSource(t, app, restoredAt, users)
	require.Equal(t, restoredAt, refreshed.SourceSucceededAt, "identical successful source recheck advances source evidence")
	require.NotEqual(t, original.RedisRevision, refreshed.RedisRevision)
}

func TestV2RefusesLegacyAndPolicyMismatch(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	require.NoError(t, app.redisUserCache.Set([]define.AllowListUser{{Mail: "legacy@example.com"}}))
	require.False(t, app.bootstrapFromRedis(), "legacy has no source age")
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "fresh@example.com"}})
	var err error
	app.redisSnapshots, err = cache.NewRedisSnapshotStore(app.redisClient, "migration-test", strings.Repeat("b", 64))
	require.NoError(t, err)
	require.False(t, app.bootstrapFromRedis())
	require.Equal(t, "fresh@example.com", app.userCache.Get()[0].Mail)
}

func TestShadowReadsLegacyAndDoesNotInventSourceTime(t *testing.T) {
	app, mr := migrationFixture(t, "shadow")
	require.NoError(t, app.redisUserCache.Set([]define.AllowListUser{{Mail: "legacy@example.com"}}))
	app.loadFromRedis = app.readRedisUserCache
	app.redisCacheExists = app.readRedisUserCacheExists
	require.True(t, app.bootstrapFromRedis())
	require.True(t, app.snapshots.Load().SourceSucceededAt.IsZero())
	require.False(t, app.snapshots.Load().RestoredAt.IsZero())
	// A renewal of an unknown-age V1 baseline must not create a V2 publication.
	_, release := app.acquireSnapshotLease()
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	release()
	require.False(t, mr.Exists("warden:{migration-test}:snapshot:v2"))
	// Even if V2 exists, shadow bootstrap remains authoritative on V1.
	app.redisSnapshotMode = "v2"
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "v2@example.com"}})
	app.redisSnapshotMode = "shadow"
	require.True(t, app.bootstrapFromRedis())
	require.Equal(t, "legacy@example.com", app.userCache.Get()[0].Mail)
}

func TestV2RecoveryPreservesInputHealthAndEmptyRevocation(t *testing.T) {
	for _, users := range [][]define.AllowListUser{{}, {{Mail: "invalid", UserID: "rejected"}}} {
		app, _ := migrationFixture(t, "v2")
		original := publishMigrationSource(t, app, time.Now(), users)
		require.True(t, app.bootstrapFromRedis())
		snap := app.snapshots.Load()
		require.Equal(t, original.Count, snap.Count)
		require.Equal(t, original.Version, snap.Version)
		require.Zero(t, snap.EffectiveCount)
	}
}

func TestPublicationCannotReplaceNewerSnapshotOrFailures(t *testing.T) {
	store := newSnapshotStore()
	_, snap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "a@example.com"}})
	store.Store(snap)
	expected := store.Load()
	store.RecordRefreshFailure("timeout")
	store.recordPublication(expected, time.Now(), "revision")
	failures, reason := store.RefreshFailure()
	require.EqualValues(t, 1, failures)
	require.Equal(t, "timeout", reason)
	store.Store(snap)
	newer := store.Load()
	store.recordPublication(expected, time.Now(), "old")
	require.Same(t, newer, store.Load())
}

func TestSnapshotPolicyPartitionsValidationContracts(t *testing.T) {
	cfg := &cmd.Config{}
	baseline := snapshotPolicy(cfg)
	for _, change := range []func(*cmd.Config){
		func(c *cmd.Config) { c.Mode = "ONLY_REMOTE" },
		func(c *cmd.Config) { c.EmptyRulesetPolicy = "availability-first" },
		func(c *cmd.Config) { c.RemoteEncryptionRequired = true },
		func(c *cmd.Config) { c.RemoteDecryptEnabled = true; c.RemoteRSAPrivateKey = "configured" },
	} {
		modified := *cfg
		change(&modified)
		require.NotEqual(t, baseline, snapshotPolicy(&modified))
	}
}

func TestV2BackgroundRefreshSeparatesSuccessFromRetention(t *testing.T) {
	app, file, _ := policyTestApp(t, "availability-first")
	backing, _ := migrationFixture(t, "v2")
	app.redisSnapshotMode = "v2"
	app.redisSnapshots = backing.redisSnapshots
	app.publishToRedis = app.publishRedisSnapshots
	at := time.Now().Add(-time.Minute).UTC()
	app.snapshotNow = func() time.Time { return at }
	app.backgroundTask(file, "")
	first := app.snapshots.Load()
	require.Equal(t, at, first.SourceSucceededAt)
	require.NotEmpty(t, first.RedisRevision)
	writeRuleSet(t, file, rejectedRuleSet)
	at = at.Add(10 * time.Second)
	app.backgroundTask(file, "")
	retained := app.snapshots.Load()
	require.Equal(t, first.SourceSucceededAt, retained.SourceSucceededAt)
	require.Equal(t, first.PublishedAt, retained.PublishedAt)
	require.Equal(t, first.RedisRevision, retained.RedisRevision)
	writeRuleSet(t, file, "invalid json")
	app.backgroundTask(file, "")
	require.Equal(t, first.SourceSucceededAt, app.snapshots.Load().SourceSucceededAt)
	writeRuleSet(t, file, goodRuleSet)
	app.backgroundTask(file, "")
	recovered := app.snapshots.Load()
	require.Equal(t, at, recovered.SourceSucceededAt)
	require.NotEqual(t, first.RedisRevision, recovered.RedisRevision)
	failures, _ := app.snapshots.RefreshFailure()
	require.Zero(t, failures)
}

func TestShadowPublicationKeepsLegacyAuthorityAndReleasesBothLocks(t *testing.T) {
	app, mr := migrationFixture(t, "shadow")
	locker := &stubRefreshLocker{locked: true}
	app.redisRefreshLocker = locker
	users := []define.AllowListUser{{Mail: "shadow@example.com"}}
	published := publishMigrationSource(t, app, time.Now(), users)
	require.NotEmpty(t, published.RedisRevision)
	require.Equal(t, 1, locker.unlocks)
	require.False(t, mr.Exists("warden:{migration-test}:snapshot:v2:lease"))
	legacy, err := app.redisUserCache.Get()
	require.NoError(t, err)
	require.Equal(t, app.userCache.Get(), legacy)
	// An occupied V2 lease must not prevent the successful V1 writer from publishing.
	lease, err := app.redisSnapshots.Acquire(context.Background(), time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	require.Nil(t, app.snapshotLease)
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	release()
	require.NoError(t, lease.Release(context.Background()))
}
