package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rediskitlock "github.com/soulteary/redis-kit/lock"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// Composition coverage only: use existing released native fixtures, storage,
// bootstrap and complete registered HTTP routes, not a reimplemented age model.
// Redis here is explicitly miniredis; source completion clock is a collaborator.
func TestUpgradeRecoveredSnapshotEnforcesNativeHTTP(t *testing.T) {
	users := []define.AllowListUser{{UserID: "member", Mail: "member@example.com"}}
	fixture := func(t *testing.T, mode string) *App {
		t.Helper()
		app, _ := migrationFixture(t, mode)
		base := responseMetadataApp(t)
		app.rateLimiter = base.rateLimiter
		app.apiKey = base.apiKey
		app.environment = "production"
		app.appMode = "DEFAULT"
		app.snapshotMaxAge = time.Minute
		policy, e := cmd.ParseSnapshotReadPolicy("", "production")
		require.NoError(t, e)
		require.Equal(t, "enforce", policy)
		app.snapshotReadEnforce = policy == "enforce"
		app.redisRefreshLocker = rediskitlock.NewRedisLockerWithLockTime(app.redisClient, redisRefreshLockTTL)
		app.loadFromRedis = app.readRedisUserCache
		app.redisCacheExists = app.readRedisUserCacheExists
		return app
	}
	request := func(t *testing.T, app *App, want int, reason string) {
		t.Helper()
		mux := registerRoutes(app)
		for _, path := range []string{"/v1/users", "/v1/user?user_id=member", "/v1/lookup?identifier=member"} {
			r := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			r.Header.Set("X-API-Key", "key")
			r.Header.Set(snapshotMetadataHeader, "v1")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.Equal(t, want, w.Code, path)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			require.Equal(t, reason, w.Header().Get(sourceErrorHeader))
			if want == http.StatusOK {
				require.Contains(t, w.Body.String(), "member")
				require.NotEmpty(t, w.Header().Get(sourceSucceededHeader))
				require.NotEmpty(t, w.Header().Get(snapshotValidUntilHeader))
			} else {
				require.NotContains(t, w.Body.String(), "member@example.com")
			}
		}
		denied := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/user?user_id=member", http.NoBody)
		r.Header.Set(snapshotMetadataHeader, "v1")
		mux.ServeHTTP(denied, r)
		require.Equal(t, http.StatusUnauthorized, denied.Code)
		require.Empty(t, denied.Header().Get(sourceSucceededHeader))
		require.Empty(t, denied.Header().Get(sourceErrorHeader))
	}
	for _, mode := range []string{"legacy", "shadow"} {
		t.Run(mode+"_restore_does_not_create_source_evidence", func(t *testing.T) {
			app := fixture(t, mode)
			require.NoError(t, app.redisUserCache.Set(users))
			require.True(t, app.bootstrapFromRedis())
			restored := app.snapshots.Load()
			require.True(t, restored.SourceSucceededAt.IsZero())
			require.False(t, restored.RestoredAt.IsZero())
			require.False(t, restored.LoadedAt.IsZero())
			request(t, app, http.StatusServiceUnavailable, "unknown")
			require.Equal(t, 1, app.userCache.Len(), "refusal must not pretend old rules were deleted")
			publishMigrationSource(t, app, time.Now().Add(-time.Second), users)
			request(t, app, http.StatusOK, "")
		})
	}
	t.Run("v2_restore_and_renew_do_not_extend_source_age", func(t *testing.T) {
		app := fixture(t, "v2")
		source := time.Now().Add(-10 * time.Minute).UTC()
		publishMigrationSource(t, app, source, users)
		app.snapshots = newSnapshotStore()
		app.userCache = cache.NewSafeUserCache()
		app.snapshotNow = func() time.Time { return time.Now().UTC() }
		require.True(t, app.bootstrapFromRedis())
		require.Equal(t, source, app.snapshots.Load().SourceSucceededAt)
		request(t, app, http.StatusServiceUnavailable, "expired")
		writer, release := app.acquireRedisRefreshWriter()
		require.True(t, writer)
		e := app.publishRedisSnapshots(app.userCache.Get())
		release()
		require.NoError(t, e)
		require.Equal(t, source, app.snapshots.Load().SourceSucceededAt)
		request(t, app, http.StatusServiceUnavailable, "expired")
		publishMigrationSource(t, app, time.Now().Add(-time.Second), users)
		request(t, app, http.StatusOK, "")
	})
	t.Run("v2_never_falls_back_to_legacy_unknown_metadata", func(t *testing.T) {
		app := fixture(t, "v2")
		require.NoError(t, app.redisUserCache.Set(users))
		require.False(t, app.bootstrapFromRedis())
		request(t, app, http.StatusServiceUnavailable, "unknown")
		old, e := app.redisUserCache.Get()
		require.NoError(t, e)
		require.Len(t, old, 1, "v2 refusal must retain legacy cache")
		publishMigrationSource(t, app, time.Now().Add(-time.Second), users)
		request(t, app, http.StatusOK, "")
	})
}
