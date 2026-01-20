package cmd

import (
	// Standard library
	"fmt"
	"strconv"
	"strings"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/validator"
)

// ValidateConfig validates configuration validity
func ValidateConfig(cfg *Config) error {
	var errors []string

	// Validate port
	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.port_invalid", cfg.Port))
	}

	// Validate Redis address format
	if cfg.Redis != "" {
		parts := strings.Split(cfg.Redis, ":")
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("Invalid Redis address format: %s (should be host:port)", cfg.Redis))
		} else {
			if port, err := strconv.Atoi(parts[1]); err != nil || port < 1 || port > 65535 {
				errors = append(errors, fmt.Sprintf("Invalid Redis port: %s", parts[1]))
			}
		}
	}

	// Validate remote configuration URL (enhanced SSRF protection)
	if cfg.RemoteConfig != "" && cfg.RemoteConfig != define.DEFAULT_REMOTE_CONFIG {
		if err := validator.ValidateRemoteURL(cfg.RemoteConfig); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid remote configuration URL: %s (%v)", cfg.RemoteConfig, err))
		}
	}

	// Validate task interval
	if cfg.TaskInterval < 1 {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.task_interval_invalid", cfg.TaskInterval))
	}

	// Validate mode
	validModes := map[string]bool{
		"DEFAULT":                          true,
		"REMOTE_FIRST":                     true,
		"ONLY_REMOTE":                      true,
		"ONLY_LOCAL":                       true,
		"LOCAL_FIRST":                      true,
		"REMOTE_FIRST_ALLOW_REMOTE_FAILED": true,
		"LOCAL_FIRST_ALLOW_REMOTE_FAILED":  true,
	}
	if !validModes[cfg.Mode] {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.mode_invalid", cfg.Mode))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s:\n  - %s", i18n.TWithLang(i18n.LangZH, "error.config_validation_failed"), strings.Join(errors, "\n  - "))
	}

	return nil
}
