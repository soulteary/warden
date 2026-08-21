package cmd

import (
	// Standard library
	"encoding/json"
	"fmt"
	"os"
	"strings"

	// External packages
	"github.com/soulteary/cli-kit/validator"

	// Internal packages
	"github.com/soulteary/warden/internal/config"
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

	// Validate merge mode using the dedicated MergeMode type (data-loading concern only).
	if mode, ok := config.ParseMergeMode(cfg.Mode); !ok || !mode.Validate() {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.mode_invalid", cfg.Mode))
	}

	// Validate deployment environment (security-policy concern only).
	environment, envOK := config.ParseEnvironment(cfg.Environment)
	if !envOK {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.environment_invalid", cfg.Environment))
	}

	// Production-specific security validation. These key off ENVIRONMENT, never the merge mode.
	if environment.IsProduction() {
		errors = append(errors, validateProduction(cfg)...)
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

	// When remote decrypt is enabled, either RSA private key file or inline PEM must be set
	if cfg.RemoteDecryptEnabled {
		if cfg.RemoteRSAPrivateKeyFile == "" && cfg.RemoteRSAPrivateKey == "" {
			errors = append(errors, "REMOTE_DECRYPT_ENABLED is true but neither REMOTE_RSA_PRIVATE_KEY_FILE nor REMOTE_RSA_PRIVATE_KEY is set")
		} else if cfg.RemoteRSAPrivateKeyFile != "" {
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
		// When only REMOTE_RSA_PRIVATE_KEY (inline PEM) is set, key is validated at load time
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s:\n  - %s", i18n.TWithLang(i18n.LangZH, "error.config_validation_failed"), strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateProduction returns production-only security violations. These key off the
// deployment ENVIRONMENT (never the merge mode).
func validateProduction(cfg *Config) []string {
	var errs []string

	// 1. Insecure TLS is forbidden in production.
	if cfg.HTTPInsecureTLS {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_tls_not_allowed"))
	}

	hasRemote := cfg.RemoteConfig != "" && cfg.RemoteConfig != define.DEFAULT_REMOTE_CONFIG
	if hasRemote {
		// 2. Remote fetch must have a bounded timeout.
		if cfg.HTTPTimeout <= 0 {
			errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_remote_timeout_required"))
		}
		// 3. Remote encryption must be required (fail closed) in production.
		if !cfg.RemoteEncryptionRequired {
			errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_remote_encryption_required"))
		}
	}

	// 4. At least one authentication mechanism must be configured in production. The
	// user-facing endpoints (full user rules / lookup) must never be exposed without
	// authentication. We fail startup (not merely warn) when NONE of API key / HMAC
	// keys / mTLS client-cert verification is configured.
	if !hasConfiguredAuth(cfg) {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_auth_required"))
	}

	return errs
}

// hasConfiguredAuth reports whether any service authentication mechanism is
// configured: an API key, at least one HMAC key, or mTLS client-cert verification
// (a client CA plus required-client-cert). A TLS server cert alone is transport
// encryption, not client authentication, so it does not count.
func hasConfiguredAuth(cfg *Config) bool {
	if strings.TrimSpace(cfg.APIKey) != "" {
		return true
	}
	if hasHMACKeys(cfg.HMACKeys) {
		return true
	}
	if strings.TrimSpace(cfg.TLSCAFile) != "" && cfg.TLSRequireClientCert {
		return true
	}
	return false
}

// hasHMACKeys reports whether the HMAC keys JSON contains at least one key. It
// parses defensively and never logs the secret material.
func hasHMACKeys(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	var keys map[string]string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return false
	}
	for id, secret := range keys {
		if strings.TrimSpace(id) != "" && strings.TrimSpace(secret) != "" {
			return true
		}
	}
	return false
}
