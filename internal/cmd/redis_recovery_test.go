package cmd

import (
	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedisRecoveryPolicy(t *testing.T) {
	for _, raw := range []string{"", "bootstrap", " BOOTSTRAP ", "stale", " STALE "} {
		_, err := ParseRedisSnapshotRecovery(raw)
		require.NoError(t, err)
	}
	_, err := ParseRedisSnapshotRecovery("always")
	require.Error(t, err)
	require.NoError(t, validateRedisRecovery(&Config{}))
	require.Error(t, validateRedisRecovery(&Config{RedisSnapshotRecovery: "stale"}))
	require.Error(t, validateRedisRecovery(&Config{RedisSnapshotRecovery: "stale", RedisEnabled: true, RedisSnapshotMode: "shadow"}))
	require.NoError(t, validateRedisRecovery(&Config{RedisSnapshotRecovery: "stale", RedisEnabled: true, RedisSnapshotMode: "v2"}))
	t.Setenv("REDIS_SNAPSHOT_RECOVERY", "stale")
	withCleanFlags(t)
	for _, cfg := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Equal(t, "stale", cfg.RedisSnapshotRecovery)
	}
	require.ErrorContains(t, ValidateConfig(&Config{RedisSnapshotRecovery: "invalid"}), "REDIS_SNAPSHOT_RECOVERY")
}
