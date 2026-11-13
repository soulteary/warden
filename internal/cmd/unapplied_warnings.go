package cmd

import (
	"fmt"
	"strings"

	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
)

// UnappliedSettingWarnings returns the non-fatal findings about settings that
// are accepted but do not take effect as written, for the caller to log once
// at startup.
func UnappliedSettingWarnings(cfg *Config) []string {
	var warnings []string
	warnings = append(warnings, unwiredConfigKeyWarnings(cfg.ConfigFile)...)
	warnings = append(warnings, intervalWarnings(cfg.TaskInterval)...)
	return warnings
}

// unwiredConfigKeyWarnings names the configuration-file keys that are parsed
// but not applied, so a tuned value that changes nothing is not mistaken for
// one that took effect.
func unwiredConfigKeyWarnings(configFile string) []string {
	keys := config.UnwiredKeysInFile(configFile)
	if len(keys) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("configuration file sets keys that are parsed but not applied: %s; the service uses its built-in values for them (see \"YAML schema and application wiring\" in docs/enUS/CONFIGURATION.md)", strings.Join(keys, ", "))}
}

// intervalWarnings reports a refresh interval below the minimum the scheduler
// applies. Validation accepts any interval of at least one second.
func intervalWarnings(taskInterval int) []string {
	if taskInterval < 1 || taskInterval >= define.DEFAULT_TASK_INTERVAL {
		return nil
	}
	return []string{fmt.Sprintf("refresh interval of %d seconds is below the minimum of %d seconds; refreshing every %d seconds", taskInterval, define.DEFAULT_TASK_INTERVAL, define.DEFAULT_TASK_INTERVAL)}
}
