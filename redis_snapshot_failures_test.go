package main

import (
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestSnapshotInitializationRejectsInvalidContracts(t *testing.T) {
	for _, cfg := range []*cmd.Config{{RedisSnapshotMode: "invalid"}, {RedisSnapshotMode: "v2"}, {RedisSnapshotMode: "v2", RedisSnapshotNamespace: "bad namespace"}} {
		app, _ := migrationFixture(t, "v2")
		app.redisSnapshots = nil
		if cfg.RedisSnapshotNamespace == "" {
			app.redisClient = nil
		}
		app.initRedisSnapshots(cfg)
		require.Nil(t, app.redisSnapshots)
	}
	app := &App{redisSnapshotMode: "v2", log: logger.GetLoggerKit()}
	ok, release := app.acquireSnapshotLease()
	require.False(t, ok)
	release()
}
func TestSnapshotLeaseAndPublicationConnectionFailures(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	// A new source generation needs publication rather than renewal.
	prepared, err := app.prepareUsers([]define.AllowListUser{{Mail: "other@example.com"}})
	require.NoError(t, err)
	snapshot := *app.snapshots.Load()
	snapshot.RedisRevision = ""
	snapshot.view = prepared.ReadView()
	snapshot.EffectiveVersion = snapshot.view.GetHash()
	snapshot.Version = snapshot.EffectiveVersion
	app.snapshots.Store(&snapshot)
	ok, release := app.acquireRedisRefreshWriter()
	require.True(t, ok)
	mr.Close()
	require.Error(t, app.publishRedisSnapshots(nil))
	release()
	ok, release = app.acquireRedisRefreshWriter()
	require.False(t, ok)
	release()
}
func TestLegacySnapshotPublicationAndMissingBootstrapCallbacks(t *testing.T) {
	app, _ := migrationFixture(t, "legacy")
	require.NoError(t, app.publishRedisSnapshots(nil))
	require.False(t, app.bootstrapFromRedis())
	users, ok := app.resolveEmptyRedisRead()
	require.False(t, ok)
	require.Nil(t, users)
}
