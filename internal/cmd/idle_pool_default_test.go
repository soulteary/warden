package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// The default of HTTP_MAX_IDLE_CONNS is the pool size the remote client uses.
func TestDefaultIdlePoolMatchesTheClient(t *testing.T) {
	for _, key := range []string{"CONFIG", "KEY", "MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG_FILE", "REDIS", "REDIS_ENABLED", "API_KEY", "DATA_DIR", "HTTP_TIMEOUT", "INTERVAL", "PORT"} {
		t.Setenv(key, "")
	}
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"warden"}
	t.Setenv("HTTP_MAX_IDLE_CONNS", "")
	cfg, err := GetArgsWithError()
	require.NoError(t, err)
	require.Equal(t, 20, cfg.HTTPMaxIdleConns)

	t.Setenv("HTTP_MAX_IDLE_CONNS", "64")
	cfg, err = GetArgsWithError()
	require.NoError(t, err)
	require.Equal(t, 64, cfg.HTTPMaxIdleConns)
}
