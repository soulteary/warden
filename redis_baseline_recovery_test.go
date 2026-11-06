package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

func failedSourceRecoveryApp(t *testing.T) (app *App, dataFile string) {
	t.Helper()
	app, _ = migrationFixture(t, "v2")
	app.appMode = "ONLY_REMOTE"
	// This local endpoint fails immediately; no real upstream or DNS is needed.
	app.configURL = "http://127.0.0.1:1/unavailable"
	var err error
	app.rulesLoader, err = newTestRulesLoader(&cmd.Config{HTTPTimeout: 1}, app.appMode)
	require.NoError(t, err)
	return app, filepath.Join(t.TempDir(), "missing.json")
}

func TestV2SourceFailureRecoversBaselineWithoutResettingEvidence(t *testing.T) {
	for _, users := range [][]define.AllowListUser{{{Mail: "member@example.com", UserID: "member"}}, {}} {
		app, file := failedSourceRecoveryApp(t)
		published := publishMigrationSource(t, app, time.Now().Add(-10*time.Minute), users)
		app.snapshots = newSnapshotStore()
		app.userCache = cache.NewSafeUserCache()
		app.snapshots.RecordRefreshFailure("timeout")
		app.snapshotNow = func() time.Time { return time.Now().UTC() }
		app.backgroundTask(file, "")
		restored := app.snapshots.Load()
		require.True(t, app.hasKnownGoodSnapshot())
		require.Equal(t, published.SourceSucceededAt, restored.SourceSucceededAt)
		require.Equal(t, published.PublishedAt, restored.PublishedAt)
		require.Equal(t, published.RedisRevision, restored.RedisRevision)
		require.Equal(t, published.LoadedAt, restored.LoadedAt)
		require.Equal(t, published.EffectiveVersion, restored.view.GetHash())
		require.False(t, restored.RestoredAt.IsZero())
		failures, _ := app.snapshots.RefreshFailure()
		require.EqualValues(t, 2, failures)
	}
}

func TestV2RecoveryDoesNotReplaceKnownGoodIncludingEmpty(t *testing.T) {
	for _, users := range [][]define.AllowListUser{{{Mail: "local@example.com"}}, {}} {
		app, file := failedSourceRecoveryApp(t)
		original := publishMigrationSource(t, app, time.Now(), users)
		app.backgroundTask(file, "")
		require.Same(t, original, app.snapshots.Load())
		require.False(t, app.recoverVersionedBaseline())
	}
}

func TestV2RecoveryRejectsIncompatibleBaselineAndPreservesOtherModes(t *testing.T) {
	app, file := failedSourceRecoveryApp(t)
	published := publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	for _, mode := range []string{"legacy", "shadow"} {
		app.snapshots = newSnapshotStore()
		app.userCache = cache.NewSafeUserCache()
		app.redisSnapshotMode = mode
		app.backgroundTask(file, "")
		require.False(t, app.hasKnownGoodSnapshot())
	}
	app.redisSnapshotMode = "v2"
	app.appMode = "ONLY_LOCAL"
	require.False(t, app.recoverVersionedBaseline())
	app.appMode = "ONLY_REMOTE"
	var err error
	app.redisSnapshots, err = cache.NewRedisSnapshotStore(app.redisClient, "migration-test", strings.Repeat("b", 64))
	require.NoError(t, err)
	app.backgroundTask(file, "")
	require.False(t, app.hasKnownGoodSnapshot())
	// Failed restoration never rewrites the stored source evidence.
	store, err := cache.NewRedisSnapshotStore(app.redisClient, "migration-test", strings.Repeat("a", 64))
	require.NoError(t, err)
	got, err := store.Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, published.SourceSucceededAt, got.SourceSucceededAt)
}

type firstSnapshotReadFailure struct{ failed atomic.Bool }

func (h *firstSnapshotReadFailure) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *firstSnapshotReadFailure) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *firstSnapshotReadFailure) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// Read has one key and one size argument; publication has two keys.
		if cmd.Name() == "evalsha" && len(cmd.Args()) == 5 && h.failed.CompareAndSwap(false, true) {
			return errors.New("transient startup snapshot read failure")
		}
		return next(ctx, cmd)
	}
}

func TestV2StartupRetriesBaselineAfterSourceFailure(t *testing.T) {
	app, file := failedSourceRecoveryApp(t)
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "member@example.com"}})
	app.snapshots = newSnapshotStore()
	app.userCache = cache.NewSafeUserCache()
	hook := &firstSnapshotReadFailure{}
	app.redisClient.AddHook(hook)
	require.NoError(t, app.loadInitialData(file, ""))
	require.True(t, hook.failed.Load(), "the initial bootstrap actually failed")
	require.True(t, app.hasKnownGoodSnapshot(), "the source-error path retries the shared snapshot")
}

func TestV2IdentityFailureRecoversSharedBaseline(t *testing.T) {
	app, file := failedSourceRecoveryApp(t)
	publishMigrationSource(t, app, time.Now(), []define.AllowListUser{{Mail: "shared@example.com"}})
	app.snapshots = newSnapshotStore()
	app.userCache = cache.NewSafeUserCache()
	// DEFAULT with no remote URL loads the conflicting local source, while still
	// allowing the existing Redis-first recovery policy.
	app.appMode = "DEFAULT"
	app.configURL = ""
	var err error
	app.rulesLoader, err = newTestRulesLoader(&cmd.Config{HTTPTimeout: 1}, app.appMode)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, []byte(`[{"mail":"a@example.com","user_id":"same"},{"mail":"b@example.com","user_id":"same"}]`), 0o600))
	app.backgroundTask(file, "")
	require.True(t, app.hasKnownGoodSnapshot())
	failures, reason := app.snapshots.RefreshFailure()
	require.EqualValues(t, 1, failures)
	require.Equal(t, reasonIdentityConflict, reason)
	require.Equal(t, "shared@example.com", app.userCache.Get()[0].Mail)
}
