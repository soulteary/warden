package cmd

import (
	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedisSnapshotMigrationConfig(t *testing.T) {
	for _, mode := range []string{"", "legacy", "shadow", "v2", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &Config{RedisEnabled: true, RedisSnapshotMode: mode, RedisSnapshotNamespace: "users-prod"}
			if mode == "invalid" {
				require.Error(t, validateRedisSnapshot(cfg))
				return
			}
			require.NoError(t, validateRedisSnapshot(cfg))
			if RedisSnapshotMode(mode) != "legacy" {
				cfg.RedisSnapshotNamespace = ""
				require.Error(t, validateRedisSnapshot(cfg))
				cfg.RedisSnapshotNamespace = "users-prod"
				cfg.RedisEnabled = false
				require.Error(t, validateRedisSnapshot(cfg))
			}
		})
	}
	t.Setenv("REDIS_SNAPSHOT_MODE", " V2 ")
	t.Setenv("REDIS_SNAPSHOT_NAMESPACE", " users-prod ")
	withCleanFlags(t)
	for _, cfg := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Equal(t, "v2", cfg.RedisSnapshotMode)
		require.Equal(t, "users-prod", cfg.RedisSnapshotNamespace)
	}
}
