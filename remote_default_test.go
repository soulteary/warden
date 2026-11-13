package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/loader"
)

// With no remote configured, the default mode serves the local file as a
// plain local source: nothing is fetched and the snapshot is not degraded.
func TestDefaultModeWithoutRemoteIsLocalOnly(t *testing.T) {
	for _, key := range []string{"CONFIG", "KEY", "MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG_FILE", "REDIS", "DATA_DIR", "HTTP_TIMEOUT", "INTERVAL", "PORT", "IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS"} {
		t.Setenv(key, "")
	}
	t.Setenv("REDIS_ENABLED", "false")
	t.Setenv("API_KEY", "key")
	dataFile := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(`[{"phone":"13800138000","mail":"admin@example.com","status":"active"}]`), 0o600))
	t.Setenv("DATA_FILE", dataFile)
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"warden"}

	cfg, err := cmd.GetArgsWithError()
	require.NoError(t, err)
	require.NoError(t, cmd.ValidateConfig(cfg))

	app := NewApp(cfg)
	t.Cleanup(app.rateLimiter.Stop)
	require.Empty(t, app.configURL)
	require.Equal(t, 1, app.userCache.Len())
	snap := app.snapshots.Load()
	require.Equal(t, loader.SourceLocal, snap.Source)
	require.False(t, snap.Degraded, "a deployment without a remote source is not running on a fallback")

	app.backgroundTask(app.dataFile, app.dataDir)
	failures, _ := app.snapshots.RefreshFailure()
	require.Zero(t, failures)
	require.False(t, app.snapshots.Load().Degraded)

	rec := serve(registerRoutes(app), apiKeyRequest("/health", ""), "127.0.0.1:1234")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "remote_failed")
}
