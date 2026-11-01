package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/router"
	"github.com/stretchr/testify/require"
)

func TestSnapshotStrictReadRoutes(t *testing.T) {
	for _, state := range []string{"fresh", "stale", "unknown", "legacy", "fallback", "future"} {
		for _, enforce := range []bool{false, true} {
			t.Run(state+map[bool]string{true: "/enforce", false: "/observe"}[enforce], func(t *testing.T) {
				app := responseMetadataApp(t)
				app.snapshotReadEnforce = enforce
				snap := *app.snapshots.Load()
				switch state {
				case "stale":
					snap.SourceSucceededAt = time.Now().Add(-3 * time.Minute)
				case "unknown":
					snap.SourceSucceededAt = time.Time{}
				case "legacy":
					snap = *snapshotFromRedis(app.userCache.Get(), app.userCache.ReadView())
				case "fallback":
					snap.view = nil
				case "future":
					snap.SourceSucceededAt = time.Now().Add(time.Hour)
				}
				app.snapshots.Store(&snap)
				mux := registerRoutes(app)
				for _, path := range []string{"/", "/data.json", "/v1/users", "/?page=1&page_size=1", "/user?user_id=member", "/v1/user?user_id=member", "/v1/lookup?identifier=member", "/user?user_id=missing", "/v1/user?user_id=missing", "/v1/lookup?identifier=missing"} {
					for _, negotiated := range []bool{false, true} {
						req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
						req.Header.Set("X-API-Key", "key")
						if negotiated {
							req.Header.Set(snapshotMetadataHeader, "v1")
						}
						rec := httptest.NewRecorder()
						mux.ServeHTTP(rec, req)
						expected := http.StatusOK
						if path == "/user?user_id=missing" || path == "/v1/user?user_id=missing" || path == "/v1/lookup?identifier=missing" {
							expected = http.StatusNotFound
						}
						if enforce && state != "fresh" {
							expected = http.StatusServiceUnavailable
						}
						require.Equal(t, expected, rec.Code, path)
						if enforce {
							require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
						}
					}
				}
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
				require.Equal(t, http.StatusUnauthorized, rec.Code)
				require.Empty(t, rec.Header().Get(snapshotVersionHeader))
			})
		}
	}
}

func TestSnapshotStrictReadDeadline(t *testing.T) {
	app := responseMetadataApp(t)
	pinned := captureHealthState(app.snapshots, app.userCache)
	source := pinned.state.snapshot.SourceSucceededAt
	for _, tc := range []struct {
		at    time.Time
		fresh bool
	}{
		{source.Add(-time.Nanosecond), false}, {source, true},
		{source.Add(app.snapshotMaxAge - time.Nanosecond), true},
		{source.Add(app.snapshotMaxAge), false}, {source.Add(app.snapshotMaxAge + time.Nanosecond), false},
	} {
		pinned.observedAt = tc.at
		require.Equal(t, tc.fresh, snapshotReadFresh(pinned, app.snapshotMaxAge))
	}
}

func TestSnapshotStrictReadPinsUnnegotiatedGeneration(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshotReadEnforce = true
	_, next := preparedTestSnapshot(t, []define.AllowListUser{{UserID: "new"}})
	next.SourceSucceededAt = time.Now().Add(-time.Hour)
	handler := app.withSnapshotMetadata(func(source cache.ViewSource) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			app.snapshots.Store(next)
			router.JSON(source, nil)(w, r)
		}
	})
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "member")
	require.NotContains(t, rec.Body.String(), "new")
	require.Empty(t, rec.Header().Get(snapshotMetadataHeader))
	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestSnapshotStrictReadFreshEmpty(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshotReadEnforce = true
	_, snap := preparedTestSnapshot(t, []define.AllowListUser{})
	snap.SourceSucceededAt = time.Now().Add(-time.Second)
	app.snapshots.Store(snap)
	mux := registerRoutes(app)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/", 200}, {"/user?user_id=missing", 404}, {"/v1/lookup?identifier=missing", 404}} {
		req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
		req.Header.Set("X-API-Key", "key")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code)
	}
}

func TestSnapshotStrictReadRestoredSourceAge(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	sourceTime := time.Now().Add(-time.Hour).UTC()
	publishMigrationSource(t, app, sourceTime, []define.AllowListUser{{UserID: "member", Mail: "member@example.com"}})
	app.snapshots = newSnapshotStore()
	app.userCache = cache.NewSafeUserCache()
	app.snapshotNow = func() time.Time { return time.Now().UTC() }
	require.True(t, app.bootstrapFromRedis())
	require.False(t, snapshotReadFresh(captureHealthState(app.snapshots, app.userCache), time.Minute))
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	require.NoError(t, app.publishRedisSnapshots(app.userCache.Get()))
	release()
	require.False(t, snapshotReadFresh(captureHealthState(app.snapshots, app.userCache), time.Minute))
	require.Equal(t, sourceTime, app.snapshots.Load().SourceSucceededAt)
	publishMigrationSource(t, app, time.Now().Add(-time.Second), []define.AllowListUser{{UserID: "member", Mail: "member@example.com"}})
	require.True(t, snapshotReadFresh(captureHealthState(app.snapshots, app.userCache), time.Minute))
}

func TestSnapshotReadPolicyAppConfiguration(t *testing.T) {
	for _, tc := range []struct {
		policy  string
		enforce bool
	}{{"", false}, {"observe", false}, {" ENFORCE ", true}, {"invalid", true}} {
		app := NewApp(&cmd.Config{SnapshotReadPolicy: tc.policy})
		require.Equal(t, tc.enforce, app.snapshotReadEnforce)
	}
}
