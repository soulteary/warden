// Package cmd provides command-line argument parsing and configuration management functionality.
// Supports loading configuration from command-line arguments, environment variables and configuration files.
package cmd

import (
	// Standard library
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// Internal packages
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/errors"
)

// Config stores application configuration
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Config struct {
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

// GetArgs parses command-line arguments and environment variables, returns configuration struct
// Priority: command-line arguments > environment variables > configuration file > default values
// If -config-file parameter is provided, will attempt to load from configuration file
func GetArgs() *Config {
	// Create FlagSet to parse command-line arguments
	// Need to define all possible parameters to avoid "flag provided but not defined" error
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var configFileFlag string
	var portFlag int
	var redisFlag, configFlag, keyFlag, modeFlag string
	var intervalFlag int
	var httpTimeoutFlag, httpMaxIdleConnsFlag int
	var httpInsecureTLSFlag bool
	var redisPasswordFlag string
	var redisEnabledFlag bool
	var apiKeyFlag string

	fs.StringVar(&configFileFlag, "config-file", "", "Configuration file path (supports YAML format)")
	// Define all parameters to avoid undefined parameter errors (but only used for parsing here, actual values processed later)
	fs.IntVar(&portFlag, "port", 0, "web port")
	fs.StringVar(&redisFlag, "redis", "", "redis host and port")
	fs.StringVar(&redisPasswordFlag, "redis-password", "", "redis password")
	fs.BoolVar(&redisEnabledFlag, "redis-enabled", true, "enable Redis (default: true)")
	fs.StringVar(&configFlag, "config", "", "remote config url")
	fs.StringVar(&keyFlag, "key", "", "remote config key")
	fs.StringVar(&modeFlag, "mode", "", "app mode")
	fs.IntVar(&intervalFlag, "interval", 0, "task interval")
	fs.IntVar(&httpTimeoutFlag, "http-timeout", 0, "HTTP request timeout in seconds")
	fs.IntVar(&httpMaxIdleConnsFlag, "http-max-idle-conns", 0, "HTTP max idle connections")
	fs.BoolVar(&httpInsecureTLSFlag, "http-insecure-tls", false, "skip TLS certificate verification (development only)")
	fs.StringVar(&apiKeyFlag, "api-key", "", "API key for authentication")

	// Parse once to get configuration file path
	if err := fs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// If configuration file is specified, attempt to load from it
	if configFileFlag != "" {
		if newCfg, err := config.LoadFromFile(configFileFlag); err == nil {
			// Successfully loaded configuration file, convert to old format and apply command-line argument overrides
			legacyCfg := newCfg.ToCmdConfig()
			// Command-line arguments have highest priority, will override values in configuration file
			overrideWithFlags(legacyCfg)
			return convertToConfig(legacyCfg)
		}
		// Configuration file loading failed, continue using original logic (backward compatibility)
	}

	// Original logic: load from command-line arguments and environment variables
	return getArgsFromFlags()
}

// processPortFromFlags processes port configuration
func processPortFromFlags(cfg *Config, fs *flag.FlagSet, portFlag int) {
	if hasFlag(fs, "port") {
		cfg.Port = strconv.Itoa(portFlag)
	} else if portEnv := os.Getenv("PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			cfg.Port = strconv.Itoa(port)
		}
	}
}

// processRedisFromFlags processes Redis configuration
func processRedisFromFlags(cfg *Config, fs *flag.FlagSet, redisFlag, redisPasswordFlag string, redisEnabledFlag bool) {
	// Check if Mode is ONLY_LOCAL (check both cfg.Mode and environment variable for safety)
	isOnlyLocalMode := false
	if cfg.Mode != "" {
		isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(cfg.Mode)) == "ONLY_LOCAL"
	} else if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
		isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(modeEnv)) == "ONLY_LOCAL"
	}

	// Process Redis enabled state (priority: command-line argument > environment variable > default value)
	if hasFlag(fs, "redis-enabled") {
		cfg.RedisEnabled = redisEnabledFlag
	} else if redisEnabledEnv := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); redisEnabledEnv != "" {
		// Supports true/false/1/0
		cfg.RedisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	} else {
		// Default behavior: disable Redis in ONLY_LOCAL mode, enable otherwise (backward compatibility)
		if isOnlyLocalMode {
			cfg.RedisEnabled = false
		} else {
			cfg.RedisEnabled = true
		}
	}

	if hasFlag(fs, "redis") {
		cfg.Redis = redisFlag
	} else if redisEnv := strings.TrimSpace(os.Getenv("REDIS")); redisEnv != "" {
		cfg.Redis = redisEnv
	}

	// Process Redis password (priority: environment variable > password file > command-line argument)
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))

	switch {
	case redisPasswordEnv != "":
		cfg.RedisPassword = redisPasswordEnv
	case redisPasswordFile != "":
		if password, err := readPasswordFromFile(redisPasswordFile); err == nil {
			cfg.RedisPassword = password
		}
	case hasFlag(fs, "redis-password"):
		cfg.RedisPassword = redisPasswordFlag
	}
}

// processRemoteConfigFromFlags processes remote configuration
func processRemoteConfigFromFlags(cfg *Config, fs *flag.FlagSet, configFlag, keyFlag string) {
	if hasFlag(fs, "config") {
		cfg.RemoteConfig = configFlag
	} else if configEnv := strings.TrimSpace(os.Getenv("CONFIG")); configEnv != "" {
		cfg.RemoteConfig = configEnv
	}

	if hasFlag(fs, "key") {
		cfg.RemoteKey = keyFlag
	} else if keyEnv := strings.TrimSpace(os.Getenv("KEY")); keyEnv != "" {
		cfg.RemoteKey = keyEnv
	}
}

// processTaskFromFlags processes task configuration
func processTaskFromFlags(cfg *Config, fs *flag.FlagSet, intervalFlag int) {
	if hasFlag(fs, "interval") {
		cfg.TaskInterval = intervalFlag
	} else if intervalEnv := os.Getenv("INTERVAL"); intervalEnv != "" {
		if interval, err := strconv.Atoi(intervalEnv); err == nil {
			cfg.TaskInterval = interval
		}
	}
}

// processModeFromFlags processes mode configuration
func processModeFromFlags(cfg *Config, fs *flag.FlagSet, modeFlag string) {
	if hasFlag(fs, "mode") {
		cfg.Mode = modeFlag
	} else if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
		cfg.Mode = modeEnv
	}
}

// processHTTPFromFlags processes HTTP configuration
func processHTTPFromFlags(cfg *Config, fs *flag.FlagSet, httpTimeoutFlag, httpMaxIdleConnsFlag int, httpInsecureTLSFlag bool) {
	if hasFlag(fs, "http-timeout") {
		cfg.HTTPTimeout = httpTimeoutFlag
	} else if timeoutEnv := os.Getenv("HTTP_TIMEOUT"); timeoutEnv != "" {
		// Supports two formats: integer seconds (e.g., "30") or duration format (e.g., "30s", "1m30s")
		if timeout, err := time.ParseDuration(timeoutEnv); err == nil {
			cfg.HTTPTimeout = int(timeout.Seconds())
		} else if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
			cfg.HTTPTimeout = timeout
		}
	}

	if hasFlag(fs, "http-max-idle-conns") {
		cfg.HTTPMaxIdleConns = httpMaxIdleConnsFlag
	} else if maxIdleConnsEnv := os.Getenv("HTTP_MAX_IDLE_CONNS"); maxIdleConnsEnv != "" {
		if maxIdleConns, err := strconv.Atoi(maxIdleConnsEnv); err == nil {
			cfg.HTTPMaxIdleConns = maxIdleConns
		}
	}

	if hasFlag(fs, "http-insecure-tls") {
		cfg.HTTPInsecureTLS = httpInsecureTLSFlag
	} else if insecureTLSEnv := os.Getenv("HTTP_INSECURE_TLS"); insecureTLSEnv != "" {
		cfg.HTTPInsecureTLS = strings.EqualFold(insecureTLSEnv, "true") || insecureTLSEnv == "1"
	}
}

// processAPIKeyFromFlags processes API Key configuration
func processAPIKeyFromFlags(cfg *Config, fs *flag.FlagSet, apiKeyFlag string) {
	if hasFlag(fs, "api-key") {
		cfg.APIKey = apiKeyFlag
	} else if apiKeyEnv := strings.TrimSpace(os.Getenv("API_KEY")); apiKeyEnv != "" {
		cfg.APIKey = apiKeyEnv
	}
}

// getArgsFromFlags loads configuration from command-line arguments and environment variables (original logic)
func getArgsFromFlags() *Config {
	cfg := &Config{
		Port:             strconv.Itoa(define.DEFAULT_PORT),
		Redis:            define.DEFAULT_REDIS,
		RedisEnabled:     true, // Default to enable Redis (backward compatibility)
		RemoteConfig:     define.DEFAULT_REMOTE_CONFIG,
		RemoteKey:        define.DEFAULT_REMOTE_KEY,
		TaskInterval:     define.DEFAULT_TASK_INTERVAL,
		Mode:             define.DEFAULT_MODE,
		HTTPTimeout:      define.DEFAULT_TIMEOUT,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// Create FlagSet to parse command-line arguments
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	var portFlag int
	var redisFlag, configFlag, keyFlag, modeFlag string
	var intervalFlag int
	var httpTimeoutFlag, httpMaxIdleConnsFlag int
	var httpInsecureTLSFlag bool
	var redisPasswordFlag string
	var redisEnabledFlag bool

	fs.IntVar(&portFlag, "port", define.DEFAULT_PORT, "web port")
	fs.StringVar(&redisFlag, "redis", define.DEFAULT_REDIS, "redis host and port")
	fs.StringVar(&redisPasswordFlag, "redis-password", "", "redis password")
	fs.BoolVar(&redisEnabledFlag, "redis-enabled", true, "enable Redis (default: true)")
	fs.StringVar(&configFlag, "config", define.DEFAULT_REMOTE_CONFIG, "remote config url")
	fs.StringVar(&keyFlag, "key", define.DEFAULT_REMOTE_KEY, "remote config key")
	fs.StringVar(&modeFlag, "mode", define.DEFAULT_MODE, "app mode")
	fs.IntVar(&intervalFlag, "interval", define.DEFAULT_TASK_INTERVAL, "task interval")
	fs.IntVar(&httpTimeoutFlag, "http-timeout", define.DEFAULT_TIMEOUT, "HTTP request timeout in seconds")
	fs.IntVar(&httpMaxIdleConnsFlag, "http-max-idle-conns", 100, "HTTP max idle connections")
	fs.BoolVar(&httpInsecureTLSFlag, "http-insecure-tls", false, "skip TLS certificate verification (development only)")
	var apiKeyFlag string
	fs.StringVar(&apiKeyFlag, "api-key", "", "API key for authentication")

	// Parse command-line arguments
	if err := fs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// Process each configuration item
	// Process Mode first, as it may affect other configurations (e.g., Redis in ONLY_LOCAL mode)
	processModeFromFlags(cfg, fs, modeFlag)
	processPortFromFlags(cfg, fs, portFlag)
	processRedisFromFlags(cfg, fs, redisFlag, redisPasswordFlag, redisEnabledFlag)
	processRemoteConfigFromFlags(cfg, fs, configFlag, keyFlag)
	processTaskFromFlags(cfg, fs, intervalFlag)
	processHTTPFromFlags(cfg, fs, httpTimeoutFlag, httpMaxIdleConnsFlag, httpInsecureTLSFlag)
	processAPIKeyFromFlags(cfg, fs, apiKeyFlag)

	return cfg
}

// hasFlag checks if command-line argument is set
func hasFlag(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// readPasswordFromFile reads password from file (security improvement)
// File path should be absolute path or relative to working directory
// File content will have leading and trailing whitespace trimmed
func readPasswordFromFile(filePath string) (string, error) {
	// Security check: ensure file path is relative or absolute path, prevent path traversal attacks
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Read file content
	// #nosec G304 -- file path has been validated via filepath.Abs, is safe
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	// Trim leading and trailing whitespace
	password := strings.TrimSpace(string(data))
	return password, nil
}

// convertToConfig converts internal configuration type to Config
func convertToConfig(cfg *config.CmdConfigData) *Config {
	return &Config{
		Port:             cfg.Port,
		Redis:            cfg.Redis,
		RedisPassword:    cfg.RedisPassword,
		RedisEnabled:     cfg.RedisEnabled,
		RemoteConfig:     cfg.RemoteConfig,
		RemoteKey:        cfg.RemoteKey,
		TaskInterval:     cfg.TaskInterval,
		Mode:             cfg.Mode,
		HTTPTimeout:      cfg.HTTPTimeout,
		HTTPMaxIdleConns: cfg.HTTPMaxIdleConns,
		HTTPInsecureTLS:  cfg.HTTPInsecureTLS,
		APIKey:           cfg.APIKey,
	}
}

// overrideWithFlags overrides configuration with command-line arguments (command-line arguments have highest priority)
func overrideWithFlags(cfg *config.CmdConfigData) {
	// Create new FlagSet to parse command-line arguments (avoid duplicate flag definitions)
	overrideFs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var portFlag int
	var redisFlag, configFlag, keyFlag, modeFlag string
	var intervalFlag int
	var httpTimeoutFlag, httpMaxIdleConnsFlag int
	var httpInsecureTLSFlag bool
	var redisPasswordFlag string
	var redisEnabledFlag bool
	var apiKeyFlag string

	overrideFs.IntVar(&portFlag, "port", 0, "web port")
	overrideFs.StringVar(&redisFlag, "redis", "", "redis host and port")
	overrideFs.StringVar(&redisPasswordFlag, "redis-password", "", "redis password")
	overrideFs.BoolVar(&redisEnabledFlag, "redis-enabled", true, "enable Redis (default: true)")
	overrideFs.StringVar(&configFlag, "config", "", "remote config url")
	overrideFs.StringVar(&keyFlag, "key", "", "remote config key")
	overrideFs.StringVar(&modeFlag, "mode", "", "app mode")
	overrideFs.IntVar(&intervalFlag, "interval", 0, "task interval")
	overrideFs.IntVar(&httpTimeoutFlag, "http-timeout", 0, "HTTP request timeout in seconds")
	overrideFs.IntVar(&httpMaxIdleConnsFlag, "http-max-idle-conns", 0, "HTTP max idle connections")
	overrideFs.BoolVar(&httpInsecureTLSFlag, "http-insecure-tls", false, "skip TLS certificate verification (development only)")
	overrideFs.StringVar(&apiKeyFlag, "api-key", "", "API key for authentication")

	if err := overrideFs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// If command-line arguments are set, override configuration
	if hasFlag(overrideFs, "port") && portFlag > 0 {
		cfg.Port = strconv.Itoa(portFlag)
	}
	if hasFlag(overrideFs, "redis") && redisFlag != "" {
		cfg.Redis = redisFlag
	}
	if hasFlag(overrideFs, "redis-password") && redisPasswordFlag != "" {
		cfg.RedisPassword = redisPasswordFlag
	}
	if hasFlag(overrideFs, "redis-enabled") {
		cfg.RedisEnabled = redisEnabledFlag
	}
	if hasFlag(overrideFs, "config") && configFlag != "" {
		cfg.RemoteConfig = configFlag
	}
	if hasFlag(overrideFs, "key") && keyFlag != "" {
		cfg.RemoteKey = keyFlag
	}
	if hasFlag(overrideFs, "mode") && modeFlag != "" {
		cfg.Mode = modeFlag
	}
	if hasFlag(overrideFs, "interval") && intervalFlag > 0 {
		cfg.TaskInterval = intervalFlag
	}
	if hasFlag(overrideFs, "http-timeout") && httpTimeoutFlag > 0 {
		cfg.HTTPTimeout = httpTimeoutFlag
	}
	if hasFlag(overrideFs, "http-max-idle-conns") && httpMaxIdleConnsFlag > 0 {
		cfg.HTTPMaxIdleConns = httpMaxIdleConnsFlag
	}
	if hasFlag(overrideFs, "http-insecure-tls") {
		cfg.HTTPInsecureTLS = httpInsecureTLSFlag
	}
	if hasFlag(overrideFs, "api-key") && apiKeyFlag != "" {
		cfg.APIKey = apiKeyFlag
	}

	// If Mode is ONLY_LOCAL and RedisEnabled was not explicitly set via command-line, disable Redis
	if !hasFlag(overrideFs, "redis-enabled") {
		isOnlyLocalMode := false
		if cfg.Mode != "" {
			isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(cfg.Mode)) == "ONLY_LOCAL"
		} else if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
			isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(modeEnv)) == "ONLY_LOCAL"
		}
		if isOnlyLocalMode {
			cfg.RedisEnabled = false
		}
	}
}

// LoadConfig loads configuration (new interface, supports configuration file)
// Priority: command-line arguments > environment variables > configuration file > default values
func LoadConfig(configFile string) (*Config, error) {
	// Try to load from configuration file
	if configFile != "" {
		newCfg, err := config.LoadFromFile(configFile)
		if err == nil {
			legacyCfg := newCfg.ToCmdConfig()
			// Apply environment variable overrides (environment variables have higher priority than configuration file)
			overrideFromEnvInternal(legacyCfg)
			return convertToConfig(legacyCfg), nil
		}
		// Configuration file loading failed, return error
		return nil, errors.ErrConfigLoad.WithError(err)
	}

	// No configuration file, use original logic
	cfg := getArgsFromFlags()
	return cfg, nil
}

// overrideFromEnvInternal overrides configuration from environment variables (internal version)
func overrideFromEnvInternal(cfg *config.CmdConfigData) {
	// Environment variables have higher priority than configuration file, but lower than command-line arguments
	// Only process environment variables here, command-line arguments are processed in GetArgs
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			cfg.Port = strconv.Itoa(port)
		}
	}

	if redisEnv := strings.TrimSpace(os.Getenv("REDIS")); redisEnv != "" {
		cfg.Redis = redisEnv
	}

	// Redis password priority: environment variable > password file > configuration file
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))

	if redisPasswordEnv != "" {
		cfg.RedisPassword = redisPasswordEnv
	} else if redisPasswordFile != "" {
		if password, err := readPasswordFromFile(redisPasswordFile); err == nil {
			cfg.RedisPassword = password
		}
	}

	if configEnv := strings.TrimSpace(os.Getenv("CONFIG")); configEnv != "" {
		cfg.RemoteConfig = configEnv
	}

	if keyEnv := strings.TrimSpace(os.Getenv("KEY")); keyEnv != "" {
		cfg.RemoteKey = keyEnv
	}

	if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
		cfg.Mode = modeEnv
	}

	if intervalEnv := os.Getenv("INTERVAL"); intervalEnv != "" {
		if interval, err := strconv.Atoi(intervalEnv); err == nil {
			cfg.TaskInterval = interval
		}
	}

	if timeoutEnv := os.Getenv("HTTP_TIMEOUT"); timeoutEnv != "" {
		// Supports two formats: integer seconds (e.g., "30") or duration format (e.g., "30s", "1m30s")
		if timeout, err := time.ParseDuration(timeoutEnv); err == nil {
			cfg.HTTPTimeout = int(timeout.Seconds())
		} else if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
			cfg.HTTPTimeout = timeout
		}
	}

	if maxIdleConnsEnv := os.Getenv("HTTP_MAX_IDLE_CONNS"); maxIdleConnsEnv != "" {
		if maxIdleConns, err := strconv.Atoi(maxIdleConnsEnv); err == nil {
			cfg.HTTPMaxIdleConns = maxIdleConns
		}
	}

	if insecureTLSEnv := os.Getenv("HTTP_INSECURE_TLS"); insecureTLSEnv != "" {
		cfg.HTTPInsecureTLS = strings.EqualFold(insecureTLSEnv, "true") || insecureTLSEnv == "1"
	}

	if apiKeyEnv := strings.TrimSpace(os.Getenv("API_KEY")); apiKeyEnv != "" {
		cfg.APIKey = apiKeyEnv
	}

	// Redis enabled state
	// Check if Mode is ONLY_LOCAL (check both cfg.Mode and environment variable)
	isOnlyLocalMode := false
	if cfg.Mode != "" {
		isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(cfg.Mode)) == "ONLY_LOCAL"
	} else if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
		isOnlyLocalMode = strings.ToUpper(strings.TrimSpace(modeEnv)) == "ONLY_LOCAL"
	}

	if redisEnabledEnv := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); redisEnabledEnv != "" {
		// Explicitly set via environment variable
		cfg.RedisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	} else if isOnlyLocalMode {
		// In ONLY_LOCAL mode, disable Redis by default if not explicitly enabled
		cfg.RedisEnabled = false
	}
	// Note: If not ONLY_LOCAL mode and REDIS_ENABLED is not set, keep the value from config file
}
