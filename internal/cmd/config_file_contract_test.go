package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFileSelectionAndIntervalContract(t *testing.T) {
	for _, tc := range []struct {
		name         string
		cliFile      bool
		cliInterval  bool
		wantInterval int
	}{
		{"environment_file_and_interval", false, false, 15},
		{"cli_file_overrides_environment_file", true, false, 15},
		{"cli_interval_overrides_environment", true, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldArgs := os.Args
			t.Cleanup(func() { os.Args = oldArgs })
			for _, key := range []string{"MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG", "PORT", "REDIS", "HTTP_TIMEOUT", "HTTP_MAX_IDLE_CONNS", "HTTP_INSECURE_TLS"} {
				t.Setenv(key, "")
			}
			t.Setenv("REDIS_ENABLED", "false")
			t.Setenv("INTERVAL", "15")
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte("app:\n  mode: ONLY_LOCAL\ntask:\n  interval: 9s\ntracing:\n  enabled: true\n  endpoint: http://localhost:4318\n"), 0o600))
			os.Args = []string{"warden"}
			if tc.cliFile {
				t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "not-selected.yaml"))
				os.Args = append(os.Args, "--config-file", path)
			} else {
				t.Setenv("CONFIG_FILE", path)
			}
			if tc.cliInterval {
				os.Args = append(os.Args, "--interval", "20")
			}
			cfg, err := GetArgsWithError()
			require.NoError(t, err)
			require.Equal(t, path, cfg.ConfigFile, "main uses this path to initialize tracing")
			require.Equal(t, tc.wantInterval, cfg.TaskInterval)
		})
	}
}
