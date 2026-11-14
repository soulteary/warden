package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

// clearRemoteEnv removes every variable that selects a source or a mode, so a
// test starts from the built-in defaults.
func clearRemoteEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CONFIG", "KEY", "MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG_FILE", "REDIS", "REDIS_ENABLED", "API_KEY", "DATA_DIR", "HTTP_TIMEOUT", "INTERVAL", "PORT"} {
		t.Setenv(key, "")
	}
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	os.Args = []string{"warden"}
}

// A production deployment that configures only local data and an API key must
// start: it has no remote source, so the remote transport rules do not apply.
func TestProductionStartsWithoutARemoteSource(t *testing.T) {
	sources := map[string]func(t *testing.T){
		"environment": func(*testing.T) {},
		"config file": func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte("server:\n  port: \"8081\"\n"), 0o600))
			t.Setenv("CONFIG_FILE", path)
		},
	}
	for name, prepare := range sources {
		t.Run(name, func(t *testing.T) {
			clearRemoteEnv(t)
			t.Setenv("ENVIRONMENT", "production")
			t.Setenv("API_KEY", "an-api-key-for-the-test")
			t.Setenv("REDIS_ENABLED", "false")
			prepare(t)

			cfg, err := GetArgsWithError()
			require.NoError(t, err)
			require.Empty(t, cfg.RemoteConfig)
			require.Equal(t, define.DEFAULT_MODE, cfg.Mode)
			require.NoError(t, ValidateConfig(cfg))
		})
	}
}

// An explicitly configured remote is used as given, including the address that
// used to be the built-in default.
func TestExplicitRemoteSourceIsKept(t *testing.T) {
	clearRemoteEnv(t)
	t.Setenv("REDIS_ENABLED", "false")
	t.Setenv("CONFIG", define.DEFAULT_REMOTE_CONFIG)

	cfg, err := GetArgsWithError()
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/data.json", cfg.RemoteConfig)
	require.NoError(t, ValidateConfig(cfg), "development accepts the address as before")

	cfg.Environment = "production"
	cfg.APIKey = "an-api-key-for-the-test"
	require.ErrorContains(t, ValidateConfig(cfg), "requires HTTPS", "production still applies the remote transport rules")
}

// The modes that must read the remote source cannot run without one.
func TestStrictRemoteModesRequireARemoteSource(t *testing.T) {
	for _, mode := range []string{"ONLY_REMOTE", "REMOTE_FIRST", "only_remote"} {
		cfg := &Config{Port: "8081", TaskInterval: 5, Mode: mode, Environment: "development"}
		require.ErrorContains(t, ValidateConfig(cfg), "requires a remote source", mode)
	}
	for _, mode := range []string{"DEFAULT", "ONLY_LOCAL", "LOCAL_FIRST", "LOCAL_FIRST_ALLOW_REMOTE_FAILED", "REMOTE_FIRST_ALLOW_REMOTE_FAILED"} {
		cfg := &Config{Port: "8081", TaskInterval: 5, Mode: mode, Environment: "development"}
		require.NoError(t, ValidateConfig(cfg), mode)
	}
}
