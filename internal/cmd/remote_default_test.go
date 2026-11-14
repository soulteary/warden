package cmd

import (
	"os"
	"path/filepath"
	"strings"
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

// The address that used to be the built-in default is validated like any
// other. Startup validation skipped it, so the service started with a remote
// source that the connection policy refuses on every fetch.
func TestFormerDefaultRemoteIsValidatedLikeAnyOtherAddress(t *testing.T) {
	clearRemoteEnv(t)
	t.Setenv("REDIS_ENABLED", "false")
	t.Setenv("CONFIG", "http://localhost:8080/data.json")

	cfg, err := GetArgsWithError()
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/data.json", cfg.RemoteConfig, "an explicitly configured remote is used as given")

	formerDefault := ValidateConfig(cfg)
	require.ErrorContains(t, formerDefault, "Invalid remote configuration URL: http://localhost:8080/data.json")

	// The same refusal as for any other address on the local machine.
	cfg.RemoteConfig = "http://localhost:9999/data.json"
	otherPort := ValidateConfig(cfg)
	require.Error(t, otherPort)
	require.Equal(t, strings.ReplaceAll(otherPort.Error(), ":9999", ":8080"), formerDefault.Error())

	for _, mode := range []string{"ONLY_REMOTE", "REMOTE_FIRST"} {
		strict := &Config{Port: "8081", TaskInterval: 5, Mode: mode, Environment: "development", RemoteConfig: "http://localhost:8080/data.json"}
		require.ErrorContains(t, ValidateConfig(strict), "Invalid remote configuration URL", mode)
	}
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
