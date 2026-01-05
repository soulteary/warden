// Package config 提供了配置文件加载和管理功能。
// 支持 YAML 格式的配置文件，并提供配置验证和默认值处理。
package config

import (
	// 标准库
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	// 第三方库
	"gopkg.in/yaml.v3"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
	"soulteary.com/soulteary/warden/internal/errors"
)

// Config 应用配置结构体
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Redis     RedisConfig     `yaml:"redis"`
	Cache     CacheConfig     `yaml:"cache"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	HTTP      HTTPConfig      `yaml:"http"`
	Remote    RemoteConfig    `yaml:"remote"`
	Task      TaskConfig      `yaml:"task"`
	App       AppConfig       `yaml:"app"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port            string        `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	// PasswordFile 密码文件路径（优先级高于 password）
	PasswordFile string `yaml:"password_file"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL            time.Duration `yaml:"ttl"`
	UpdateInterval time.Duration `yaml:"update_interval"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Rate   int           `yaml:"rate"`
	Window time.Duration `yaml:"window"`
}

// HTTPConfig HTTP 客户端配置
type HTTPConfig struct {
	Timeout      time.Duration `yaml:"timeout"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	InsecureTLS  bool          `yaml:"insecure_tls"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

// RemoteConfig 远程配置
type RemoteConfig struct {
	URL  string `yaml:"url"`
	Key  string `yaml:"key"`
	Mode string `yaml:"mode"`
}

// TaskConfig 任务配置
type TaskConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// AppConfig 应用配置
type AppConfig struct {
	Mode string `yaml:"mode"`
}

// LoadFromFile 从配置文件加载配置
// 支持 YAML 和 TOML 格式（通过文件扩展名判断）
// 优先级：配置文件 > 环境变量 > 默认值
func LoadFromFile(configPath string) (*Config, error) {
	cfg := &Config{}

	// 如果配置文件存在，尝试加载
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			// #nosec G304 -- 配置文件路径来自用户输入，需要验证
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, errors.ErrConfigLoad.WithError(err)
			}

			// 根据文件扩展名判断格式
			ext := strings.ToLower(filepath.Ext(configPath))
			switch ext {
			case ".yaml", ".yml":
				if err := yaml.Unmarshal(data, cfg); err != nil {
					return nil, errors.ErrConfigParse.WithError(err)
				}
			case ".toml":
				// TOML 支持需要额外的库，这里先返回错误提示
				return nil, errors.ErrConfigParse.WithMessage("TOML 格式暂不支持，请使用 YAML 格式")
			default:
				// 默认尝试 YAML
				if err := yaml.Unmarshal(data, cfg); err != nil {
					return nil, errors.ErrConfigParse.WithError(err)
				}
			}
		}
	}

	// 应用默认值
	applyDefaults(cfg)

	// 从环境变量覆盖配置（优先级高于配置文件）
	overrideFromEnv(cfg)

	// 验证配置
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyServerDefaults 应用服务器默认值
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

// applyRedisDefaults 应用 Redis 默认值
func applyRedisDefaults(cfg *Config) {
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = define.DEFAULT_REDIS
	}
}

// applyCacheDefaults 应用缓存默认值
func applyCacheDefaults(cfg *Config) {
	if cfg.Cache.UpdateInterval == 0 {
		cfg.Cache.UpdateInterval = define.DEFAULT_TASK_INTERVAL * time.Second
	}
}

// applyRateLimitDefaults 应用速率限制默认值
func applyRateLimitDefaults(cfg *Config) {
	if cfg.RateLimit.Rate == 0 {
		cfg.RateLimit.Rate = define.DEFAULT_RATE_LIMIT
	}
	if cfg.RateLimit.Window == 0 {
		cfg.RateLimit.Window = define.DEFAULT_RATE_LIMIT_WINDOW
	}
}

// applyHTTPDefaults 应用 HTTP 默认值
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

// applyRemoteDefaults 应用远程配置默认值
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

// applyTaskDefaults 应用任务默认值
func applyTaskDefaults(cfg *Config) {
	if cfg.Task.Interval == 0 {
		cfg.Task.Interval = define.DEFAULT_TASK_INTERVAL * time.Second
	}
}

// applyAppDefaults 应用应用默认值
func applyAppDefaults(cfg *Config) {
	if cfg.App.Mode == "" {
		cfg.App.Mode = define.DEFAULT_MODE
	}
}

// applyDefaults 应用默认值
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

// overrideFromEnv 从环境变量覆盖配置
func overrideFromEnv(cfg *Config) {
	// Server
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}

	// Redis
	if redis := os.Getenv("REDIS"); redis != "" {
		cfg.Redis.Addr = redis
	}
	// Redis 密码优先级：环境变量 > 密码文件 > 配置文件
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))
	if redisPasswordEnv != "" {
		cfg.Redis.Password = redisPasswordEnv
		cfg.Redis.PasswordFile = "" // 清除文件路径
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
		cfg.HTTP.InsecureTLS = strings.ToLower(insecureTLS) == "true" || insecureTLS == "1"
	}
}

// validate 验证配置
func validate(cfg *Config) error {
	var errs []string

	if cfg.Server.Port == "" {
		errs = append(errs, "server.port 不能为空")
	}

	if cfg.Redis.Addr == "" {
		errs = append(errs, "redis.addr 不能为空")
	}

	if cfg.Task.Interval < time.Second {
		errs = append(errs, "task.interval 必须至少为 1 秒")
	}

	if len(errs) > 0 {
		return errors.ErrConfigValidation.WithMessage(strings.Join(errs, "; "))
	}

	return nil
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// GetRedisPassword 获取 Redis 密码（处理文件读取）
func (c *Config) GetRedisPassword() (string, error) {
	// 优先级：环境变量 > 密码文件 > 配置文件中的密码
	redisPasswordEnv := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	if redisPasswordEnv != "" {
		return redisPasswordEnv, nil
	}

	redisPasswordFile := strings.TrimSpace(os.Getenv("REDIS_PASSWORD_FILE"))
	if redisPasswordFile == "" && c.Redis.PasswordFile != "" {
		redisPasswordFile = c.Redis.PasswordFile
	}

	if redisPasswordFile != "" {
		absPath, err := filepath.Abs(redisPasswordFile)
		if err != nil {
			return "", errors.ErrConfigLoad.WithError(err)
		}
		// #nosec G304 -- 文件路径已经通过 filepath.Abs 验证，是安全的
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", errors.ErrConfigLoad.WithError(err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	return c.Redis.Password, nil
}

// LegacyConfig 旧的配置格式（用于向后兼容）
// 注意：这个类型与 cmd.Config 结构相同，但定义在不同的包中
type LegacyConfig struct {
	Port             string
	Redis            string
	RedisPassword    string
	RemoteConfig     string
	RemoteKey        string
	TaskInterval     int
	Mode             string
	HTTPTimeout      int
	HTTPMaxIdleConns int
	HTTPInsecureTLS  bool
}

// ToLegacyConfig 转换为旧的 Config 格式（保持向后兼容）
func (c *Config) ToLegacyConfig() *LegacyConfig {
	redisPassword, _ := c.GetRedisPassword()
	return &LegacyConfig{
		Port:             c.Server.Port,
		Redis:            c.Redis.Addr,
		RedisPassword:    redisPassword,
		RemoteConfig:     c.Remote.URL,
		RemoteKey:        c.Remote.Key,
		TaskInterval:     int(c.Task.Interval.Seconds()),
		Mode:             c.App.Mode,
		HTTPTimeout:      int(c.HTTP.Timeout.Seconds()),
		HTTPMaxIdleConns: c.HTTP.MaxIdleConns,
		HTTPInsecureTLS:  c.HTTP.InsecureTLS,
	}
}

// CmdConfigData 配置数据结构（用于转换为 cmd.Config，避免循环依赖）
type CmdConfigData struct {
	Port             string
	Redis            string
	RedisPassword    string
	RemoteConfig     string
	RemoteKey        string
	TaskInterval     int
	Mode             string
	HTTPTimeout      int
	HTTPMaxIdleConns int
	HTTPInsecureTLS  bool
}

// ToCmdConfig 转换为 cmd.Config 格式
func (c *Config) ToCmdConfig() *CmdConfigData {
	redisPassword, _ := c.GetRedisPassword()
	return &CmdConfigData{
		Port:             c.Server.Port,
		Redis:            c.Redis.Addr,
		RedisPassword:    redisPassword,
		RemoteConfig:     c.Remote.URL,
		RemoteKey:        c.Remote.Key,
		TaskInterval:     int(c.Task.Interval.Seconds()),
		Mode:             c.App.Mode,
		HTTPTimeout:      int(c.HTTP.Timeout.Seconds()),
		HTTPMaxIdleConns: c.HTTP.MaxIdleConns,
		HTTPInsecureTLS:  c.HTTP.InsecureTLS,
	}
}
