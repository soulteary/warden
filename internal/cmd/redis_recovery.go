package cmd

import (
	"fmt"
	"os"
	"strings"
)

// ParseRedisSnapshotRecovery normalizes the environment-only recovery policy.
func ParseRedisSnapshotRecovery(raw string) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(raw))
	if policy == "" {
		policy = "bootstrap"
	}
	switch policy {
	case "bootstrap", "stale":
		return policy, nil
	default:
		return "", fmt.Errorf("REDIS_SNAPSHOT_RECOVERY must be bootstrap or stale")
	}
}

func processRedisRecoveryFromEnv(cfg *Config) {
	cfg.RedisSnapshotRecovery = os.Getenv("REDIS_SNAPSHOT_RECOVERY")
}

func validateRedisRecovery(cfg *Config) error {
	policy, err := ParseRedisSnapshotRecovery(cfg.RedisSnapshotRecovery)
	if err != nil {
		return err
	}
	if policy == "stale" && (!cfg.RedisEnabled || RedisSnapshotMode(cfg.RedisSnapshotMode) != "v2") {
		return fmt.Errorf("REDIS_SNAPSHOT_RECOVERY=stale requires Redis and REDIS_SNAPSHOT_MODE=v2")
	}
	return nil
}
