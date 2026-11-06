package cmd

import (
	"fmt"
	"github.com/soulteary/warden/internal/config"
	"os"
	"strings"
)

// ParseSnapshotReadPolicy normalizes the environment-only data read policy.
func ParseSnapshotReadPolicy(raw string, environment ...string) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(raw))
	if policy == "" {
		policy = "observe"
		if len(environment) > 0 {
			env, ok := config.ParseEnvironment(environment[0])
			if ok && env.IsProduction() {
				policy = "enforce"
			}
		}
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
