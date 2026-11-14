package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwiredKeysInFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	require.Nil(t, UnwiredKeysInFile(""))
	require.Nil(t, UnwiredKeysInFile(filepath.Join(dir, "missing.yaml")))
	require.Nil(t, UnwiredKeysInFile(write("broken.yaml", "server: [not a mapping")))
	require.Nil(t, UnwiredKeysInFile(write("wired.yaml", "server:\n  port: \"8081\"\ntask:\n  interval: 30s\n")))
	require.Equal(t,
		[]string{"cache.ttl", "cache.update_interval", "http.max_retries", "http.retry_delay", "redis.db", "server.idle_timeout", "server.max_header_bytes", "server.read_timeout", "server.shutdown_timeout", "server.write_timeout"},
		UnwiredKeysInFile(write("all.yaml", "server:\n  read_timeout: 5s\n  write_timeout: 5s\n  shutdown_timeout: 5s\n  idle_timeout: 120s\n  max_header_bytes: 1\nredis:\n  db: 0\ncache:\n  ttl: 1h\n  update_interval: 5s\nhttp:\n  max_retries: 3\n  retry_delay: 1s\n")))
}
