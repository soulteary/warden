package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Startup validation reports a private-origin list that is not valid, and
// file settings whose path cannot be examined or read.
func TestValidateConfigReportsUnusableOriginsAndPaths(t *testing.T) {
	base := func() *Config {
		return &Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_LOCAL", Environment: "development"}
	}
	require.NoError(t, ValidateConfig(base()))

	t.Run("private origins", func(t *testing.T) {
		t.Setenv("REMOTE_PRIVATE_ORIGINS", "https://rules.internal/path")
		require.ErrorContains(t, ValidateConfig(base()), "REMOTE_PRIVATE_ORIGINS entries must be exact HTTP(S) origins")
	})

	// A path below a regular file exists neither as a file nor as "missing":
	// the operating system reports that a parent is not a directory.
	file := filepath.Join(t.TempDir(), "plain-file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))
	below := filepath.Join(file, "child")

	t.Run("data directory below a file", func(t *testing.T) {
		cfg := base()
		cfg.DataDir = below
		err := ValidateConfig(cfg)
		require.ErrorContains(t, err, "DATA_DIR")
		require.NotContains(t, err.Error(), "does not exist")
	})

	t.Run("key file below a file", func(t *testing.T) {
		cfg := base()
		cfg.RemoteDecryptEnabled = true
		cfg.RemoteRSAPrivateKeyFile = below
		err := ValidateConfig(cfg)
		require.ErrorContains(t, err, "REMOTE_RSA_PRIVATE_KEY_FILE")
		require.NotContains(t, err.Error(), "does not exist")
	})

	t.Run("key file without read permission", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission bits do not restrict the superuser")
		}
		key := filepath.Join(t.TempDir(), "key.pem")
		require.NoError(t, os.WriteFile(key, []byte("key"), 0o200))
		cfg := base()
		cfg.RemoteDecryptEnabled = true
		cfg.RemoteRSAPrivateKeyFile = key
		require.ErrorContains(t, ValidateConfig(cfg), "is not readable")
	})
}

// An unknown merge mode has no remote source to apply remote rules to; the
// mode itself is reported by its own check.
func TestUsesRemoteSource(t *testing.T) {
	for _, tc := range []struct {
		mode, remote string
		want         bool
	}{
		{"DEFAULT", "https://rules.example/data.json", true},
		{"ONLY_REMOTE", "https://rules.example/data.json", true},
		{"ONLY_LOCAL", "https://rules.example/data.json", false},
		{"DEFAULT", "  ", false},
		{"NOT_A_MODE", "https://rules.example/data.json", false},
	} {
		require.Equal(t, tc.want, usesRemoteSource(&Config{Mode: tc.mode, RemoteConfig: tc.remote}), "%s %q", tc.mode, tc.remote)
	}
}
