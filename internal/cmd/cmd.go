// Package cmd provides command-line argument parsing and configuration management functionality.
// Supports loading configuration from command-line arguments, environment variables and configuration files.
package cmd

import (
	// Standard library
	"flag"
	"os"
	"strconv"
	"strings"

	// External packages
	"github.com/soulteary/cli-kit/configutil"
	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/cli-kit/flagutil"

	// Internal packages
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/errors"
)

// Config stores application configuration
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Config struct {
	Port                 string // 16 bytes
	Redis                string // 16 bytes
	RedisPassword        string // 16 bytes
	RedisEnabled         bool   // 1 byte (padding to 8 bytes)
	RemoteConfig         string // 16 bytes
	RemoteKey            string // 16 bytes
	Mode                 string // 16 bytes
	APIKey               string // 16 bytes
	DataFile             string // 16 bytes - local user data file path
	TaskInterval         int    // 8 bytes
	HTTPTimeout          int    // 8 bytes
	HTTPMaxIdleConns     int    // 8 bytes
	HTTPInsecureTLS      bool   // 1 byte (padding to 8 bytes)
	HMACKeys             string // JSON map key_id -> secret (env WARDEN_HMAC_KEYS)
	HMACToleranceSec     int    // env WARDEN_HMAC_TIMESTAMP_TOLERANCE, default 60
	TLSCertFile          string // env WARDEN_TLS_CERT
	TLSKeyFile           string // env WARDEN_TLS_KEY
	TLSCAFile            string // env WARDEN_TLS_CA (client CA for mTLS)
	TLSRequireClientCert bool   // env WARDEN_TLS_REQUIRE_CLIENT_CERT
}

// flagValues holds parsed flag values
//
//nolint:govet // fieldalignment: field order optimized for memory efficiency
type flagValues struct {
	// int fields (8 bytes each)
	port             int
	interval         int
	httpTimeout      int
	httpMaxIdleConns int
	// string fields (16 bytes each)
	configFile    string
	redis         string
	redisPassword string
	config        string
	key           string
	mode          string
	apiKey        string
	dataFile      string
	// bool fields (1 byte each, but padded to 8 bytes)
	redisEnabled    bool
	httpInsecureTLS bool
}

// flagDefaults holds default values for flags
//
//nolint:govet // fieldalignment: field order optimized for memory efficiency
type flagDefaults struct {
	// int fields (8 bytes each)
	port             int
	interval         int
	httpTimeout      int
	httpMaxIdleConns int
	// string fields (16 bytes each)
	redis    string
	config   string
	key      string
	mode     string
	dataFile string
	// bool fields (1 byte each, but padded to 8 bytes)
	redisEnabled    bool
	httpInsecureTLS bool
}

// registerFlags registers all flags in a FlagSet and returns parsed values
// If defaults is nil, uses zero values (for override scenarios)
func registerFlags(fs *flag.FlagSet, defaults *flagDefaults) *flagValues {
	vals := &flagValues{}

	// Determine default values
	portDef := 0
	redisDef := ""
	redisEnabledDef := true
	configDef := ""
	keyDef := ""
	modeDef := ""
	dataFileDef := ""
	intervalDef := 0
	httpTimeoutDef := 0
	httpMaxIdleConnsDef := 0
	httpInsecureTLSDef := false

	if defaults != nil {
		portDef = defaults.port
		redisDef = defaults.redis
		redisEnabledDef = defaults.redisEnabled
		configDef = defaults.config
		keyDef = defaults.key
		modeDef = defaults.mode
		dataFileDef = defaults.dataFile
		intervalDef = defaults.interval
		httpTimeoutDef = defaults.httpTimeout
		httpMaxIdleConnsDef = defaults.httpMaxIdleConns
		httpInsecureTLSDef = defaults.httpInsecureTLS
	}

	fs.StringVar(&vals.configFile, "config-file", "", "Configuration file path (supports YAML format)")
	fs.IntVar(&vals.port, "port", portDef, "web port")
	fs.StringVar(&vals.redis, "redis", redisDef, "redis host and port")
	fs.StringVar(&vals.redisPassword, "redis-password", "", "redis password")
	fs.BoolVar(&vals.redisEnabled, "redis-enabled", redisEnabledDef, "enable Redis (default: true)")
	fs.StringVar(&vals.config, "config", configDef, "remote config url")
	fs.StringVar(&vals.key, "key", keyDef, "remote config key")
	fs.StringVar(&vals.mode, "mode", modeDef, "app mode")
	fs.IntVar(&vals.interval, "interval", intervalDef, "task interval")
	fs.IntVar(&vals.httpTimeout, "http-timeout", httpTimeoutDef, "HTTP request timeout in seconds")
	fs.IntVar(&vals.httpMaxIdleConns, "http-max-idle-conns", httpMaxIdleConnsDef, "HTTP max idle connections")
	fs.BoolVar(&vals.httpInsecureTLS, "http-insecure-tls", httpInsecureTLSDef, "skip TLS certificate verification (development only)")
	fs.StringVar(&vals.apiKey, "api-key", "", "API key for authentication")
	fs.StringVar(&vals.dataFile, "data-file", dataFileDef, "local user data file path")

	return vals
}

// GetArgs parses command-line arguments and environment variables, returns configuration struct
// Priority: command-line arguments > environment variables > configuration file > default values
// If -config-file parameter is provided, will attempt to load from configuration file
func GetArgs() *Config {
	// Create FlagSet to parse command-line arguments
	// Need to define all possible parameters to avoid "flag provided but not defined" error
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	// Use zero defaults for override scenario
	flagVals := registerFlags(fs, nil)

	// Parse once to get configuration file path
	if err := fs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// If configuration file is specified, attempt to load from it
	if flagVals.configFile != "" {
		if newCfg, err := config.LoadFromFile(flagVals.configFile); err == nil {
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
func processPortFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default port is already set in cfg, so we use 0 as default here and only override if set
	if portVal := configutil.ResolveIntAsString(fs, "port", "PORT", 0, false); portVal != "0" {
		cfg.Port = portVal
	}
}

// processRedisFromFlags processes Redis configuration
// flagVals can be nil when only processing environment variables (e.g., in overrideFromEnvInternal)
func processRedisFromFlags(cfg *Config, fs *flag.FlagSet, flagVals *flagValues) {
	// Check if Mode is ONLY_LOCAL (check both cfg.Mode and environment variable for safety)
	isOnlyLocalMode := false
	if cfg.Mode != "" {
		isOnlyLocalMode = strings.EqualFold(strings.TrimSpace(cfg.Mode), "ONLY_LOCAL")
	} else if modeEnv := env.GetTrimmed("MODE", ""); modeEnv != "" {
		isOnlyLocalMode = strings.EqualFold(modeEnv, "ONLY_LOCAL")
	}

	// Check if Redis address is explicitly set (command-line argument or environment variable)
	redisExplicitlySet := (flagVals != nil && flagutil.HasFlag(fs, "redis")) || env.GetTrimmed("REDIS", "") != ""

	// Process Redis address first
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default is already set in cfg, so we only override if explicitly set
	if redisVal := configutil.ResolveString(fs, "redis", "REDIS", "", true); redisVal != "" {
		cfg.Redis = redisVal
	}

	// Process Redis enabled state (priority: command-line argument > environment variable > default value)
	if flagVals != nil && flagutil.HasFlag(fs, "redis-enabled") {
		cfg.RedisEnabled = flagVals.redisEnabled
	} else if redisEnabledEnv := env.GetTrimmed("REDIS_ENABLED", ""); redisEnabledEnv != "" {
		// Supports true/false/1/0
		cfg.RedisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	} else {
		// Default behavior:
		// - If Redis address is explicitly set, enable Redis (user intent to use Redis)
		// - In ONLY_LOCAL mode without explicit Redis address, disable Redis by default
		// - Otherwise, enable Redis (backward compatibility)
		switch {
		case redisExplicitlySet:
			// User explicitly set Redis address, enable Redis
			cfg.RedisEnabled = true
		case isOnlyLocalMode:
			// ONLY_LOCAL mode without explicit Redis address, disable Redis by default
			cfg.RedisEnabled = false
		default:
			// Other modes, enable Redis by default
			cfg.RedisEnabled = true
		}
	}

	// Process Redis password (priority: environment variable > password file > command-line argument)
	redisPasswordEnv := env.GetTrimmed("REDIS_PASSWORD", "")
	redisPasswordFile := env.GetTrimmed("REDIS_PASSWORD_FILE", "")

	switch {
	case redisPasswordEnv != "":
		cfg.RedisPassword = redisPasswordEnv
	case redisPasswordFile != "":
		if password, err := flagutil.ReadPasswordFromFile(redisPasswordFile); err == nil {
			cfg.RedisPassword = password
		}
	case flagVals != nil && flagutil.HasFlag(fs, "redis-password"):
		cfg.RedisPassword = flagVals.redisPassword
	}
}

// processRemoteConfigFromFlags processes remote configuration
func processRemoteConfigFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default is already set in cfg, so we only override if explicitly set
	if configVal := configutil.ResolveString(fs, "config", "CONFIG", "", true); configVal != "" {
		cfg.RemoteConfig = configVal
	}

	if keyVal := configutil.ResolveString(fs, "key", "KEY", "", true); keyVal != "" {
		cfg.RemoteKey = keyVal
	}
}

// processTaskFromFlags processes task configuration
func processTaskFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default is already set in cfg, so we use it as fallback
	cfg.TaskInterval = configutil.ResolveInt(fs, "interval", "INTERVAL", cfg.TaskInterval, false)
}

// processModeFromFlags processes mode configuration
func processModeFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default is already set in cfg, so we only override if explicitly set
	if modeVal := configutil.ResolveString(fs, "mode", "MODE", "", true); modeVal != "" {
		cfg.Mode = modeVal
	}
}

// processHTTPFromFlags processes HTTP configuration
// flagVals can be nil when only processing environment variables (e.g., in overrideFromEnvInternal)
func processHTTPFromFlags(cfg *Config, fs *flag.FlagSet, flagVals *flagValues) {
	// HTTP_TIMEOUT: CLI flag has highest priority
	if flagVals != nil && flagutil.HasFlag(fs, "http-timeout") {
		cfg.HTTPTimeout = flagVals.httpTimeout
	} else if env.Has("HTTP_TIMEOUT") {
		// Supports two formats: integer seconds (e.g., "30") or duration format (e.g., "30s", "1m30s")
		if timeout := env.GetDuration("HTTP_TIMEOUT", 0); timeout > 0 {
			cfg.HTTPTimeout = int(timeout.Seconds())
		} else if timeout := env.GetInt("HTTP_TIMEOUT", 0); timeout > 0 {
			cfg.HTTPTimeout = timeout
		}
	}
	// Note: Default is already set in cfg, so we don't need to set it here

	// Use configutil for http-max-idle-conns
	cfg.HTTPMaxIdleConns = configutil.ResolveInt(fs, "http-max-idle-conns", "HTTP_MAX_IDLE_CONNS", cfg.HTTPMaxIdleConns, false)

	// Use configutil for http-insecure-tls
	cfg.HTTPInsecureTLS = configutil.ResolveBool(fs, "http-insecure-tls", "HTTP_INSECURE_TLS", cfg.HTTPInsecureTLS)
}

// processAPIKeyFromFlags processes API Key configuration
func processAPIKeyFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	// Default is empty string, so we only override if explicitly set
	if apiKeyVal := configutil.ResolveString(fs, "api-key", "API_KEY", "", true); apiKeyVal != "" {
		cfg.APIKey = apiKeyVal
	}
}

// processDataFileFromFlags processes local data file path configuration
func processDataFileFromFlags(cfg *Config, fs *flag.FlagSet) {
	// Use configutil to resolve with priority: CLI > ENV > default
	cfg.DataFile = configutil.ResolveString(fs, "data-file", "DATA_FILE", define.DEFAULT_DATA_FILE, true)
	if cfg.DataFile == "" {
		cfg.DataFile = define.DEFAULT_DATA_FILE
	}
}

// processServiceAuthFromEnv reads service-to-service auth config from env (no CLI flags).
const (
	defaultHMACToleranceSec = 60
)

func processServiceAuthFromEnv(cfg *Config) {
	if v := env.GetTrimmed("WARDEN_HMAC_KEYS", ""); v != "" {
		cfg.HMACKeys = v
	}
	if v := env.GetInt("WARDEN_HMAC_TIMESTAMP_TOLERANCE", 0); v > 0 {
		cfg.HMACToleranceSec = v
	} else if cfg.HMACToleranceSec <= 0 && cfg.HMACKeys != "" {
		cfg.HMACToleranceSec = defaultHMACToleranceSec
	}
	if v := env.GetTrimmed("WARDEN_TLS_CERT", ""); v != "" {
		cfg.TLSCertFile = v
	}
	if v := env.GetTrimmed("WARDEN_TLS_KEY", ""); v != "" {
		cfg.TLSKeyFile = v
	}
	if v := env.GetTrimmed("WARDEN_TLS_CA", ""); v != "" {
		cfg.TLSCAFile = v
	}
	if v := env.GetTrimmed("WARDEN_TLS_REQUIRE_CLIENT_CERT", ""); v != "" {
		cfg.TLSRequireClientCert = strings.EqualFold(v, "true") || v == "1"
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
		DataFile:         define.DEFAULT_DATA_FILE,
		HTTPTimeout:      define.DEFAULT_TIMEOUT,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// Create FlagSet to parse command-line arguments
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Register flags with default values
	defaults := &flagDefaults{
		port:             define.DEFAULT_PORT,
		redis:            define.DEFAULT_REDIS,
		redisEnabled:     true,
		config:           define.DEFAULT_REMOTE_CONFIG,
		key:              define.DEFAULT_REMOTE_KEY,
		mode:             define.DEFAULT_MODE,
		dataFile:         define.DEFAULT_DATA_FILE,
		interval:         define.DEFAULT_TASK_INTERVAL,
		httpTimeout:      define.DEFAULT_TIMEOUT,
		httpMaxIdleConns: 100,
		httpInsecureTLS:  false,
	}
	flagVals := registerFlags(fs, defaults)

	// Parse command-line arguments
	if err := fs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// Process each configuration item
	// Process Mode first, as it may affect other configurations (e.g., Redis in ONLY_LOCAL mode)
	processModeFromFlags(cfg, fs)
	processPortFromFlags(cfg, fs)
	processRedisFromFlags(cfg, fs, flagVals)
	processRemoteConfigFromFlags(cfg, fs)
	processTaskFromFlags(cfg, fs)
	processHTTPFromFlags(cfg, fs, flagVals)
	processAPIKeyFromFlags(cfg, fs)
	processDataFileFromFlags(cfg, fs)
	processServiceAuthFromEnv(cfg)

	return cfg
}

// convertToConfig converts internal configuration type to Config
func convertToConfig(cfg *config.CmdConfigData) *Config {
	return &Config{
		Port:                 cfg.Port,
		Redis:                cfg.Redis,
		RedisPassword:        cfg.RedisPassword,
		RedisEnabled:         cfg.RedisEnabled,
		RemoteConfig:         cfg.RemoteConfig,
		RemoteKey:            cfg.RemoteKey,
		TaskInterval:         cfg.TaskInterval,
		Mode:                 cfg.Mode,
		DataFile:             cfg.DataFile,
		HTTPTimeout:          cfg.HTTPTimeout,
		HTTPMaxIdleConns:     cfg.HTTPMaxIdleConns,
		HTTPInsecureTLS:      cfg.HTTPInsecureTLS,
		APIKey:               cfg.APIKey,
		HMACKeys:             cfg.HMACKeys,
		HMACToleranceSec:     cfg.HMACToleranceSec,
		TLSCertFile:          cfg.TLSCertFile,
		TLSKeyFile:           cfg.TLSKeyFile,
		TLSCAFile:            cfg.TLSCAFile,
		TLSRequireClientCert: cfg.TLSRequireClientCert,
	}
}

// overrideWithFlags overrides configuration with command-line arguments (command-line arguments have highest priority)
func overrideWithFlags(cfg *config.CmdConfigData) {
	// Create new FlagSet to parse command-line arguments (avoid duplicate flag definitions)
	overrideFs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	// Use zero defaults for override scenario
	flagVals := registerFlags(overrideFs, nil)

	if err := overrideFs.Parse(os.Args[1:]); err != nil {
		// Ignore parsing errors, continue using default values
		_ = err // Explicitly ignore error
	}

	// Convert CmdConfigData to Config for processing
	tempCfg := &Config{
		Port:             cfg.Port,
		Redis:            cfg.Redis,
		RedisPassword:    cfg.RedisPassword,
		RedisEnabled:     cfg.RedisEnabled,
		RemoteConfig:     cfg.RemoteConfig,
		RemoteKey:        cfg.RemoteKey,
		Mode:             cfg.Mode,
		APIKey:           cfg.APIKey,
		DataFile:         cfg.DataFile,
		TaskInterval:     cfg.TaskInterval,
		HTTPTimeout:      cfg.HTTPTimeout,
		HTTPMaxIdleConns: cfg.HTTPMaxIdleConns,
		HTTPInsecureTLS:  cfg.HTTPInsecureTLS,
	}

	// Process each configuration item using unified processing functions
	// Process Mode first, as it may affect other configurations (e.g., Redis in ONLY_LOCAL mode)
	processModeFromFlags(tempCfg, overrideFs)
	processPortFromFlags(tempCfg, overrideFs)
	processRedisFromFlags(tempCfg, overrideFs, flagVals)
	processRemoteConfigFromFlags(tempCfg, overrideFs)
	processTaskFromFlags(tempCfg, overrideFs)
	processHTTPFromFlags(tempCfg, overrideFs, flagVals)
	processAPIKeyFromFlags(tempCfg, overrideFs)
	processDataFileFromFlags(tempCfg, overrideFs)

	// Copy back to CmdConfigData
	cfg.Port = tempCfg.Port
	cfg.Redis = tempCfg.Redis
	cfg.RedisPassword = tempCfg.RedisPassword
	cfg.RedisEnabled = tempCfg.RedisEnabled
	cfg.RemoteConfig = tempCfg.RemoteConfig
	cfg.RemoteKey = tempCfg.RemoteKey
	cfg.Mode = tempCfg.Mode
	cfg.APIKey = tempCfg.APIKey
	cfg.DataFile = tempCfg.DataFile
	cfg.TaskInterval = tempCfg.TaskInterval
	cfg.HTTPTimeout = tempCfg.HTTPTimeout
	cfg.HTTPMaxIdleConns = tempCfg.HTTPMaxIdleConns
	cfg.HTTPInsecureTLS = tempCfg.HTTPInsecureTLS
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
	// Use configutil with empty FlagSet (no CLI flags) to only check ENV
	emptyFs := flag.NewFlagSet("empty", flag.ContinueOnError)

	// Convert CmdConfigData to Config for processing
	tempCfg := &Config{
		Port:                 cfg.Port,
		Redis:                cfg.Redis,
		RedisPassword:        cfg.RedisPassword,
		RedisEnabled:         cfg.RedisEnabled,
		RemoteConfig:         cfg.RemoteConfig,
		RemoteKey:            cfg.RemoteKey,
		Mode:                 cfg.Mode,
		APIKey:               cfg.APIKey,
		DataFile:             cfg.DataFile,
		TaskInterval:         cfg.TaskInterval,
		HTTPTimeout:          cfg.HTTPTimeout,
		HTTPMaxIdleConns:     cfg.HTTPMaxIdleConns,
		HTTPInsecureTLS:      cfg.HTTPInsecureTLS,
		HMACKeys:             cfg.HMACKeys,
		HMACToleranceSec:     cfg.HMACToleranceSec,
		TLSCertFile:          cfg.TLSCertFile,
		TLSKeyFile:           cfg.TLSKeyFile,
		TLSCAFile:            cfg.TLSCAFile,
		TLSRequireClientCert: cfg.TLSRequireClientCert,
	}

	// Process each configuration item using unified processing functions
	// Pass nil for flagVals since we're only processing environment variables
	processModeFromFlags(tempCfg, emptyFs)
	processPortFromFlags(tempCfg, emptyFs)
	processRedisFromFlags(tempCfg, emptyFs, nil)
	processRemoteConfigFromFlags(tempCfg, emptyFs)
	processTaskFromFlags(tempCfg, emptyFs)
	processHTTPFromFlags(tempCfg, emptyFs, nil)
	processAPIKeyFromFlags(tempCfg, emptyFs)
	processDataFileFromFlags(tempCfg, emptyFs)
	processServiceAuthFromEnv(tempCfg)

	// Copy back to CmdConfigData
	cfg.Port = tempCfg.Port
	cfg.Redis = tempCfg.Redis
	cfg.RedisPassword = tempCfg.RedisPassword
	cfg.RedisEnabled = tempCfg.RedisEnabled
	cfg.RemoteConfig = tempCfg.RemoteConfig
	cfg.RemoteKey = tempCfg.RemoteKey
	cfg.Mode = tempCfg.Mode
	cfg.APIKey = tempCfg.APIKey
	cfg.DataFile = tempCfg.DataFile
	cfg.TaskInterval = tempCfg.TaskInterval
	cfg.HTTPTimeout = tempCfg.HTTPTimeout
	cfg.HTTPMaxIdleConns = tempCfg.HTTPMaxIdleConns
	cfg.HTTPInsecureTLS = tempCfg.HTTPInsecureTLS
	cfg.HMACKeys = tempCfg.HMACKeys
	cfg.HMACToleranceSec = tempCfg.HMACToleranceSec
	cfg.TLSCertFile = tempCfg.TLSCertFile
	cfg.TLSKeyFile = tempCfg.TLSKeyFile
	cfg.TLSCAFile = tempCfg.TLSCAFile
	cfg.TLSRequireClientCert = tempCfg.TLSRequireClientCert
}
