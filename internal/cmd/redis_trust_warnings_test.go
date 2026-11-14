package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedisTrustWarnings(t *testing.T) {
	warnings := RedisTrustWarnings(&Config{Environment: "production", RedisEnabled: true})
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "warden:users:cache")
	require.Contains(t, warnings[0], "REDIS_PASSWORD")

	require.Len(t, RedisTrustWarnings(&Config{Environment: "production", RedisEnabled: true, RedisSnapshotMode: "shadow"}), 1)
	for name, cfg := range map[string]*Config{
		"password":    {Environment: "production", RedisEnabled: true, RedisPassword: "secret"},
		"disabled":    {Environment: "production"},
		"development": {Environment: "development", RedisEnabled: true},
		"v2":          {Environment: "production", RedisEnabled: true, RedisSnapshotMode: "v2"},
	} {
		require.Empty(t, RedisTrustWarnings(cfg), name)
	}
}
