package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/soulteary/cli-kit/flagutil"

	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
)

// processRedisTransportFromEnv reads the environment-only Redis ACL username and
// TLS settings. An unparsable REDIS_TLS_ENABLED is left for ValidateConfig to
// reject, so a typo cannot silently select a plaintext connection.
func processRedisTransportFromEnv(cfg *Config) {
	cfg.RedisUsername = strings.TrimSpace(os.Getenv("REDIS_USERNAME"))
	if enabled, err := ParseRedisTLSEnabled(os.Getenv("REDIS_TLS_ENABLED")); err == nil {
		cfg.RedisTLSEnabled = enabled
	}
	cfg.RedisTLSCAFile = strings.TrimSpace(os.Getenv("REDIS_TLS_CA_FILE"))
	cfg.RedisTLSServerName = strings.TrimSpace(os.Getenv("REDIS_TLS_SERVER_NAME"))
}

// ParseRedisTLSEnabled parses REDIS_TLS_ENABLED. The empty value means disabled.
func ParseRedisTLSEnabled(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	enabled, ok := define.ParseBool(raw)
	if !ok {
		return false, fmt.Errorf("expected a boolean (%s)", define.BooleanSpellings)
	}
	return enabled, nil
}

// RedisTLSConfig builds the client TLS configuration for Redis. It returns nil
// when TLS is disabled. Certificates are verified against REDIS_TLS_CA_FILE when
// set and against the system roots otherwise.
func RedisTLSConfig(cfg *Config) (*tls.Config, error) {
	if !cfg.RedisTLSEnabled {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.RedisTLSServerName,
	}
	if cfg.RedisTLSCAFile != "" {
		// #nosec G304 -- path is operator configuration (env REDIS_TLS_CA_FILE), not request input
		caPEM, err := os.ReadFile(cfg.RedisTLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read REDIS_TLS_CA_FILE: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("REDIS_TLS_CA_FILE %q contains no PEM certificate", cfg.RedisTLSCAFile)
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// validateRedisTransport rejects Redis TLS settings that would otherwise be
// ignored or fail only when the first connection is made.
func validateRedisTransport(cfg *Config) []string {
	var errs []string
	if raw := strings.TrimSpace(os.Getenv("REDIS_TLS_ENABLED")); raw != "" {
		if _, err := ParseRedisTLSEnabled(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid REDIS_TLS_ENABLED %q: %v", raw, err))
		}
	}
	if !cfg.RedisTLSEnabled {
		if cfg.RedisTLSCAFile != "" || cfg.RedisTLSServerName != "" {
			errs = append(errs, "REDIS_TLS_CA_FILE and REDIS_TLS_SERVER_NAME require REDIS_TLS_ENABLED=true")
		}
		return errs
	}
	if _, err := RedisTLSConfig(cfg); err != nil {
		errs = append(errs, err.Error())
	}
	return errs
}

// validateRedisPasswordFile rejects an explicitly configured REDIS_PASSWORD_FILE
// that did not yield a password. Loading tolerates the read error, so without
// this check Warden would connect to Redis without credentials.
func validateRedisPasswordFile(cfg *Config) []string {
	if !cfg.RedisEnabled || cfg.RedisPassword != "" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("REDIS_PASSWORD")) != "" {
		return nil
	}
	file := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))
	if file == "" {
		return nil
	}
	if _, err := flagutil.ReadPasswordFromFile(file); err != nil {
		return []string{fmt.Sprintf("REDIS_PASSWORD_FILE %q is not readable: %v", file, err)}
	}
	return []string{fmt.Sprintf("REDIS_PASSWORD_FILE %q is empty", file)}
}

// validateConfigFileRedisPassword applies the same rule to redis.password_file
// from the configuration file, which is not visible on the converted Config.
func validateConfigFileRedisPassword(fileCfg *config.Config, cfg *Config) error {
	file := strings.TrimSpace(fileCfg.Redis.PasswordFile)
	if file == "" || !cfg.RedisEnabled || cfg.RedisPassword != "" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("REDIS_PASSWORD")) != "" || strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE")) != "" {
		// REDIS_PASSWORD_FILE is reported by ValidateConfig.
		return nil
	}
	if _, err := fileCfg.GetRedisPassword(); err != nil {
		return fmt.Errorf("redis.password_file %q is not readable: %w", file, err)
	}
	return fmt.Errorf("redis.password_file %q is empty", file)
}
