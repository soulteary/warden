package main

import (
	"context"
	health "github.com/soulteary/health-kit/v4"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSnapshotReadDecisionReasons(t *testing.T) {
	app := responseMetadataApp(t)
	pinned := captureHealthState(app.snapshots, app.userCache)
	now := pinned.observedAt
	for _, tc := range []struct {
		change       func(*healthReadState)
		name, reason string
	}{
		{name: "fresh", reason: readAllowed, change: func(p *healthReadState) {}},
		{name: "no_snapshot", reason: readNoSnapshot, change: func(p *healthReadState) { p.state.snapshot = nil }},
		{name: "view_unknown", reason: readViewUnknown, change: func(p *healthReadState) { p.state.snapshot.view = nil }},
		{name: "view_mismatch", reason: readViewUnknown, change: func(p *healthReadState) { p.view = cache.NewSafeUserCache().ReadView() }},
		{name: "source_unknown", reason: readSourceUnknown, change: func(p *healthReadState) { p.state.snapshot.SourceSucceededAt = time.Time{} }},
		{name: "future", reason: readSourceFuture, change: func(p *healthReadState) { p.state.snapshot.SourceSucceededAt = now.Add(time.Nanosecond) }},
		{name: "deadline", reason: readSourceExpired, change: func(p *healthReadState) { p.state.snapshot.SourceSucceededAt = now.Add(-app.snapshotMaxAge) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := *app.snapshots.Load()
			p := *pinned
			p.state = &snapshotState{snapshot: &snap}
			tc.change(&p)
			decision := evaluateSnapshotRead(&p, app.snapshotMaxAge)
			require.Equal(t, tc.reason, decision.reason)
			require.Equal(t, tc.reason == readAllowed, snapshotReadFresh(&p, app.snapshotMaxAge))
		})
	}
	require.Equal(t, readNoSnapshot, evaluateSnapshotRead(nil, time.Minute).reason)
	require.Equal(t, readInvalidMaxAge, evaluateSnapshotRead(pinned, 0).reason)
}

func TestReadReadinessAcrossModesAndHealthAliases(t *testing.T) {
	modes := []string{"DEFAULT", "ONLY_LOCAL", "ONLY_REMOTE", "REMOTE_FIRST", "LOCAL_FIRST", "REMOTE_FIRST_ALLOW_REMOTE_FAILED", "LOCAL_FIRST_ALLOW_REMOTE_FAILED"}
	for _, mode := range modes {
		for _, state := range []string{"fresh", "expired", "unknown", "legacy", "fallback", "future", "empty", "no_snapshot", "legacy_old_source_fresh"} {
			t.Run(mode+"/"+state, func(t *testing.T) {
				app := responseMetadataApp(t)
				app.appMode = mode
				app.snapshotReadEnforce = true
				snap := *app.snapshots.Load()
				switch state {
				case "expired":
					snap.SourceSucceededAt = time.Now().Add(-time.Hour)
				case "unknown":
					snap.SourceSucceededAt = time.Time{}
				case "legacy":
					snap = *snapshotFromRedis(app.userCache.Get(), app.userCache.ReadView())
				case "fallback":
					snap.view = nil
				case "future":
					snap.SourceSucceededAt = time.Now().Add(time.Hour)
				case "empty":
					_, empty := preparedTestSnapshot(t, []define.AllowListUser{})
					snap = *empty
					snap.SourceSucceededAt = time.Now().Add(-time.Second)
				case "legacy_old_source_fresh":
					snap.LoadedAt = time.Now().Add(-time.Hour)
				}
				app.snapshots.Store(&snap)
				if state == "no_snapshot" {
					app.snapshots = newSnapshotStore()
				}
				mux := registerRoutes(app)
				expected := http.StatusServiceUnavailable
				if state == "fresh" || state == "empty" || state == "legacy_old_source_fresh" {
					expected = http.StatusOK
				}
				for _, path := range []string{"/", "/health", "/healthcheck", "/v1/health", "/v1/healthcheck"} {
					req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
					req.Header.Set("X-API-Key", "key")
					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, req)
					require.Equal(t, expected, rec.Code, path+rec.Body.String())
				}
				checker := setupHealthChecker(nil, app.userCache, app.snapshots, app.snapshotMaxAge, mode, "development", false, false, "", true)
				result := checker.Check(context.Background())
				require.Equal(t, snapshotReadFresh(captureHealthState(app.snapshots, app.userCache), app.snapshotMaxAge), result.Checks["read_ready"].Status == health.StatusHealthy)
			})
		}
	}
}

func TestObserveDoesNotAddReadReadiness(t *testing.T) {
	app := responseMetadataApp(t)
	snap := *app.snapshots.Load()
	snap.SourceSucceededAt = time.Time{}
	app.snapshots.Store(&snap)
	checker := setupHealthChecker(nil, app.userCache, app.snapshots, app.snapshotMaxAge, "ONLY_REMOTE", "development", false, false, "", false)
	result := checker.Check(context.Background())
	require.NotContains(t, result.Checks, "read_ready")
	require.Equal(t, health.StatusHealthy, result.Status)
}

func TestReadReadinessPinsPublicationAndObservation(t *testing.T) {
	app := responseMetadataApp(t)
	pinned := captureHealthState(app.snapshots, app.userCache)
	snap := pinned.state.snapshot
	checker := setupHealthChecker(nil, app.userCache, app.snapshots, app.snapshotMaxAge, "DEFAULT", "development", false, false, "", true)
	next := *snap
	next.SourceSucceededAt = time.Now().Add(-time.Hour)
	app.snapshots.Store(&next)
	ctx := context.WithValue(context.Background(), healthReadKey{}, pinned)
	result := checker.Check(ctx)
	require.Equal(t, readAllowed, result.Checks["read_ready"].Metadata["reason"])
	require.Equal(t, snap.SourceSucceededAt.Add(app.snapshotMaxAge).UTC().Format(time.RFC3339Nano), result.Checks["read_ready"].Metadata["valid_until"])
	require.Equal(t, readSourceExpired, checker.Check(context.Background()).Checks["read_ready"].Metadata["reason"])
	// The exact deadline is shared by the data guard and health checker, without sleeps.
	pinned.observedAt = snap.SourceSucceededAt.Add(app.snapshotMaxAge)
	require.False(t, snapshotReadFresh(pinned, app.snapshotMaxAge))
	require.Equal(t, readSourceExpired, checker.Check(ctx).Checks["read_ready"].Metadata["reason"])
}

func TestReadReadinessProductionHidesDetails(t *testing.T) {
	app := responseMetadataApp(t)
	app.environment = "production"
	app.snapshotReadEnforce = true
	snap := *app.snapshots.Load()
	snap.SourceSucceededAt = time.Time{}
	app.snapshots.Store(&snap)
	rec := httptest.NewRecorder()
	registerRoutes(app).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.NotContains(t, rec.Body.String(), "read_ready")
	require.NotContains(t, rec.Body.String(), "source_unknown")
}

func TestReadReadinessRetainsOtherCriticalChecks(t *testing.T) {
	app := responseMetadataApp(t)
	checker := setupHealthChecker(nil, app.userCache, app.snapshots, app.snapshotMaxAge, "DEFAULT", "development", true, true, "", true)
	result := checker.Check(context.Background())
	require.Equal(t, health.StatusHealthy, result.Checks["read_ready"].Status)
	require.Equal(t, health.StatusUnhealthy, result.Checks["redis"].Status)
	require.Equal(t, health.StatusUnhealthy, result.Status)
	_, snap := preparedTestSnapshot(t, []define.AllowListUser{})
	snap.Count = 1 // an uninitialized/rejected effective set is not accepted empty input
	snap.SourceSucceededAt = time.Now().Add(-time.Second)
	app.snapshots.Store(snap)
	checker = setupHealthChecker(nil, app.userCache, app.snapshots, app.snapshotMaxAge, "DEFAULT", "development", false, false, "", true)
	result = checker.Check(context.Background())
	require.Equal(t, health.StatusHealthy, result.Checks["read_ready"].Status)
	require.Equal(t, health.StatusUnhealthy, result.Checks["data"].Status)
	require.Equal(t, health.StatusUnhealthy, result.Status)
}
