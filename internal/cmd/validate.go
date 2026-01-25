package cmd

import (
	// Standard library
	"fmt"
	"strings"

	// External packages
	"github.com/soulteary/cli-kit/validator"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
)

// ValidateConfig validates configuration validity
func ValidateConfig(cfg *Config) error {
	var errors []string

	// Validate port using cli-kit validator
	if _, err := validator.ValidatePortString(cfg.Port); err != nil {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.port_invalid", cfg.Port))
	}

	// Validate Redis address format using cli-kit validator
	if cfg.Redis != "" {
		if _, _, err := validator.ValidateHostPort(cfg.Redis); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid Redis address format: %s (should be host:port): %v", cfg.Redis, err))
		}
	}

	// Validate remote configuration URL (enhanced SSRF protection using cli-kit/validator)
	if cfg.RemoteConfig != "" && cfg.RemoteConfig != define.DEFAULT_REMOTE_CONFIG {
		if err := validator.ValidateURL(cfg.RemoteConfig, nil); err != nil {
			errors = append(errors, fmt.Sprintf("Invalid remote configuration URL: %s (%v)", cfg.RemoteConfig, err))
		}
	}

	// Validate task interval
	if cfg.TaskInterval < 1 {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.task_interval_invalid", cfg.TaskInterval))
	}

	// Validate mode using cli-kit validator
	validModes := []string{
		"DEFAULT",
		"REMOTE_FIRST",
		"ONLY_REMOTE",
		"ONLY_LOCAL",
		"LOCAL_FIRST",
		"REMOTE_FIRST_ALLOW_REMOTE_FAILED",
		"LOCAL_FIRST_ALLOW_REMOTE_FAILED",
	}
	if err := validator.ValidateEnum(cfg.Mode, validModes, true); err != nil {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.mode_invalid", cfg.Mode))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s:\n  - %s", i18n.TWithLang(i18n.LangZH, "error.config_validation_failed"), strings.Join(errors, "\n  - "))
	}

	return nil
}
