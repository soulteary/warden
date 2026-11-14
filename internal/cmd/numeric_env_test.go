package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func clearNumericEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CONFIG", "KEY", "MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG_FILE", "REDIS", "API_KEY", "DATA_DIR",
		"PORT", "INTERVAL", "HTTP_TIMEOUT", "HTTP_MAX_IDLE_CONNS", "WARDEN_HMAC_TIMESTAMP_TOLERANCE"} {
		t.Setenv(key, "")
	}
	t.Setenv("REDIS_ENABLED", "false")
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	os.Args = []string{"warden"}
}

// startupError loads and validates the configuration the way main does.
func startupError(t *testing.T) error {
	t.Helper()
	cfg, err := GetArgsWithError()
	if err != nil {
		return err
	}
	return ValidateConfig(cfg)
}

// A numeric variable that does not hold a usable number stops startup and
// names the variable. Each of these used to select the default silently.
func TestUnusableNumericEnvStopsStartup(t *testing.T) {
	for name, values := range map[string][]string{
		"PORT":                            {"abc", "0", "70000", "80.5"},
		"INTERVAL":                        {"abc", "0", "-5", "5s"},
		"HTTP_TIMEOUT":                    {"abc", "0", "-1", "500ms", "3601", "2h", "18446744074"},
		"HTTP_MAX_IDLE_CONNS":             {"abc", "-1"},
		"WARDEN_HMAC_TIMESTAMP_TOLERANCE": {"abc", "0", "6O"},
	} {
		for _, raw := range values {
			clearNumericEnv(t)
			t.Setenv(name, raw)
			err := startupError(t)
			require.Error(t, err, "%s=%s", name, raw)
			require.Contains(t, err.Error(), "Invalid "+name, "%s=%s", name, raw)
		}
	}
}

// Usable values are applied as before.
func TestNumericEnvIsApplied(t *testing.T) {
	clearNumericEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("INTERVAL", "15")
	t.Setenv("HTTP_TIMEOUT", "1m30s")
	t.Setenv("HTTP_MAX_IDLE_CONNS", "0")
	t.Setenv("WARDEN_HMAC_TIMESTAMP_TOLERANCE", "120")
	cfg, err := GetArgsWithError()
	require.NoError(t, err)
	require.NoError(t, ValidateConfig(cfg))
	require.Equal(t, "9090", cfg.Port)
	require.Equal(t, 15, cfg.TaskInterval)
	require.Equal(t, 90, cfg.HTTPTimeout)

	clearNumericEnv(t)
	t.Setenv("HTTP_TIMEOUT", " 3600 ")
	require.NoError(t, startupError(t))
	require.NoError(t, startupError(t), "unset variables select the defaults")
}

// The bound also holds for a timeout that comes from the flag or the
// configuration file.
func TestHTTPTimeoutIsBoundedFromEverySource(t *testing.T) {
	clearNumericEnv(t)
	base := Config{Port: "8081", TaskInterval: 5, Mode: "DEFAULT", Environment: "development"}
	for _, seconds := range []int{-1, MaxHTTPTimeoutSeconds + 1, 1 << 40} {
		cfg := base
		cfg.HTTPTimeout = seconds
		require.ErrorContains(t, ValidateConfig(&cfg), "Invalid HTTP timeout", "%d", seconds)
	}
	for _, seconds := range []int{0, 5, MaxHTTPTimeoutSeconds} {
		cfg := base
		cfg.HTTPTimeout = seconds
		require.NoError(t, ValidateConfig(&cfg), "%d", seconds)
	}
}

func TestParseHTTPTimeout(t *testing.T) {
	for raw, want := range map[string]time.Duration{"30": 30 * time.Second, "30s": 30 * time.Second, "1m30s": 90 * time.Second, "1h": time.Hour} {
		got, err := ParseHTTPTimeout(raw)
		require.NoError(t, err, raw)
		require.Equal(t, want, got, raw)
	}
	padded, err := ParseHTTPTimeout(" 5 ")
	require.NoError(t, err)
	require.Equal(t, 5*time.Second, padded, "surrounding space is ignored")
	for _, raw := range []string{"", "abc", "0", "999ms", "3601", "61m", "9223372036854775807"} {
		_, err := ParseHTTPTimeout(raw)
		require.Error(t, err, raw)
	}
}
