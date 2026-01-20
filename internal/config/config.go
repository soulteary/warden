// Package config provides configuration file loading and management functionality.
// Supports YAML format configuration files, and provides configuration validation and default value handling.
package config

import (
	// Standard library
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Third-party libraries
	"gopkg.in/yaml.v3"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/errors"
	"github.com/soulteary/warden/internal/i18n"
)

// Config application configuration struct
//
//nolint:govet // fieldalignment: field order is affected by YAML serialization tags, optimization may break configuration file compatibility
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Redis     RedisConfig     `yaml:"redis"`
	Remote    RemoteConfig    `yaml:"remote"`
	HTTP      HTTPConfig      `yaml:"http"`
	Cache     CacheConfig     `yaml:"cache"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	App       AppConfig       `yaml:"app"`
	Task      TaskConfig      `yaml:"task"`
}

// ServerConfig server configuration
type ServerConfig struct {
	Port            string        `yaml:"port"`             // 16 bytes
	ReadTimeout     time.Duration `yaml:"read_timeout"`     // 8 bytes
	WriteTimeout    time.Duration `yaml:"write_timeout"`    // 8 bytes
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"` // 8 bytes
	IdleTimeout     time.Duration `yaml:"idle_timeout"`     // 8 bytes
	MaxHeaderBytes  int           `yaml:"max_header_bytes"` // 8 bytes
}

// RedisConfig Redis configuration
type RedisConfig struct {
	Addr         string `yaml:"addr"`          // 16 bytes
	Password     string `yaml:"password"`      // 16 bytes
	PasswordFile string `yaml:"password_file"` // 16 bytes
	DB           int    `yaml:"db"`            // 8 bytes
}

// CacheConfig cache configuration
type CacheConfig struct {
	TTL            time.Duration `yaml:"ttl"`
	UpdateInterval time.Duration `yaml:"update_interval"`
}

// RateLimitConfig rate limit configuration
type RateLimitConfig struct {
	Rate   int           `yaml:"rate"`
	Window time.Duration `yaml:"window"`
}

// HTTPConfig HTTP client configuration
type HTTPConfig struct {
	Timeout      time.Duration `yaml:"timeout"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	InsecureTLS  bool          `yaml:"insecure_tls"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

// RemoteConfig remote configuration
type RemoteConfig struct {
	URL  string `yaml:"url"`
	Key  string `yaml:"key"`
	Mode string `yaml:"mode"`
}

// TaskConfig task configuration
type TaskConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// AppConfig application configuration
type AppConfig struct {
	Mode   string `yaml:"mode"`
	APIKey string `yaml:"api_key"` // API Key for authentication (sensitive information, recommend using environment variables)
}

// LoadFromFile loads configuration from configuration file
// Supports YAML and TOML formats (determined by file extension)
// Priority: configuration file > environment variables > default values
func LoadFromFile(configPath string) (*Config, error) {
	cfg := &Config{}

	// If configuration file exists, attempt to load
	if configPath != "" {
		// Validate configuration file path to prevent path traversal attacks
		// Note: does not restrict directory here, allows reading configuration file from any location
		// If restriction is needed, can pass allowedDirs parameter
		validatedPath, err := validateConfigPath(configPath)
		if err != nil {
			return nil, errors.ErrConfigLoad.WithError(err)
		}

		if _, err := os.Stat(validatedPath); err == nil {
			// #nosec G304 -- configuration file path has been validated, is safe
			data, err := os.ReadFile(validatedPath)
			if err != nil {
				return nil, errors.ErrConfigLoad.WithError(err)
			}

			// Determine format by file extension
			ext := strings.ToLower(filepath.Ext(validatedPath))
			switch ext {
			case ".yaml", ".yml":
				if err := yaml.Unmarshal(data, cfg); err != nil {
					return nil, errors.ErrConfigParse.WithError(err)
				}
			case ".toml":
				// TOML support requires additional library, return error message here
				return nil, errors.ErrConfigParse.WithMessage(i18n.TWithLang(i18n.LangZH, "error.toml_not_supported"))
			default:
				// Default to try YAML
				if err := yaml.Unmarshal(data, cfg); err != nil {
					return nil, errors.ErrConfigParse.WithError(err)
				}
			}
		}
	}

	// Apply default values
	applyDefaults(cfg)

	// Override configuration from environment variables (priority higher than configuration file)
	overrideFromEnv(cfg)

	// Validate configuration
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyServerDefaults applies server default values
func applyServerDefaults(cfg *Config) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = fmt.Sprintf("%d", define.DEFAULT_PORT)
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = define.DEFAULT_TIMEOUT * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = define.DEFAULT_TIMEOUT * time.Second
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = define.SHUTDOWN_TIMEOUT
	}
	if cfg.Server.MaxHeaderBytes == 0 {
		cfg.Server.MaxHeaderBytes = define.MAX_HEADER_BYTES
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = define.IDLE_TIMEOUT
	}
}

// applyRedisDefaults applies Redis default values
func applyRedisDefaults(cfg *Config) {
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = define.DEFAULT_REDIS
	}
}

// applyCacheDefaults applies cache default values
func applyCacheDefaults(cfg *Config) {
	if cfg.Cache.UpdateInterval == 0 {
		cfg.Cache.UpdateInterval = define.DEFAULT_TASK_INTERVAL * time.Second
	}
}

// applyRateLimitDefaults applies rate limit default values
func applyRateLimitDefaults(cfg *Config) {
	if cfg.RateLimit.Rate == 0 {
		cfg.RateLimit.Rate = define.DEFAULT_RATE_LIMIT
	}
	if cfg.RateLimit.Window == 0 {
		cfg.RateLimit.Window = define.DEFAULT_RATE_LIMIT_WINDOW
	}
}

// applyHTTPDefaults applies HTTP default values
func applyHTTPDefaults(cfg *Config) {
	if cfg.HTTP.Timeout == 0 {
		cfg.HTTP.Timeout = define.DEFAULT_TIMEOUT * time.Second
	}
	if cfg.HTTP.MaxIdleConns == 0 {
		cfg.HTTP.MaxIdleConns = define.DEFAULT_MAX_IDLE_CONNS
	}
	if cfg.HTTP.MaxRetries == 0 {
		cfg.HTTP.MaxRetries = define.HTTP_RETRY_MAX_RETRIES
	}
	if cfg.HTTP.RetryDelay == 0 {
		cfg.HTTP.RetryDelay = define.HTTP_RETRY_DELAY
	}
}

// applyRemoteDefaults applies remote configuration default values
func applyRemoteDefaults(cfg *Config) {
	if cfg.Remote.URL == "" {
		cfg.Remote.URL = define.DEFAULT_REMOTE_CONFIG
	}
	if cfg.Remote.Key == "" {
		cfg.Remote.Key = define.DEFAULT_REMOTE_KEY
	}
	if cfg.Remote.Mode == "" {
		cfg.Remote.Mode = define.DEFAULT_MODE
	}
}

// applyTaskDefaults applies task default values
func applyTaskDefaults(cfg *Config) {
	if cfg.Task.Interval == 0 {
		cfg.Task.Interval = define.DEFAULT_TASK_INTERVAL * time.Second
	}
}

// applyAppDefaults applies application default values
func applyAppDefaults(cfg *Config) {
	if cfg.App.Mode == "" {
		cfg.App.Mode = define.DEFAULT_MODE
	}
	// API Key defaults to empty, needs to be set via environment variable or configuration file
}

// applyDefaults applies default values
func applyDefaults(cfg *Config) {
	applyServerDefaults(cfg)
	applyRedisDefaults(cfg)
	applyCacheDefaults(cfg)
	applyRateLimitDefaults(cfg)
	applyHTTPDefaults(cfg)
	applyRemoteDefaults(cfg)
	applyTaskDefaults(cfg)
	applyAppDefaults(cfg)
}

// overrideFromEnv overrides configuration from environment variables
func overrideFromEnv(cfg *Config) {
	// Server
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}

	// Redis
	if redis := os.Getenv("REDIS"); redis != "" {
		cfg.Redis.Addr = redis
	}
	// Redis password priority: environment variable > password file > configuration file
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))
	if redisPasswordEnv != "" {
		cfg.Redis.Password = redisPasswordEnv
		cfg.Redis.PasswordFile = "" // Clear file path
	} else if redisPasswordFile != "" {
		cfg.Redis.PasswordFile = redisPasswordFile
	}

	// Remote
	if config := os.Getenv("CONFIG"); config != "" {
		cfg.Remote.URL = config
	}
	if key := os.Getenv("KEY"); key != "" {
		cfg.Remote.Key = key
	}
	if mode := os.Getenv("MODE"); mode != "" {
		cfg.Remote.Mode = mode
		cfg.App.Mode = mode
	}

	// HTTP
	if timeout := os.Getenv("HTTP_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.HTTP.Timeout = t
		}
	}
	if maxIdleConns := os.Getenv("HTTP_MAX_IDLE_CONNS"); maxIdleConns != "" {
		if n, err := parseInt(maxIdleConns); err == nil {
			cfg.HTTP.MaxIdleConns = n
		}
	}
	if insecureTLS := os.Getenv("HTTP_INSECURE_TLS"); insecureTLS != "" {
		cfg.HTTP.InsecureTLS = strings.EqualFold(insecureTLS, "true") || insecureTLS == "1"
	}

	// App
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		cfg.App.APIKey = apiKey
	}
}

// validate validates configuration
func validate(cfg *Config) error {
	var errs []string

	if cfg.Server.Port == "" {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.server_port_empty"))
	}

	if cfg.Redis.Addr == "" {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.redis_addr_empty"))
	}

	if cfg.Task.Interval < time.Second {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.task_interval_too_short"))
	}

	// Force TLS verification in production environment
	isProduction := cfg.App.Mode == "production" || cfg.App.Mode == "prod"
	if isProduction && cfg.HTTP.InsecureTLS {
		errs = append(errs, i18n.TWithLang(i18n.LangZH, "validation.prod_tls_not_allowed"))
	}

	if len(errs) > 0 {
		return errors.ErrConfigValidation.WithMessage(strings.Join(errs, "; "))
	}

	return nil
}

// parseInt parses integer
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// validateConfigPath validates configuration file path to prevent path traversal attacks
func validateConfigPath(path string) (string, error) {
	if path == "" {
		return "", errors.ErrConfigLoad.WithMessage("configuration file path cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.ErrConfigLoad.WithError(err)
	}

	// Check if contains path traversal
	if strings.Contains(absPath, "..") {
		return "", errors.ErrConfigLoad.WithMessage("configuration file path cannot contain path traversal characters (..)")
	}

	return absPath, nil
}

// GetRedisPassword gets Redis password (handles file reading)
func (c *Config) GetRedisPassword() (string, error) {
	// Priority: environment variable > password file > password in configuration file
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	if redisPasswordEnv != "" {
		return redisPasswordEnv, nil
	}

	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))
	if redisPasswordFile == "" && c.Redis.PasswordFile != "" {
		redisPasswordFile = c.Redis.PasswordFile
	}

	if redisPasswordFile != "" {
		// Validate file path to prevent path traversal attacks
		absPath, err := validateConfigPath(redisPasswordFile)
		if err != nil {
			return "", errors.ErrConfigLoad.WithError(err)
		}
		// #nosec G304 -- file path has been validated, is safe
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", errors.ErrConfigLoad.WithError(err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	return c.Redis.Password, nil
}

// LegacyConfig old configuration format (for backward compatibility)
// Note: this type has the same structure as cmd.Config, but defined in a different package
type LegacyConfig struct {
	Port             string // 16 bytes
	Redis            string // 16 bytes
	RedisPassword    string // 16 bytes
	RemoteConfig     string // 16 bytes
	RemoteKey        string // 16 bytes
	Mode             string // 16 bytes
	TaskInterval     int    // 8 bytes
	HTTPTimeout      int    // 8 bytes
	HTTPMaxIdleConns int    // 8 bytes
	HTTPInsecureTLS  bool   // 1 byte (padding to 8 bytes)
}

// ToLegacyConfig converts to old Config format (maintains backward compatibility)
func (c *Config) ToLegacyConfig() *LegacyConfig {
	redisPassword, err := c.GetRedisPassword()
	if err != nil {
		// If password retrieval fails, use empty string
		redisPassword = ""
	}
	// Prefer Remote.Mode, if empty then use App.Mode
	mode := strings.TrimSpace(c.Remote.Mode)
	if mode == "" {
		mode = strings.TrimSpace(c.App.Mode)
	}
	// If still empty, use default value
	if mode == "" {
		mode = define.DEFAULT_MODE
	}
	return &LegacyConfig{
		Port:             c.Server.Port,
		Redis:            c.Redis.Addr,
		RedisPassword:    redisPassword,
		RemoteConfig:     c.Remote.URL,
		RemoteKey:        c.Remote.Key,
		TaskInterval:     int(c.Task.Interval.Seconds()),
		Mode:             mode,
		HTTPTimeout:      int(c.HTTP.Timeout.Seconds()),
		HTTPMaxIdleConns: c.HTTP.MaxIdleConns,
		HTTPInsecureTLS:  c.HTTP.InsecureTLS,
	}
}

// CmdConfigData configuration data structure (used to convert to cmd.Config, avoid circular dependency)
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type CmdConfigData struct {
	Port             string // 16 bytes
	Redis            string // 16 bytes
	RedisPassword    string // 16 bytes
	RedisEnabled     bool   // 1 byte (padding to 8 bytes)
	RemoteConfig     string // 16 bytes
	RemoteKey        string // 16 bytes
	Mode             string // 16 bytes
	APIKey           string // 16 bytes
	TaskInterval     int    // 8 bytes
	HTTPTimeout      int    // 8 bytes
	HTTPMaxIdleConns int    // 8 bytes
	HTTPInsecureTLS  bool   // 1 byte (padding to 8 bytes)
}

// ToCmdConfig converts to cmd.Config format
func (c *Config) ToCmdConfig() *CmdConfigData {
	redisPassword, err := c.GetRedisPassword()
	if err != nil {
		// If password retrieval fails, use empty string
		redisPassword = ""
	}
	// Prefer Remote.Mode, if empty then use App.Mode
	mode := strings.TrimSpace(c.Remote.Mode)
	if mode == "" {
		mode = strings.TrimSpace(c.App.Mode)
	}
	// If still empty, use default value
	if mode == "" {
		mode = define.DEFAULT_MODE
	}
	// Redis enabled state: read from environment variable, default to enabled
	redisEnabled := true
	if redisEnabledEnv := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); redisEnabledEnv != "" {
		redisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	}

	return &CmdConfigData{
		Port:             c.Server.Port,
		Redis:            c.Redis.Addr,
		RedisPassword:    redisPassword,
		RedisEnabled:     redisEnabled,
		RemoteConfig:     c.Remote.URL,
		RemoteKey:        c.Remote.Key,
		TaskInterval:     int(c.Task.Interval.Seconds()),
		Mode:             mode,
		HTTPTimeout:      int(c.HTTP.Timeout.Seconds()),
		HTTPMaxIdleConns: c.HTTP.MaxIdleConns,
		HTTPInsecureTLS:  c.HTTP.InsecureTLS,
		APIKey:           c.App.APIKey,
	}
}
