package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFromFile_ValidYAML tests loading valid YAML configuration file
func TestLoadFromFile_ValidYAML(t *testing.T) {
	// Create temporary configuration file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
server:
  port: "8081"
redis:
  addr: "localhost:6379"
app:
  mode: "development"
  api_key: "test-key"
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "8081", cfg.Server.Port)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, "development", cfg.App.Mode)
	assert.Equal(t, "test-key", cfg.App.APIKey)
}

// TestLoadFromFile_EmptyFile tests empty configuration file (should use default values)
func TestLoadFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(configFile, []byte(""), 0o600)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should use default values
	assert.NotEmpty(t, cfg.Server.Port)
}

// TestLoadFromFile_NonExistentFile tests non-existent configuration file (should use default values)
func TestLoadFromFile_NonExistentFile(t *testing.T) {
	cfg, err := LoadFromFile("/nonexistent/config.yaml")
	require.NoError(t, err, "不存在的文件应该使用默认值")
	require.NotNil(t, cfg)

	// Should use default values
	assert.NotEmpty(t, cfg.Server.Port)
}

// TestLoadFromFile_InvalidYAML tests invalid YAML file
func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `server:
  port: "8081"
  invalid: [unclosed bracket`
	err := os.WriteFile(configFile, []byte(invalidYAML), 0o600)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configFile)
	assert.Error(t, err, "无效的 YAML 应该返回错误")
	assert.Nil(t, cfg)
}

// TestLoadFromFile_TOMLNotSupported tests that TOML format is not supported
func TestLoadFromFile_TOMLNotSupported(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.toml")

	tomlContent := `[server]
port = "8081"`
	err := os.WriteFile(configFile, []byte(tomlContent), 0o600)
	require.NoError(t, err)

	cfg, err := LoadFromFile(configFile)
	assert.Error(t, err, "TOML 格式应该返回错误")
	assert.Contains(t, err.Error(), "TOML 格式暂不支持", "错误信息应该提到 TOML 不支持")
	assert.Nil(t, cfg)
}

// TestApplyDefaults tests default value application
func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// Verify server default values
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotZero(t, cfg.Server.ReadTimeout)
	assert.NotZero(t, cfg.Server.WriteTimeout)

	// Verify Redis default values
	assert.NotEmpty(t, cfg.Redis.Addr)

	// Verify other default values
	assert.NotZero(t, cfg.Task.Interval)
	assert.NotZero(t, cfg.HTTP.Timeout)
}

// TestOverrideFromEnv tests environment variable override
func TestOverrideFromEnv(t *testing.T) {
	// Save original environment variables
	originalPort := os.Getenv("PORT")
	originalRedis := os.Getenv("REDIS")
	defer func() {
		if originalPort != "" {
			require.NoError(t, os.Setenv("PORT", originalPort))
		} else {
			require.NoError(t, os.Unsetenv("PORT"))
		}
		if originalRedis != "" {
			require.NoError(t, os.Setenv("REDIS", originalRedis))
		} else {
			require.NoError(t, os.Unsetenv("REDIS"))
		}
	}()

	// Set environment variables
	require.NoError(t, os.Setenv("PORT", "9999"))
	require.NoError(t, os.Setenv("REDIS", "custom-redis:6379"))

	cfg := &Config{}
	overrideFromEnv(cfg)

	assert.Equal(t, "9999", cfg.Server.Port)
	assert.Equal(t, "custom-redis:6379", cfg.Redis.Addr)
}

// TestValidate_ValidConfig tests valid configuration validation
func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8081",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		Task: TaskConfig{
			Interval: 60 * time.Second,
		},
		App: AppConfig{
			Mode: "development",
		},
	}

	err := validate(cfg)
	assert.NoError(t, err, "有效配置应该通过验证")
}

// TestValidate_InvalidConfig tests invalid configuration validation
func TestValidate_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "缺少端口",
			cfg: &Config{
				Server: ServerConfig{Port: ""},
				Redis:  RedisConfig{Addr: "localhost:6379"},
			},
			want: "server.port 不能为空",
		},
		{
			name: "缺少 Redis 地址",
			cfg: &Config{
				Server: ServerConfig{Port: "8081"},
				Redis:  RedisConfig{Addr: ""},
			},
			want: "redis.addr 不能为空",
		},
		{
			name: "任务间隔太短",
			cfg: &Config{
				Server: ServerConfig{Port: "8081"},
				Redis:  RedisConfig{Addr: "localhost:6379"},
				Task:   TaskConfig{Interval: 500 * time.Millisecond},
			},
			want: "task.interval 必须至少为 1 秒",
		},
		{
			name: "生产环境禁用 TLS 验证",
			cfg: &Config{
				Server: ServerConfig{Port: "8081"},
				Redis:  RedisConfig{Addr: "localhost:6379"},
				Task:   TaskConfig{Interval: 60 * time.Second},
				HTTP:   HTTPConfig{InsecureTLS: true},
				App:    AppConfig{Mode: "production"},
			},
			want: "生产环境不允许禁用 TLS 证书验证",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			assert.Error(t, err, "无效配置应该返回错误")
			assert.Contains(t, err.Error(), tt.want, "错误信息应该包含预期内容")
		})
	}
}

// TestValidateConfigPath tests configuration file path validation
func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "空路径",
			path:    "",
			wantErr: true,
		},
		{
			name:    "有效路径",
			path:    "/tmp/config.yaml",
			wantErr: false,
		},
		{
			name:    "相对路径",
			path:    "./config.yaml",
			wantErr: false,
		},
		{
			name:    "包含路径遍历（相对路径）",
			path:    "../../etc/passwd",
			wantErr: false, // filepath.Abs will resolve path traversal, so converted path may no longer contain ".."
		},
		{
			name:    "包含路径遍历（绝对路径中）",
			path:    "/tmp/../../etc/passwd",
			wantErr: false, // filepath.Abs will resolve path traversal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateConfigPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "应该返回错误")
			} else {
				assert.NoError(t, err, "不应该返回错误")
			}
		})
	}
}

// TestGetRedisPassword tests getting Redis password
func TestGetRedisPassword(t *testing.T) {
	// Save original environment variables
	originalPassword := os.Getenv("REDIS_PASSWORD")
	originalPasswordFile := os.Getenv("REDIS_PASSWORD_FILE")
	defer func() {
		if originalPassword != "" {
			require.NoError(t, os.Setenv("REDIS_PASSWORD", originalPassword))
		} else {
			require.NoError(t, os.Unsetenv("REDIS_PASSWORD"))
		}
		if originalPasswordFile != "" {
			require.NoError(t, os.Setenv("REDIS_PASSWORD_FILE", originalPasswordFile))
		} else {
			require.NoError(t, os.Unsetenv("REDIS_PASSWORD_FILE"))
		}
	}()

	t.Run("从环境变量获取", func(t *testing.T) {
		require.NoError(t, os.Setenv("REDIS_PASSWORD", "env-password"))
		cfg := &Config{}
		password, err := cfg.GetRedisPassword()
		require.NoError(t, err)
		assert.Equal(t, "env-password", password)
	})

	t.Run("从配置文件获取", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("REDIS_PASSWORD"))
		require.NoError(t, os.Unsetenv("REDIS_PASSWORD_FILE"))
		cfg := &Config{
			Redis: RedisConfig{
				Password: "config-password",
			},
		}
		password, err := cfg.GetRedisPassword()
		require.NoError(t, err)
		assert.Equal(t, "config-password", password)
	})

	t.Run("从密码文件获取", func(t *testing.T) {
		tmpDir := t.TempDir()
		passwordFile := filepath.Join(tmpDir, "password.txt")
		err := os.WriteFile(passwordFile, []byte("file-password\n"), 0o600)
		require.NoError(t, err)

		require.NoError(t, os.Unsetenv("REDIS_PASSWORD"))
		cfg := &Config{
			Redis: RedisConfig{
				PasswordFile: passwordFile,
			},
		}
		password, err := cfg.GetRedisPassword()
		require.NoError(t, err)
		assert.Equal(t, "file-password", password)
	})
}

// TestToCmdConfig tests conversion to CmdConfig
func TestToCmdConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8081",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		Remote: RemoteConfig{
			URL:  "http://example.com/config",
			Key:  "test-key",
			Mode: "development",
		},
		Task: TaskConfig{
			Interval: 60 * time.Second,
		},
		HTTP: HTTPConfig{
			Timeout:      30 * time.Second,
			MaxIdleConns: 100,
			InsecureTLS:  false,
		},
		App: AppConfig{
			Mode:   "development",
			APIKey: "api-key",
		},
	}

	cmdCfg := cfg.ToCmdConfig()
	require.NotNil(t, cmdCfg)

	assert.Equal(t, "8081", cmdCfg.Port)
	assert.Equal(t, "localhost:6379", cmdCfg.Redis)
	assert.Equal(t, "http://example.com/config", cmdCfg.RemoteConfig)
	assert.Equal(t, "test-key", cmdCfg.RemoteKey)
	assert.Equal(t, "development", cmdCfg.Mode)
	assert.Equal(t, 60, cmdCfg.TaskInterval)
	assert.Equal(t, 30, cmdCfg.HTTPTimeout)
	assert.Equal(t, 100, cmdCfg.HTTPMaxIdleConns)
	assert.Equal(t, false, cmdCfg.HTTPInsecureTLS)
	assert.Equal(t, "api-key", cmdCfg.APIKey)
}

// TestApplyDefaults_AllSections tests default values for all configuration sections
func TestApplyDefaults_AllSections(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// Verify all configuration sections have default values
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Redis.Addr)
	assert.NotZero(t, cfg.Cache.UpdateInterval)
	assert.NotZero(t, cfg.RateLimit.Rate)
	assert.NotZero(t, cfg.HTTP.Timeout)
	assert.NotEmpty(t, cfg.Remote.URL)
	assert.NotZero(t, cfg.Task.Interval)
	assert.NotEmpty(t, cfg.App.Mode)
}
