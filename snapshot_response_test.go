package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/router"
	"github.com/stretchr/testify/require"
)

func responseMetadataApp(t *testing.T) *App {
	t.Helper()
	for _, key := range []string{"IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS"} {
		t.Setenv(key, "")
	}
	prepared, snap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "member@example.com", UserID: "member"}})
	snap.SourceSucceededAt = time.Now().Add(-time.Minute).UTC()
	users := cache.NewSafeUserCache()
	users.ApplyPrepared(prepared)
	snapshots := newSnapshotStore()
	snapshots.Store(snap)
	limiter := middlewarekit.NewRateLimiter(middlewarekit.DefaultRateLimiterConfig())
	t.Cleanup(limiter.Stop)
	return &App{userCache: users, snapshots: snapshots, snapshotMaxAge: 2 * time.Minute, appMode: "DEFAULT", environment: "development", apiKey: "key", rateLimiter: limiter, log: logger.GetLoggerKit()}
}

func TestSnapshotResponseMetadataNegotiationAndAliases(t *testing.T) {
	app := responseMetadataApp(t)
	mux := registerRoutes(app)
	for _, path := range []string{"/", "/data.json", "/v1/users", "/?page=1&page_size=1", "/user?user_id=member", "/v1/user?user_id=member", "/v1/lookup?identifier=member", "/user?user_id=missing"} {
		request := func(optIn, authenticated bool) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			if authenticated {
				req.Header.Set("X-API-Key", "key")
			}
			if optIn {
				req.Header.Set(snapshotMetadataHeader, "v1")
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			return rec
		}
		ordinary := request(false, true)
		enabled := request(true, true)
		require.Equal(t, ordinary.Code, enabled.Code, path)
		require.Equal(t, ordinary.Body.String(), enabled.Body.String(), path)
		require.Empty(t, ordinary.Header().Get(snapshotMetadataHeader), path)
		require.Contains(t, ordinary.Header().Values("Vary"), snapshotMetadataHeader, path)
		require.Contains(t, enabled.Header().Values("Vary"), snapshotMetadataHeader, path)
		require.Equal(t, "v1", enabled.Header().Get(snapshotMetadataHeader), path)
		require.Equal(t, app.snapshots.Load().view.GetHash(), enabled.Header().Get(snapshotVersionHeader), path)
		require.Equal(t, app.snapshots.Load().SourceSucceededAt.Format(time.RFC3339Nano), enabled.Header().Get(sourceSucceededHeader), path)
		denied := request(true, false)
		require.Equal(t, http.StatusUnauthorized, denied.Code)
		require.Empty(t, denied.Header().Get(snapshotVersionHeader))
	}
}

func TestSnapshotResponseHeadersAndRowsUseOneGeneration(t *testing.T) {
	app := responseMetadataApp(t)
	old := app.snapshots.Load()
	nextPrepared, next := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "new@example.com", UserID: "new"}})
	next.SourceSucceededAt = time.Now().UTC()
	handler := app.withSnapshotMetadata(func(source cache.ViewSource) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			app.userCache.ApplyPrepared(nextPrepared)
			app.snapshots.Store(next)
			router.JSON(source, nil)(w, r)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(snapshotMetadataHeader, "v1")
	rec := httptest.NewRecorder()
	handler(rec, req)
	var users []define.AllowListUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &users))
	require.Equal(t, "member", users[0].UserID)
	require.Equal(t, old.EffectiveVersion, rec.Header().Get(snapshotVersionHeader))
	require.Equal(t, old.SourceSucceededAt.Format(time.RFC3339Nano), rec.Header().Get(sourceSucceededHeader))
	require.Equal(t, "new", app.snapshots.Load().view.Get()[0].UserID)
}

func TestSnapshotResponseUnknownV1AgeIsNotInvented(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshots.Store(snapshotFromRedis(app.userCache.Get(), app.userCache.ReadView()))
	mux := registerRoutes(app)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-API-Key", "key")
	req.Header.Set(snapshotMetadataHeader, "v1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Header().Get(snapshotVersionHeader))
	require.Empty(t, rec.Header().Get(sourceSucceededHeader))
	require.Empty(t, rec.Header().Get(snapshotValidUntilHeader))
}

func TestPaginatedSnapshotPinsFreshnessVersionAndRows(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshotReadEnforce = true
	old := app.snapshots.Load()
	nextPrepared, next := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "new@example.com", UserID: "new"}})
	next.SourceSucceededAt = time.Now().Add(-time.Hour)
	handler := app.withSnapshotMetadata(func(source cache.ViewSource) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			app.userCache.ApplyPrepared(nextPrepared)
			app.snapshots.Store(next)
			router.JSON(source, nil)(w, r)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/?page=1&page_size=1", http.NoBody)
	req.Header.Set(snapshotMetadataHeader, "v1")
	rec := httptest.NewRecorder()
	handler(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var response struct {
		Data       []define.AllowListUser `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, 1, response.Pagination.Total)
	require.Len(t, response.Data, 1)
	require.Equal(t, "member", response.Data[0].UserID)
	require.Equal(t, old.EffectiveVersion, rec.Header().Get(snapshotVersionHeader))
	require.Equal(t, old.SourceSucceededAt.Format(time.RFC3339Nano), rec.Header().Get(sourceSucceededHeader))
	// A later request sees the replacement generation's expired source evidence.
	denied := httptest.NewRecorder()
	app.withSnapshotMetadata(func(source cache.ViewSource) func(http.ResponseWriter, *http.Request) {
		return router.JSON(source, nil)
	})(denied, req)
	require.Equal(t, http.StatusServiceUnavailable, denied.Code)
}
