package cmd

import (
	"strings"

	"github.com/soulteary/warden/internal/config"
)

// RedisTrustWarnings reports a production Redis connection without a password
// while the legacy cache is in use. Outside ONLY_LOCAL, startup adopts the rule
// set stored under the fixed key warden:users:cache before it reads any source,
// so whoever can write to that Redis instance decides who is on the allow
// list. Without a password that is every host that can reach it.
func RedisTrustWarnings(cfg *Config) []string {
	environment, _ := config.ParseEnvironment(cfg.Environment)
	if !environment.IsProduction() || !cfg.RedisEnabled || strings.TrimSpace(cfg.RedisPassword) != "" {
		return nil
	}
	if RedisSnapshotMode(cfg.RedisSnapshotMode) == "v2" {
		return nil
	}
	return []string{"Redis is used without a password: the allow list cached under warden:users:cache is adopted at startup, so any host that can reach this Redis instance can change who is allowed; set REDIS_PASSWORD or REDIS_PASSWORD_FILE and do not share the instance with other services"}
}
