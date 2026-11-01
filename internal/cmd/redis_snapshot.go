package cmd

import (
	"fmt"
	"github.com/soulteary/warden/internal/cache"
	"os"
	"strings"
)

// RedisSnapshotMode normalizes the environment-only migration setting.
func RedisSnapshotMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return "legacy"
	}
	return mode
}

func processRedisSnapshotFromEnv(cfg *Config) {
	cfg.RedisSnapshotMode = RedisSnapshotMode(os.Getenv("REDIS_SNAPSHOT_MODE"))
	cfg.RedisSnapshotNamespace = strings.TrimSpace(os.Getenv("REDIS_SNAPSHOT_NAMESPACE"))
}

func validateRedisSnapshot(cfg *Config) error {
	mode := RedisSnapshotMode(cfg.RedisSnapshotMode)
	switch mode {
	case "legacy":
		return nil
	case "shadow", "v2":
		if !cfg.RedisEnabled {
			return fmt.Errorf("REDIS_SNAPSHOT_MODE requires Redis to be enabled")
		}
		if !cache.ValidateSnapshotNamespace(cfg.RedisSnapshotNamespace) {
			return fmt.Errorf("REDIS_SNAPSHOT_NAMESPACE must be 1-64 letters, digits, colons, underscores or hyphens, beginning with a letter or digit")
		}
		return nil
	default:
		return fmt.Errorf("REDIS_SNAPSHOT_MODE must be legacy, shadow or v2")
	}
}
