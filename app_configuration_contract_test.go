package main

import (
	"context"
	"github.com/soulteary/warden/internal/loader"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/remote"
	"github.com/stretchr/testify/require"
)

func TestDirectAppConfigurationFallbacks(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	t.Setenv("SNAPSHOT_MAX_AGE", "invalid")
	app := NewApp(&cmd.Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_LOCAL", Environment: "development", UserIDStrategy: "invalid", RedisSnapshotRecovery: "invalid", LegacyModeUsedForEnv: true})
	defer app.rateLimiter.Stop()
	require.Equal(t, "bootstrap", app.redisSnapshotRecovery)
	require.Equal(t, cmd.DefaultSnapshotMaxAge(5), app.snapshotMaxAge)
	require.Equal(t, define.UserIDStrategyLegacy, define.GetUserIDStrategy())
	remote.LegacyEncryptionObserver()
	require.Error(t, app.applyUsers([]define.AllowListUser{{Mail: "same@example.com", UserID: "one"}, {Mail: "same@example.com", UserID: "two"}}))
	app.rulesLoader = nil
	app.backgroundTask("", "")
}
func TestHealthCheckerHandlesUnknownViewAndLocalMode(t *testing.T) {
	store := newSnapshotStore()
	result := setupHealthChecker(nil, cache.NewSafeUserCache(), store, time.Minute, "ONLY_LOCAL", "development", false, false, "127.0.0.1, , ::1", false).Check(context.Background())
	require.NotNil(t, result)
	app := &App{log: logger.GetLoggerKit(), snapshots: store}
	app.publishToRedis = func([]define.AllowListUser) error { return context.Canceled }
	app.updateRedisIfWriter(true, nil)
}

func TestInitialLoadRejectsIdentityConflict(t *testing.T) {
	app, _ := migrationFixture(t, "legacy")
	app.appMode = "ONLY_REMOTE"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, writeErr := w.Write([]byte(`[{"mail":"a@example.com","user_id":"one"},{"mail":"b@example.com","user_id":"one"}]`))
		require.NoError(t, writeErr)
	}))
	defer server.Close()
	app.configURL = server.URL
	var err error
	app.rulesLoader, err = newTestRulesLoader(nil, "ONLY_REMOTE")
	require.NoError(t, err)
	require.NoError(t, app.loadInitialData("", ""))
	require.Zero(t, app.userCache.Len())
}
func TestInitialLocalLoadRejectsConflictingIDs(t *testing.T) {
	app, _ := migrationFixture(t, "legacy")
	app.appMode = "ONLY_LOCAL"
	var err error
	app.rulesLoader, err = newTestRulesLoader(nil, "ONLY_LOCAL")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "users.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"mail":"a@example.com","user_id":"one"},{"mail":"b@example.com","user_id":"one"}]`), 0o600))
	require.Error(t, app.loadInitialData(path, ""))
	require.Zero(t, app.userCache.Len())
}
func TestRefreshLockReleaseAndSeedFailures(t *testing.T) {
	app, _ := migrationFixture(t, "legacy")
	app.publishToRedis = func([]define.AllowListUser) error { return context.Canceled }
	app.seedRedisIfWriter(true, nil)
	app.publishToRedis = nil
	app.seedRedisIfWriter(true, nil)
	app.redisRefreshLocker = &contractRefreshLocker{}
	ok, release := app.acquireRedisRefreshWriter()
	require.True(t, ok)
	release()
	app.redisRefreshLocker = &contractRefreshLocker{fail: true}
	ok, release = app.acquireRedisRefreshWriter()
	require.False(t, ok)
	release()
}

type contractRefreshLocker struct{ fail bool }

func (l *contractRefreshLocker) Lock(string) (bool, error) {
	if l.fail {
		return false, context.Canceled
	}
	return true, nil
}
func (*contractRefreshLocker) Unlock(string) error { return context.Canceled }

func TestHealthSnapshotDiagnosticBoundaries(t *testing.T) {
	c := cache.NewSafeUserCache()
	c.Set([]define.AllowListUser{{Mail: "member@example.com"}})
	for _, tc := range []struct {
		name    string
		maxAge  time.Duration
		failure bool
	}{
		{"future source", time.Minute, false}, {"missing refresh reason", time.Minute, true}, {"disabled age", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newSnapshotStore()
			store.Store(&Snapshot{Source: loader.SourceLocal, LoadedAt: time.Now().Add(time.Minute), Count: 1, Degraded: true, view: c.ReadView()})
			if tc.failure {
				store.RecordRefreshFailure("")
			}
			result := setupHealthChecker(nil, c, store, tc.maxAge, "ONLY_LOCAL", "development", false, false, "", false).Check(context.Background())
			require.Equal(t, "unknown", result.Checks["snapshot"].Metadata["reason"])
			require.EqualValues(t, 0, result.Checks["snapshot"].Metadata["age_seconds"])
		})
	}
}
