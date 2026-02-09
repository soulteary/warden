package cmd

import (
	// Standard library
	"fmt"
	"os"
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

	// Validate DATA_DIR when set: must exist and be a directory
	if cfg.DataDir != "" {
		info, err := os.Stat(cfg.DataDir)
		if err != nil {
			if os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("DATA_DIR %q does not exist", cfg.DataDir))
			} else {
				errors = append(errors, fmt.Sprintf("DATA_DIR %q: %v", cfg.DataDir, err))
			}
		} else if !info.IsDir() {
			errors = append(errors, fmt.Sprintf("DATA_DIR %q is not a directory", cfg.DataDir))
		}
	}

	// When remote decrypt is enabled, RSA private key file must be set and exist/readable
	if cfg.RemoteDecryptEnabled {
		if cfg.RemoteRSAPrivateKeyFile == "" {
			errors = append(errors, "REMOTE_DECRYPT_ENABLED is true but REMOTE_RSA_PRIVATE_KEY_FILE is not set")
		} else {
			info, err := os.Stat(cfg.RemoteRSAPrivateKeyFile)
			switch {
			case err != nil:
				if os.IsNotExist(err) {
					errors = append(errors, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q does not exist", cfg.RemoteRSAPrivateKeyFile))
				} else {
					errors = append(errors, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q: %v", cfg.RemoteRSAPrivateKeyFile, err))
				}
			case info.IsDir():
				errors = append(errors, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q is a directory, not a file", cfg.RemoteRSAPrivateKeyFile))
			default:
				f, err := os.Open(cfg.RemoteRSAPrivateKeyFile)
				if err != nil {
					errors = append(errors, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q is not readable: %v", cfg.RemoteRSAPrivateKeyFile, err))
				} else {
					if closeErr := f.Close(); closeErr != nil {
						errors = append(errors, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q close: %v", cfg.RemoteRSAPrivateKeyFile, closeErr))
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s:\n  - %s", i18n.TWithLang(i18n.LangZH, "error.config_validation_failed"), strings.Join(errors, "\n  - "))
	}

	return nil
}
