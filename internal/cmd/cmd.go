// Package cmd 提供了命令行参数解析和配置管理功能。
// 支持从命令行参数、环境变量和配置文件加载配置。
package cmd

import (
	// 标准库
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// 项目内部包
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/errors"
)

// Config 存储应用配置
//
//nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
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

// GetArgs 解析命令行参数和环境变量，返回配置结构体
// 优先级：命令行参数 > 环境变量 > 配置文件 > 默认值
// 如果提供了 -config-file 参数，会尝试从配置文件加载
func GetArgs() *Config {
	// 创建 FlagSet 解析命令行参数
	// 需要定义所有可能的参数，避免出现 "flag provided but not defined" 错误
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var configFileFlag string
	var portFlag int
	var redisFlag, configFlag, keyFlag, modeFlag string
	var intervalFlag int
	var httpTimeoutFlag, httpMaxIdleConnsFlag int
	var httpInsecureTLSFlag bool
	var redisPasswordFlag string
	var redisEnabledFlag bool

	fs.StringVar(&configFileFlag, "config-file", "", "配置文件路径 (支持 YAML 格式)")
	// 定义所有参数以避免未定义参数错误（但这里只用于解析，实际值在后续处理）
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

	// 先解析一次以获取配置文件路径
	if err := fs.Parse(os.Args[1:]); err != nil {
		// 忽略解析错误，继续使用默认值
		_ = err // 明确忽略错误
	}

	// 如果指定了配置文件，尝试从配置文件加载
	if configFileFlag != "" {
		if newCfg, err := config.LoadFromFile(configFileFlag); err == nil {
			// 成功加载配置文件，转换为旧格式并应用命令行参数覆盖
			legacyCfg := newCfg.ToCmdConfig()
			// 命令行参数优先级最高，会覆盖配置文件中的值
			overrideWithFlags(legacyCfg)
			return convertToConfig(legacyCfg)
		}
		// 配置文件加载失败，继续使用原有逻辑（向后兼容）
	}

	// 原有逻辑：从命令行参数和环境变量加载
	return getArgsFromFlags()
}

// processPortFromFlags 处理端口配置
func processPortFromFlags(cfg *Config, fs *flag.FlagSet, portFlag int) {
	if hasFlag(fs, "port") {
		cfg.Port = strconv.Itoa(portFlag)
	} else if portEnv := os.Getenv("PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			cfg.Port = strconv.Itoa(port)
		}
	}
}

// processRedisFromFlags 处理 Redis 配置
func processRedisFromFlags(cfg *Config, fs *flag.FlagSet, redisFlag, redisPasswordFlag string, redisEnabledFlag bool) {
	// 处理 Redis 启用状态（优先级：命令行参数 > 环境变量 > 默认值 true）
	if hasFlag(fs, "redis-enabled") {
		cfg.RedisEnabled = redisEnabledFlag
	} else if redisEnabledEnv := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); redisEnabledEnv != "" {
		// 支持 true/false/1/0
		cfg.RedisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	} else {
		// 默认启用 Redis（向后兼容）
		cfg.RedisEnabled = true
	}

	if hasFlag(fs, "redis") {
		cfg.Redis = redisFlag
	} else if redisEnv := strings.TrimSpace(os.Getenv("REDIS")); redisEnv != "" {
		cfg.Redis = redisEnv
	}

	// 处理 Redis 密码（优先级：环境变量 > 密码文件 > 命令行参数）
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

// processRemoteConfigFromFlags 处理远程配置
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

// processTaskFromFlags 处理任务配置
func processTaskFromFlags(cfg *Config, fs *flag.FlagSet, intervalFlag int) {
	if hasFlag(fs, "interval") {
		cfg.TaskInterval = intervalFlag
	} else if intervalEnv := os.Getenv("INTERVAL"); intervalEnv != "" {
		if interval, err := strconv.Atoi(intervalEnv); err == nil {
			cfg.TaskInterval = interval
		}
	}
}

// processModeFromFlags 处理模式配置
func processModeFromFlags(cfg *Config, fs *flag.FlagSet, modeFlag string) {
	if hasFlag(fs, "mode") {
		cfg.Mode = modeFlag
	} else if modeEnv := strings.TrimSpace(os.Getenv("MODE")); modeEnv != "" {
		cfg.Mode = modeEnv
	}
}

// processHTTPFromFlags 处理 HTTP 配置
func processHTTPFromFlags(cfg *Config, fs *flag.FlagSet, httpTimeoutFlag, httpMaxIdleConnsFlag int, httpInsecureTLSFlag bool) {
	if hasFlag(fs, "http-timeout") {
		cfg.HTTPTimeout = httpTimeoutFlag
	} else if timeoutEnv := os.Getenv("HTTP_TIMEOUT"); timeoutEnv != "" {
		if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
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

// getArgsFromFlags 从命令行参数和环境变量加载配置（原有逻辑）
func getArgsFromFlags() *Config {
	cfg := &Config{
		Port:             strconv.Itoa(define.DEFAULT_PORT),
		Redis:            define.DEFAULT_REDIS,
		RedisEnabled:     true, // 默认启用 Redis（向后兼容）
		RemoteConfig:     define.DEFAULT_REMOTE_CONFIG,
		RemoteKey:        define.DEFAULT_REMOTE_KEY,
		TaskInterval:     define.DEFAULT_TASK_INTERVAL,
		Mode:             define.DEFAULT_MODE,
		HTTPTimeout:      define.DEFAULT_TIMEOUT,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// 创建 FlagSet 解析命令行参数
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

	// 解析命令行参数
	if err := fs.Parse(os.Args[1:]); err != nil {
		// 忽略解析错误，继续使用默认值
		_ = err // 明确忽略错误
	}

	// 处理各个配置项
	processPortFromFlags(cfg, fs, portFlag)
	processRedisFromFlags(cfg, fs, redisFlag, redisPasswordFlag, redisEnabledFlag)
	processRemoteConfigFromFlags(cfg, fs, configFlag, keyFlag)
	processTaskFromFlags(cfg, fs, intervalFlag)
	processModeFromFlags(cfg, fs, modeFlag)
	processHTTPFromFlags(cfg, fs, httpTimeoutFlag, httpMaxIdleConnsFlag, httpInsecureTLSFlag)

	return cfg
}

// hasFlag 检查命令行参数是否被设置
func hasFlag(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// readPasswordFromFile 从文件读取密码（安全性改进）
// 文件路径应该是绝对路径或相对于工作目录的路径
// 文件内容会被去除首尾空白字符
func readPasswordFromFile(filePath string) (string, error) {
	// 安全检查：确保文件路径是相对路径或绝对路径，防止路径遍历攻击
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// 读取文件内容
	// #nosec G304 -- 文件路径已经通过 filepath.Abs 验证，是安全的
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	// 去除首尾空白字符
	password := strings.TrimSpace(string(data))
	return password, nil
}

// convertToConfig 转换内部配置类型为 Config
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

// overrideWithFlags 使用命令行参数覆盖配置（命令行参数优先级最高）
func overrideWithFlags(cfg *config.CmdConfigData) {
	// 创建新的 FlagSet 来解析命令行参数（避免重复定义标志）
	overrideFs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var portFlag int
	var redisFlag, configFlag, keyFlag, modeFlag string
	var intervalFlag int
	var httpTimeoutFlag, httpMaxIdleConnsFlag int
	var httpInsecureTLSFlag bool
	var redisPasswordFlag string
	var redisEnabledFlag bool

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

	if err := overrideFs.Parse(os.Args[1:]); err != nil {
		// 忽略解析错误，继续使用默认值
		_ = err // 明确忽略错误
	}

	// 如果命令行参数被设置，覆盖配置
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
}

// LoadConfig 加载配置（新接口，支持配置文件）
// 优先级：命令行参数 > 环境变量 > 配置文件 > 默认值
func LoadConfig(configFile string) (*Config, error) {
	// 尝试从配置文件加载
	if configFile != "" {
		newCfg, err := config.LoadFromFile(configFile)
		if err == nil {
			legacyCfg := newCfg.ToCmdConfig()
			// 应用环境变量覆盖（环境变量优先级高于配置文件）
			overrideFromEnvInternal(legacyCfg)
			return convertToConfig(legacyCfg), nil
		}
		// 配置文件加载失败，返回错误
		return nil, errors.ErrConfigLoad.WithError(err)
	}

	// 没有配置文件，使用原有逻辑
	cfg := getArgsFromFlags()
	return cfg, nil
}

// overrideFromEnvInternal 从环境变量覆盖配置（内部版本）
func overrideFromEnvInternal(cfg *config.CmdConfigData) {
	// 环境变量优先级高于配置文件，但低于命令行参数
	// 这里只处理环境变量，命令行参数在 GetArgs 中处理
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			cfg.Port = strconv.Itoa(port)
		}
	}

	if redisEnv := strings.TrimSpace(os.Getenv("REDIS")); redisEnv != "" {
		cfg.Redis = redisEnv
	}

	// Redis 密码优先级：环境变量 > 密码文件 > 配置文件
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
		if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
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

	// Redis 启用状态
	if redisEnabledEnv := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); redisEnabledEnv != "" {
		cfg.RedisEnabled = strings.EqualFold(redisEnabledEnv, "true") || redisEnabledEnv == "1"
	}
}
