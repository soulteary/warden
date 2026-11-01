package main

import (
	"context"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/prommetrics"
	"github.com/stretchr/testify/require"
)

func snapshotOutcomeCount(t *testing.T, mode, op, result string) float64 {
	t.Helper()
	metric := &dto.Metric{}
	require.NoError(t, prommetrics.RedisSnapshotOperationsTotal.WithLabelValues(mode, op, result).Write(metric))
	return metric.GetCounter().GetValue()
}

func TestSnapshotDiagnosticsSeparatePublishRenewAndSkippedWrites(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	before := snapshotOutcomeCount(t, "v2", "publish", "no_lease")
	require.NoError(t, app.publishRedisSnapshots(nil))
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "publish", "no_lease"))
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	before = snapshotOutcomeCount(t, "v2", "publish", "unknown_source_time")
	require.NoError(t, app.publishRedisSnapshots(nil))
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "publish", "unknown_source_time"))
	release()
	before = snapshotOutcomeCount(t, "v2", "publish", "published")
	published := publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "publish", "published"))
	before = snapshotOutcomeCount(t, "v2", "renew", "renewed")
	writer, release = app.acquireRedisRefreshWriter()
	require.True(t, writer)
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "renew", "renewed"))
	require.Equal(t, published.PublishedAt, app.snapshots.Load().PublishedAt)
	mr.Del("warden:{migration-test}:snapshot:v2")
	before = snapshotOutcomeCount(t, "v2", "renew", "missing")
	require.ErrorIs(t, app.publishRedisSnapshots(app.userCache.Get()), cache.ErrSnapshotConflict)
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "renew", "missing"))
	release()
}

func TestSnapshotDiagnosticsReadOutcomesAndLeaseContention(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	before := snapshotOutcomeCount(t, "v2", "read", "missing")
	require.False(t, app.bootstrapFromRedis())
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "read", "missing"))
	published := publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	before = snapshotOutcomeCount(t, "v2", "read", "restored")
	require.True(t, app.bootstrapFromRedis())
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "read", "restored"))
	require.Equal(t, published.SourceSucceededAt, app.snapshots.Load().SourceSucceededAt)
	require.NoError(t, mr.Set("warden:{migration-test}:snapshot:v2", "malformed"))
	before = snapshotOutcomeCount(t, "v2", "read", "invalid_envelope")
	require.False(t, app.bootstrapFromRedis())
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "read", "invalid_envelope"))
	external, err := app.redisSnapshots.Acquire(context.Background(), time.Minute)
	require.NoError(t, err)
	before = snapshotOutcomeCount(t, "v2", "lease_acquire", "busy")
	writer, release := app.acquireSnapshotLease()
	require.False(t, writer)
	release()
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "lease_acquire", "busy"))
	require.NoError(t, external.Release(context.Background()))
	app.redisSnapshots = nil
	before = snapshotOutcomeCount(t, "v2", "read", "unavailable")
	require.False(t, app.bootstrapFromRedis())
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "read", "unavailable"))
}

func TestSnapshotDiagnosticsEffectiveRejectionAndLegacySilence(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "invalid", UserID: "rejected"}})
	policy, ok := config.ParseEmptyRulesetPolicy("availability-first")
	require.True(t, ok)
	app.emptyRulesetPolicy = policy
	before := snapshotOutcomeCount(t, "v2", "read", "effective_rejected")
	require.False(t, app.bootstrapFromRedis())
	require.Equal(t, before+1, snapshotOutcomeCount(t, "v2", "read", "effective_rejected"))
	app.redisSnapshotMode = "legacy"
	before = snapshotOutcomeCount(t, "unknown", "read", "missing")
	app.recordSnapshotResult("read", "missing")
	require.Equal(t, before, snapshotOutcomeCount(t, "unknown", "read", "missing"))
}

func TestSnapshotDiagnosticsHealthVersionsDoNotInventV1InputEvidence(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	published := publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	require.True(t, app.bootstrapFromRedis())
	checker := setupHealthChecker(nil, app.userCache, app.snapshots, time.Minute, "DEFAULT", "development", false, false, "", false)
	metadata := checker.Check(context.Background()).Checks["snapshot"].Metadata
	require.Equal(t, published.Version, metadata["input_version"])
	require.Equal(t, published.EffectiveVersion, metadata["effective_version"])
	require.Equal(t, published.RedisRevision, metadata["redis_revision"])
	app.redisSnapshotMode = "shadow"
	app.loadFromRedis = app.readRedisUserCache
	app.redisCacheExists = app.readRedisUserCacheExists
	require.NoError(t, app.redisUserCache.Set(app.userCache.Get()))
	require.True(t, app.bootstrapFromRedis())
	metadata = checker.Check(context.Background()).Checks["snapshot"].Metadata
	require.NotContains(t, metadata, "input_version")
	require.Contains(t, metadata, "effective_version")
	require.Contains(t, metadata, "version")
}
