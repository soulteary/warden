package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWarningForAnIntervalBelowTheMinimum(t *testing.T) {
	for interval, want := range map[int]bool{1: true, 4: true, 5: false, 60: false, 0: false} {
		warnings := UnappliedSettingWarnings(&Config{TaskInterval: interval})
		if !want {
			require.Empty(t, warnings, "interval %d", interval)
			continue
		}
		require.Len(t, warnings, 1, "interval %d", interval)
		require.Contains(t, warnings[0], "below the minimum of 5 seconds")
	}
}

func TestWarningNamesUnappliedConfigurationKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: \"8081\"\n  read_timeout: 30s\nredis:\n  db: 3\nhttp:\n  timeout: 5s\n  max_retries: 9\n"), 0o600))

	warnings := UnappliedSettingWarnings(&Config{TaskInterval: 5, ConfigFile: path})
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "http.max_retries, redis.db, server.read_timeout")
	require.NotContains(t, warnings[0], "server.port")
	require.NotContains(t, warnings[0], "http.timeout")

	wired := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(wired, []byte("server:\n  port: \"8081\"\nhttp:\n  max_idle_conns: 5\n"), 0o600))
	require.Empty(t, UnappliedSettingWarnings(&Config{TaskInterval: 5, ConfigFile: wired}))
}

// The example configuration marks every key the warning would name.
func TestExampleConfigurationMarksUnappliedKeys(t *testing.T) {
	example := filepath.Join("..", "..", "config.example.yaml")
	require.Len(t, UnappliedSettingWarnings(&Config{TaskInterval: 5, ConfigFile: example}), 1)
	data, err := os.ReadFile(example) // #nosec G304 -- fixed path inside the repository
	require.NoError(t, err)
	marked := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "NOT APPLIED") {
			marked++
		}
	}
	require.Equal(t, 10, marked, "one mark per unapplied key")
}
