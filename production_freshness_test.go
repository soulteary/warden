package main

import (
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProductionDefaultsEnforceLocalSourceFreshness(t *testing.T) {
	fixture := responseMetadataApp(t)
	app := NewApp(&cmd.Config{Environment: "production", Mode: "ONLY_LOCAL", Port: "8081", APIKey: "key", TaskInterval: 10})
	defer app.rateLimiter.Stop()
	require.True(t, app.snapshotReadEnforce)
	snapshot := *fixture.snapshots.Load()
	snapshot.Source = loader.SourceLocal
	snapshot.SourceSucceededAt = time.Now().Add(-time.Second)
	app.snapshots.Store(&snapshot)
	routes := registerRoutes(app)
	request := func() int {
		req := httptest.NewRequest(http.MethodGet, "/v1/users", http.NoBody)
		req.Header.Set("X-API-Key", "key")
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec.Code
	}
	require.Equal(t, http.StatusOK, request(), "fresh local source must remain usable")
	snapshot.SourceSucceededAt = time.Now().Add(-app.snapshotMaxAge - time.Second)
	app.snapshots.Store(&snapshot)
	require.Equal(t, http.StatusServiceUnavailable, request(), "expired local evidence must not be extended by publication")
}
