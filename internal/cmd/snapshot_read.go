package cmd

import (
	"fmt"
	"os"
	"strings"
)

// ParseSnapshotReadPolicy normalizes the environment-only data read policy.
func ParseSnapshotReadPolicy(raw string) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(raw))
	if policy == "" {
		policy = "observe"
	}
	switch policy {
	case "observe", "enforce":
		return policy, nil
	default:
		return "", fmt.Errorf("SNAPSHOT_READ_POLICY must be observe or enforce")
	}
}

func processSnapshotReadPolicyFromEnv(cfg *Config) {
	cfg.SnapshotReadPolicy = os.Getenv("SNAPSHOT_READ_POLICY")
}
