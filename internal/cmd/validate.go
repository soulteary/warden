package cmd

import (
	// Standard library
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	// External packages
	"github.com/soulteary/cli-kit/validator"

	// Internal packages
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/remote"
	wardenvalidator "github.com/soulteary/warden/internal/validator"
)

// ValidateConfig validates configuration validity
func ValidateConfig(cfg *Config) error {
	var errors []string
	if err := validateSourceSetID(cfg.SourceSetID); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateRedisRecovery(cfg); err != nil {
		errors = append(errors, err.Error())
	}
	if _, err := ParseSnapshotReadPolicy(cfg.SnapshotReadPolicy, cfg.Environment); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateRedisSnapshot(cfg); err != nil {
		errors = append(errors, err.Error())
	}

	errors = append(errors, validateEndpointsAndMode(cfg)...)
	errors = append(errors, validateRedisPasswordFile(cfg)...)
	errors = append(errors, validateRedisTransport(cfg)...)

	// Validate deployment environment (security-policy concern only).
	environment, envOK := config.ParseEnvironment(cfg.Environment)
	if !envOK {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.environment_invalid", cfg.Environment))
	}

	// Validate the empty-ruleset policy. An unrecognized value must fail startup rather
	// than silently fall back: an operator who set availability-first and got
	// consistency-first would keep a failure mode they explicitly opted out of.
	if _, ok := config.ParseEmptyRulesetPolicy(cfg.EmptyRulesetPolicy); !ok {
		errors = append(errors, i18n.TfWithLang(i18n.LangZH, "validation.empty_ruleset_policy_invalid", cfg.EmptyRulesetPolicy))
	}

	errors = append(errors, validateBooleanEnv()...)
	errors = append(errors, validateNumericEnv(cfg)...)
	errors = append(errors, validateHMACConfig(cfg)...)
	errors = append(errors, validateServicePolicies(cfg)...)
	if err := remote.ValidatePrivateOrigins(); err != nil {
		errors = append(errors, err.Error())
	}

	// TLS must be configured as a complete transport. startServer enables HTTPS only
	// when both the certificate and private key are present, so accepting a partial
	// configuration here would silently start a plaintext HTTP listener.
	errors = append(errors, validateTLSConfig(cfg)...)

	errors = append(errors, validateEncryptionConfig(cfg)...)

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

	errors = append(errors, validateDecryptKey(cfg)...)

	if len(errors) > 0 {
		return fmt.Errorf("%s:\n  - %s", i18n.TWithLang(i18n.LangZH, "error.config_validation_failed"), strings.Join(errors, "\n  - "))
	}

	return nil
}

const minimumDefaultSnapshotMaxAge = 30 * time.Second

// ParseSnapshotMaxAge parses the maximum age accepted by health checks. An empty
// value derives a safe default from the refresh interval, with enough slack for
// transient scheduling delays. Operators may override it with a Go duration such
// as "2m"; non-positive values are rejected instead of disabling stale detection.
func ParseSnapshotMaxAge(raw string, taskIntervalSec int) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultSnapshotMaxAge(taskIntervalSec), nil
	}
	age, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("expected a positive duration such as 30s or 2m: %w", err)
	}
	if age <= 0 {
		return 0, fmt.Errorf("must be greater than zero")
	}
	return age, nil
}

// DefaultSnapshotMaxAge derives the default freshness threshold from the
// configured refresh interval while preserving a minimum scheduling margin.
func DefaultSnapshotMaxAge(taskIntervalSec int) time.Duration {
	age := time.Duration(taskIntervalSec) * 3 * time.Second
	if age < minimumDefaultSnapshotMaxAge {
		age = minimumDefaultSnapshotMaxAge
	}
	return age
}

// ParseHMACAllowV1 parses the legacy HMAC v1 compatibility switch. The empty
// value is deliberately secure by default; v1 must be enabled explicitly.
func ParseHMACAllowV1(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}

	allow, ok := define.ParseBool(raw)
	if !ok {
		return false, fmt.Errorf("expected a boolean (%s)", define.BooleanSpellings)
	}
	return allow, nil
}

// validateBooleanEnv rejects boolean environment variables holding text that is
// not a boolean. WARDEN_HMAC_ALLOW_V1 and REDIS_TLS_ENABLED are reported by
// their own checks, which name the setting in the same way.
func validateBooleanEnv() []string {
	var errs []string
	for _, message := range define.ValidateBooleanEnv() {
		if strings.HasPrefix(message, "Invalid WARDEN_HMAC_ALLOW_V1 ") || strings.HasPrefix(message, "Invalid REDIS_TLS_ENABLED ") {
			continue
		}
		errs = append(errs, message)
	}
	return errs
}

// validateTLSConfig rejects partial TLS and mTLS configuration before the server
// is started. File contents are loaded by buildTLSConfig during startup.
func validateTLSConfig(cfg *Config) []string {
	certSet := strings.TrimSpace(cfg.TLSCertFile) != ""
	keySet := strings.TrimSpace(cfg.TLSKeyFile) != ""
	caSet := strings.TrimSpace(cfg.TLSCAFile) != ""

	var errs []string
	if certSet != keySet {
		errs = append(errs, "WARDEN_TLS_CERT and WARDEN_TLS_KEY must be configured together")
	}
	if caSet && (!certSet || !keySet) {
		errs = append(errs, "WARDEN_TLS_CA requires WARDEN_TLS_CERT and WARDEN_TLS_KEY")
	}
	if cfg.TLSRequireClientCert {
		if !certSet || !keySet {
			errs = append(errs, "WARDEN_TLS_REQUIRE_CLIENT_CERT=true requires WARDEN_TLS_CERT and WARDEN_TLS_KEY")
		}
		if !caSet {
			errs = append(errs, "WARDEN_TLS_REQUIRE_CLIENT_CERT=true requires WARDEN_TLS_CA")
		}
	}
	return errs
}

// validateProduction returns production-only security violations. These key off the
// deployment ENVIRONMENT (never the merge mode).
func validateProduction(cfg *Config) []string {
	var errs []string

	// 1. Insecure TLS is forbidden in production.
	if cfg.HTTPInsecureTLS {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_tls_not_allowed"))
	}

	hasRemote := usesRemoteSource(cfg)
	if hasRemote {
		if err := remote.ValidateSourceURL(cfg.RemoteConfig, true); err != nil {
			errs = append(errs, err.Error())
		}
		// 2. Remote fetch must have a bounded timeout.
		if cfg.HTTPTimeout <= 0 {
			errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_remote_timeout_required"))
		}
		// 3. Remote encryption must be required (fail closed) in production.
		if !cfg.RemoteEncryptionRequired {
			errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_remote_encryption_required"))
		}
		format, err := remote.ParseEncryptionFormat(cfg.RemoteEncryptionFormat)
		if err == nil && format != remote.FormatV2 {
			errs = append(errs, "production remote configuration requires REMOTE_ENCRYPTION_FORMAT=v2")
		}
	}

	// 4. At least one authentication mechanism must be configured in production. The
	// user-facing endpoints (full user rules / lookup) must never be exposed without
	// authentication. We fail startup (not merely warn) when NONE of API key / HMAC
	// keys / mTLS client-cert verification is configured.
	if !hasConfiguredAuth(cfg) {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_auth_required"))
	}
	errs = append(errs, validateProductionHMACStrength(cfg.HMACKeys)...)
	errs = append(errs, validateProductionAPIKey(cfg.APIKey)...)
	errs = append(errs, validateProductionProxyTrust(os.Getenv("IP_WHITELIST"), os.Getenv("TRUSTED_PROXY_IPS"))...)

	return errs
}

// placeholderAPIKeys are the illustrative API key values used in the README,
// documentation and example configurations. They are public, so a production
// deployment that kept one has no effective API key.
var placeholderAPIKeys = map[string]struct{}{
	"your-secret-api-key-here": {},
	"your-secret-api-key":      {},
	"your-api-key-here":        {},
	"your-api-key":             {},
	"your-key":                 {},
	"your-strong-api-key-here": {},
	"your-warden-api-key-here": {},
	"your_api_key_here":        {},
	"your_api_key":             {},
	"<set-via-secret-manager>": {},
}

// minimumRecommendedAPIKeyLength is the length below which a production API
// key is reported as weak. It is a warning threshold, not a startup requirement.
const minimumRecommendedAPIKeyLength = 16

func isPlaceholderAPIKey(apiKey string) bool {
	_, ok := placeholderAPIKeys[strings.ToLower(strings.TrimSpace(apiKey))]
	return ok
}

// validateProductionAPIKey rejects a documented placeholder used as the
// production API key. The key value is never included in the message.
func validateProductionAPIKey(apiKey string) []string {
	if isPlaceholderAPIKey(apiKey) {
		return []string{"production API_KEY is a documented placeholder value; generate a random key (for example: openssl rand -hex 32)"}
	}
	return nil
}

// APIKeyWarnings returns non-fatal findings about the configured API key for
// the caller to log at startup. It only reports in production, where the key
// may be the sole credential. The key value is never included.
func APIKeyWarnings(cfg *Config) []string {
	environment, ok := config.ParseEnvironment(cfg.Environment)
	if !ok || !environment.IsProduction() {
		return nil
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" || isPlaceholderAPIKey(apiKey) {
		return nil
	}
	if len(apiKey) < minimumRecommendedAPIKeyLength {
		return []string{fmt.Sprintf("API_KEY is shorter than %d characters; use a longer random key (for example: openssl rand -hex 32)", minimumRecommendedAPIKeyLength)}
	}
	return nil
}

const minimumProductionHMACSecretBytes = 32

// validateProductionHMACStrength prevents a syntactically valid but low-entropy
// shared secret from becoming the only production authentication credential.
// Parse errors are reported by the general key-set validation above.
func validateProductionHMACStrength(raw string) []string {
	keys, err := ParseHMACKeys(raw)
	if err != nil || len(keys) == 0 {
		return nil
	}
	for _, secret := range keys {
		if len([]byte(secret)) < minimumProductionHMACSecretBytes {
			return []string{fmt.Sprintf("production HMAC secrets must be at least %d bytes", minimumProductionHMACSecretBytes)}
		}
	}
	return nil
}

// usesRemoteSource mirrors the loader's source-selection semantics. The built-in
// remote URL is still a real remote source in every mode except ONLY_LOCAL; treating
// it as an "unset" sentinel would let production deployments skip encryption checks
// while the loader still fetches it.
func usesRemoteSource(cfg *Config) bool {
	mode, ok := config.ParseMergeMode(cfg.Mode)
	if !ok {
		return false
	}
	return !mode.IsOnlyLocal() && strings.TrimSpace(cfg.RemoteConfig) != ""
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
	keys, err := ParseHMACKeys(raw)
	return err == nil && len(keys) > 0
}

// ParseHMACKeys parses and validates the complete key set. Key identifiers are
// trimmed for stable lookup; secrets are preserved byte-for-byte but may not be
// empty or whitespace-only. Invalid entries fail the whole set closed.
func ParseHMACKeys(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("at least one key is required")
	}
	keys := make(map[string]string, len(parsed))
	for id, secret := range parsed {
		normalizedID := strings.TrimSpace(id)
		if normalizedID == "" {
			return nil, fmt.Errorf("key id must not be empty")
		}
		if strings.TrimSpace(secret) == "" {
			return nil, fmt.Errorf("secret for key %q must not be empty", normalizedID)
		}
		if _, exists := keys[normalizedID]; exists {
			return nil, fmt.Errorf("duplicate key id %q after normalization", normalizedID)
		}
		keys[normalizedID] = secret
	}
	return keys, nil
}

func validateEndpointsAndMode(cfg *Config) []string {
	var errs []string
	// Validate port using cli-kit validator
	if _, err := validator.ValidatePortString(cfg.Port); err != nil {
		errs = append(errs, i18n.TfWithLang(i18n.LangZH, "validation.port_invalid", cfg.Port))
	}

	// Validate Redis address format using cli-kit validator
	if cfg.Redis != "" {
		if _, _, err := validator.ValidateHostPort(cfg.Redis); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid Redis address format: %s (should be host:port): %v", cfg.Redis, err))
		}
	}

	// Validate remote configuration URL through the injectable SSRF-safe resolver path.
	if cfg.RemoteConfig != "" {
		if err := wardenvalidator.ValidateRemoteURL(cfg.RemoteConfig); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid remote configuration URL: %s (%v)", cfg.RemoteConfig, err))
		}
	}

	// A mode that must read the remote source cannot run without one. Without
	// this check the service would start and fail every load.
	if mode, ok := config.ParseMergeMode(cfg.Mode); ok && strings.TrimSpace(cfg.RemoteConfig) == "" &&
		(mode == config.MergeOnlyRemote || mode == config.MergeRemoteFirst) {
		errs = append(errs, fmt.Sprintf("MERGE_MODE=%s requires a remote source: set CONFIG (or --config, remote.url) to its URL, or choose a mode that does not depend on a remote source", mode))
	}

	// Validate task interval
	if cfg.TaskInterval < 1 {
		errs = append(errs, i18n.TfWithLang(i18n.LangZH, "validation.task_interval_invalid", cfg.TaskInterval))
	}
	if _, err := ParseSnapshotMaxAge(os.Getenv("SNAPSHOT_MAX_AGE"), cfg.TaskInterval); err != nil {
		errs = append(errs, fmt.Sprintf("Invalid SNAPSHOT_MAX_AGE: %v", err))
	}

	errs = append(errs, validateRateLimit(cfg)...)
	errs = append(errs, validateTrustedProxies()...)

	// Validate merge mode using the dedicated MergeMode type (data-loading concern only).
	if mode, ok := config.ParseMergeMode(cfg.Mode); !ok || !mode.Validate() {
		errs = append(errs, i18n.TfWithLang(i18n.LangZH, "validation.mode_invalid", cfg.Mode))
	}

	return errs
}

func validateHMACConfig(cfg *Config) []string {
	var errs []string
	// Validate the complete HMAC key set before it reaches authentication middleware.
	// A configured-but-empty secret must never become an authentication credential.
	if strings.TrimSpace(cfg.HMACKeys) != "" {
		if _, err := ParseHMACKeys(cfg.HMACKeys); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid WARDEN_HMAC_KEYS: %v", err))
		}
	}

	if raw := os.Getenv("WARDEN_HMAC_ALLOW_V1"); strings.TrimSpace(raw) != "" {
		if _, err := ParseHMACAllowV1(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid WARDEN_HMAC_ALLOW_V1 %q: %v", strings.TrimSpace(raw), err))
		}
	}

	return errs
}

func validateEncryptionConfig(cfg *Config) []string {
	var errs []string
	if _, err := remote.ParseEncryptionFormat(cfg.RemoteEncryptionFormat); err != nil {
		errs = append(errs, err.Error())
	}
	if cfg.RemoteEncryptionRequired {
		if !cfg.RemoteDecryptEnabled {
			errs = append(errs, "REMOTE_ENCRYPTION_REQUIRED requires REMOTE_DECRYPT_ENABLED=true")
		}
		if strings.TrimSpace(cfg.RemoteRSAPrivateKeyFile) == "" && strings.TrimSpace(cfg.RemoteRSAPrivateKey) == "" {
			errs = append(errs, "REMOTE_ENCRYPTION_REQUIRED requires REMOTE_RSA_PRIVATE_KEY_FILE or REMOTE_RSA_PRIVATE_KEY")
		}
	}

	return errs
}

func validateDecryptKey(cfg *Config) []string {
	var errs []string
	// When remote decrypt is enabled, either RSA private key file or inline PEM must be set
	if cfg.RemoteDecryptEnabled {
		if cfg.RemoteRSAPrivateKeyFile == "" && cfg.RemoteRSAPrivateKey == "" {
			errs = append(errs, "REMOTE_DECRYPT_ENABLED is true but neither REMOTE_RSA_PRIVATE_KEY_FILE nor REMOTE_RSA_PRIVATE_KEY is set")
		} else if cfg.RemoteRSAPrivateKeyFile != "" {
			info, err := os.Stat(cfg.RemoteRSAPrivateKeyFile)
			switch {
			case err != nil:
				if os.IsNotExist(err) {
					errs = append(errs, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q does not exist", cfg.RemoteRSAPrivateKeyFile))
				} else {
					errs = append(errs, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q: %v", cfg.RemoteRSAPrivateKeyFile, err))
				}
			case info.IsDir():
				errs = append(errs, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q is a directory, not a file", cfg.RemoteRSAPrivateKeyFile))
			default:
				f, err := os.Open(cfg.RemoteRSAPrivateKeyFile)
				if err != nil {
					errs = append(errs, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q is not readable: %v", cfg.RemoteRSAPrivateKeyFile, err))
				} else {
					if closeErr := f.Close(); closeErr != nil {
						errs = append(errs, fmt.Sprintf("REMOTE_RSA_PRIVATE_KEY_FILE %q close: %v", cfg.RemoteRSAPrivateKeyFile, closeErr))
					}
				}
			}
		}
		// When only REMOTE_RSA_PRIVATE_KEY (inline PEM) is set, key is validated at load time
	}
	return errs
}

// validateRateLimit rejects unusable rate-limit settings from either the
// environment or the configuration file.
func validateRateLimit(cfg *Config) []string {
	var errs []string
	if raw := strings.TrimSpace(os.Getenv("RATE_LIMIT")); raw != "" {
		if _, err := ParseRateLimit(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid RATE_LIMIT: %v", err))
		}
	}
	if raw := strings.TrimSpace(os.Getenv("RATE_LIMIT_AUTHENTICATED")); raw != "" {
		if _, err := ParseRateLimit(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid RATE_LIMIT_AUTHENTICATED: %v", err))
		}
	}
	if raw := strings.TrimSpace(os.Getenv("RATE_LIMIT_WINDOW")); raw != "" {
		if _, err := ParseRateLimitWindow(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid RATE_LIMIT_WINDOW: %v", err))
		}
	}
	errs = append(errs, responseFieldErrors(cfg.ResponseFields)...)
	// Zero means "not set" and resolves to the default, as it does in YAML.
	if cfg.RateLimit < 0 {
		errs = append(errs, fmt.Sprintf("Invalid rate limit: %d (must be at least 1)", cfg.RateLimit))
	}
	if cfg.RateLimitAuthenticated < 0 {
		errs = append(errs, fmt.Sprintf("Invalid authenticated rate limit: %d (must be at least 1)", cfg.RateLimitAuthenticated))
	}
	if cfg.RateLimitWindow < 0 || (cfg.RateLimitWindow > 0 && cfg.RateLimitWindow < time.Second) {
		errs = append(errs, fmt.Sprintf("Invalid rate limit window: %s (must be at least 1s)", cfg.RateLimitWindow))
	}
	return errs
}

// validateTrustedProxies rejects TRUSTED_PROXY_IPS entries that are not IP
// addresses or CIDR ranges. They would otherwise be dropped, and a list that
// ends up empty trusts forwarded headers from every private-address peer.
func validateTrustedProxies() []string {
	if err := define.ValidateTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS")); err != nil {
		return []string{fmt.Sprintf("Invalid TRUSTED_PROXY_IPS: %v", err)}
	}
	return nil
}

// validateProductionProxyTrust refuses an IP_WHITELIST whose client address
// can be chosen by the caller. With TRUSTED_PROXY_IPS empty, forwarded headers
// are accepted from every private or loopback peer, so such a peer can present
// an allowlisted address. HEALTH_CHECK_IP_WHITELIST is not covered: the health
// handler compares it with the TCP peer and never reads forwarded headers.
func validateProductionProxyTrust(ipWhitelist, trustedProxies string) []string {
	if strings.TrimSpace(ipWhitelist) == "" || len(define.ParseTrustedProxyIPs(trustedProxies)) > 0 {
		return nil
	}
	return []string{"production IP_WHITELIST requires TRUSTED_PROXY_IPS: list the reverse proxy addresses, or set TRUSTED_PROXY_IPS=none to ignore forwarded headers and use the TCP peer address"}
}

// knownResponseFields are the names RESPONSE_FIELDS can select. They match the
// keys of the list and user responses and the fields /v1/lookup filters.
var knownResponseFields = map[string]struct{}{
	"phone":           {},
	"mail":            {},
	"user_id":         {},
	"status":          {},
	"scope":           {},
	"role":            {},
	"name":            {},
	"dingtalk_userid": {},
}

// responseFieldErrors rejects allowlist entries that select nothing. Names are
// matched exactly, so a typo such as "email" or "Mail" would otherwise remove
// the field from every response without any notice.
func responseFieldErrors(fields []string) []string {
	var errs []string
	for _, field := range fields {
		if _, ok := knownResponseFields[field]; !ok {
			errs = append(errs, fmt.Sprintf("Invalid RESPONSE_FIELDS entry %q (want phone, mail, user_id, status, scope, role, name or dingtalk_userid)", field))
		}
	}
	return errs
}
