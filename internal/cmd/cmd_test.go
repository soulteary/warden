package cmd

import (
	"os"
	"strconv"
	"testing"

	"github.com/soulteary/cli-kit/flagutil"
	"github.com/soulteary/cli-kit/testutil"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetArgs_DefaultValues(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test"}

	cfg := GetArgs()

	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port)
	assert.Equal(t, define.DEFAULT_REDIS, cfg.Redis)
	assert.Equal(t, define.DEFAULT_REMOTE_CONFIG, cfg.RemoteConfig)
	assert.Equal(t, define.DEFAULT_REMOTE_KEY, cfg.RemoteKey)
	assert.Equal(t, define.DEFAULT_TASK_INTERVAL, cfg.TaskInterval)
	assert.Equal(t, define.DEFAULT_MODE, cfg.Mode)
}

func TestGetArgs_WithCommandLineArgs(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	// Set command-line arguments
	os.Args = []string{"test", "--port", "9090", "--redis", "127.0.0.1:6380", "--config", "http://example.com/config", "--key", "test-key", "--mode", "ONLY_LOCAL", "--interval", "10"}

	cfg := GetArgs()

	assert.Equal(t, "9090", cfg.Port, "端口应该匹配")
	assert.Equal(t, "127.0.0.1:6380", cfg.Redis, "Redis地址应该匹配")
	assert.Equal(t, "http://example.com/config", cfg.RemoteConfig, "配置URL应该匹配")
	assert.Equal(t, "test-key", cfg.RemoteKey, "密钥应该匹配")
	assert.Equal(t, "ONLY_LOCAL", cfg.Mode, "模式应该匹配")
	assert.Equal(t, 10, cfg.TaskInterval, "间隔应该匹配")
}

func TestGetArgs_WithEnvVars(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Set environment variables
	envVars := map[string]string{
		"PORT":     "8888",
		"REDIS":    "192.168.1.1:6379",
		"CONFIG":   "http://test.com/data.json",
		"KEY":      "env-key",
		"INTERVAL": "15",
		"MODE":     "REMOTE_FIRST",
	}
	if err := envMgr.SetMultiple(envVars); err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}

	os.Args = []string{"test"}

	cfg := GetArgs()

	assert.Equal(t, "8888", cfg.Port, "端口应该匹配环境变量")
	assert.Equal(t, "192.168.1.1:6379", cfg.Redis, "Redis地址应该匹配环境变量")
	assert.Equal(t, "http://test.com/data.json", cfg.RemoteConfig, "配置URL应该匹配环境变量")
	assert.Equal(t, "env-key", cfg.RemoteKey, "密钥应该匹配环境变量")
	assert.Equal(t, "REMOTE_FIRST", cfg.Mode, "模式应该匹配环境变量")
	assert.Equal(t, 15, cfg.TaskInterval, "间隔应该匹配环境变量")
}

// TestGetArgs_DataFile tests DATA_FILE env and --data-file flag
func TestGetArgs_DataFile(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Default: DataFile should be DEFAULT_DATA_FILE
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, define.DEFAULT_DATA_FILE, cfg.DataFile, "default data file path")

	// Env DATA_FILE
	require.NoError(t, envMgr.Set("DATA_FILE", "/custom/data.json"))
	os.Args = []string{"test"}
	cfg = GetArgs()
	assert.Equal(t, "/custom/data.json", cfg.DataFile, "DATA_FILE env should set DataFile")

	// Flag --data-file overrides
	os.Args = []string{"test", "--data-file", "/flag/data.json"}
	cfg = GetArgs()
	assert.Equal(t, "/flag/data.json", cfg.DataFile, "--data-file should set DataFile")
}

// TestReadPasswordFromFile tests ReadPasswordFromFile function
func TestReadPasswordFromFile(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-password-*.txt")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write password to file
	testPassword := "  test-password-123  \n"
	_, err = tmpFile.WriteString(testPassword)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Test reading password
	password, err := flagutil.ReadPasswordFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "test-password-123", password, "密码应该被正确读取并去除空白字符")
}

// TestReadPasswordFromFile_NonExistent tests reading from non-existent file
func TestReadPasswordFromFile_NonExistent(t *testing.T) {
	_, err := flagutil.ReadPasswordFromFile("/nonexistent/file/path.txt")
	assert.Error(t, err, "读取不存在的文件应该返回错误")
}

// TestReadPasswordFromFile_EmptyFile tests reading from empty file
func TestReadPasswordFromFile_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-empty-*.txt")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()
	require.NoError(t, tmpFile.Close())

	password, err := flagutil.ReadPasswordFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Empty(t, password, "空文件应该返回空字符串")
}

// TestGetArgs_RedisEnabled_OnlyLocal tests Redis enabled logic in ONLY_LOCAL mode
func TestGetArgs_RedisEnabled_OnlyLocal(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Test ONLY_LOCAL mode without explicit Redis address
	require.NoError(t, envMgr.Set("MODE", "ONLY_LOCAL"))
	require.NoError(t, envMgr.Unset("REDIS"))
	require.NoError(t, envMgr.Unset("REDIS_ENABLED"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.False(t, cfg.RedisEnabled, "ONLY_LOCAL模式且未设置Redis地址时应该禁用Redis")

	// Test ONLY_LOCAL mode with explicit Redis address
	require.NoError(t, envMgr.Set("REDIS", "localhost:6379"))
	cfg = GetArgs()
	assert.True(t, cfg.RedisEnabled, "ONLY_LOCAL模式但设置了Redis地址时应该启用Redis")
}

// TestGetArgs_RedisEnabled_Explicit tests explicit Redis enabled flag
func TestGetArgs_RedisEnabled_Explicit(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Test with REDIS_ENABLED=true
	require.NoError(t, envMgr.Set("REDIS_ENABLED", "true"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.True(t, cfg.RedisEnabled, "REDIS_ENABLED=true时应该启用Redis")

	// Test with REDIS_ENABLED=false
	require.NoError(t, envMgr.Set("REDIS_ENABLED", "false"))
	cfg = GetArgs()
	assert.False(t, cfg.RedisEnabled, "REDIS_ENABLED=false时应该禁用Redis")

	// Test with REDIS_ENABLED=1
	require.NoError(t, envMgr.Set("REDIS_ENABLED", "1"))
	cfg = GetArgs()
	assert.True(t, cfg.RedisEnabled, "REDIS_ENABLED=1时应该启用Redis")
}

// TestGetArgs_RedisPassword_FromFile tests reading Redis password from file
func TestGetArgs_RedisPassword_FromFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create password file
	tmpFile, err := os.CreateTemp("", "test-redis-password-*.txt")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("file-password-123")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("清理临时文件失败: %s", tmpFile.Name())
		}
	}()

	// Test reading from file
	require.NoError(t, envMgr.Set("REDIS_PASSWORD_FILE", tmpFile.Name()))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, "file-password-123", cfg.RedisPassword, "应该从文件读取密码")
}

// TestGetArgs_HTTPTimeout tests HTTP timeout configuration
func TestGetArgs_HTTPTimeout(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Test with integer seconds
	require.NoError(t, envMgr.Set("HTTP_TIMEOUT", "30"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, 30, cfg.HTTPTimeout, "应该正确解析整数秒数")

	// Test with duration format
	require.NoError(t, envMgr.Set("HTTP_TIMEOUT", "45s"))
	cfg = GetArgs()
	assert.Equal(t, 45, cfg.HTTPTimeout, "应该正确解析duration格式")
}

// TestGetArgs_HTTPMaxIdleConns tests HTTP max idle connections
func TestGetArgs_HTTPMaxIdleConns(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	require.NoError(t, envMgr.Set("HTTP_MAX_IDLE_CONNS", "200"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, 200, cfg.HTTPMaxIdleConns, "应该正确设置最大空闲连接数")
}

// TestGetArgs_HTTPInsecureTLS tests HTTP insecure TLS configuration
func TestGetArgs_HTTPInsecureTLS(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Test with true
	require.NoError(t, envMgr.Set("HTTP_INSECURE_TLS", "true"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.True(t, cfg.HTTPInsecureTLS, "HTTP_INSECURE_TLS=true时应该启用")

	// Test with 1
	require.NoError(t, envMgr.Set("HTTP_INSECURE_TLS", "1"))
	cfg = GetArgs()
	assert.True(t, cfg.HTTPInsecureTLS, "HTTP_INSECURE_TLS=1时应该启用")
}

// TestGetArgs_APIKey tests API key configuration
func TestGetArgs_APIKey(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	require.NoError(t, envMgr.Set("API_KEY", "test-api-key-123"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, "test-api-key-123", cfg.APIKey, "应该正确设置API密钥")
}

// TestGetArgs_CommandLinePriority tests command-line arguments priority
func TestGetArgs_CommandLinePriority(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Set environment variable
	require.NoError(t, envMgr.Set("PORT", "8888"))
	// Set command-line argument (should override env var)
	os.Args = []string{"test", "--port", "9999"}

	cfg := GetArgs()
	assert.Equal(t, "9999", cfg.Port, "命令行参数应该覆盖环境变量")
}

// TestGetArgs_InvalidPortEnv tests invalid port in environment variable
func TestGetArgs_InvalidPortEnv(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Set invalid port in environment
	require.NoError(t, envMgr.Set("PORT", "invalid"))
	os.Args = []string{"test"}

	cfg := GetArgs()
	// Should fallback to default port
	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port, "无效端口应该使用默认值")
}

// TestGetArgs_InvalidIntervalEnv tests invalid interval in environment variable
func TestGetArgs_InvalidIntervalEnv(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Set invalid interval in environment
	require.NoError(t, envMgr.Set("INTERVAL", "invalid"))
	os.Args = []string{"test"}

	cfg := GetArgs()
	// Should fallback to default interval
	assert.Equal(t, define.DEFAULT_TASK_INTERVAL, cfg.TaskInterval, "无效间隔应该使用默认值")
}

// TestGetArgs_WithConfigFile tests GetArgs with configuration file
func TestGetArgs_WithConfigFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create temporary YAML config file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write valid YAML config
	yamlContent := `server:
  port: "8080"
redis:
  addr: "localhost:6380"
remote:
  url: "http://example.com/config"
  key: "config-key"
  mode: "REMOTE_FIRST"
app:
  mode: "REMOTE_FIRST"
  api_key: "test-api-key"
task:
  interval: 20s
http:
  timeout: 60s
  max_idle_conns: 150
  insecure_tls: true
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Clear environment variables to ensure config file values are used
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE", "API_KEY", "HTTP_TIMEOUT", "HTTP_MAX_IDLE_CONNS", "HTTP_INSECURE_TLS"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test", "--config-file", tmpFile.Name()}
	cfg := GetArgs()

	assert.Equal(t, "8080", cfg.Port, "应该从配置文件读取端口")
	assert.Equal(t, "localhost:6380", cfg.Redis, "应该从配置文件读取Redis地址")
	assert.Equal(t, "http://example.com/config", cfg.RemoteConfig, "应该从配置文件读取远程配置URL")
	assert.Equal(t, "config-key", cfg.RemoteKey, "应该从配置文件读取密钥")
	assert.Equal(t, "REMOTE_FIRST", cfg.Mode, "应该从配置文件读取模式")
	assert.Equal(t, "test-api-key", cfg.APIKey, "应该从配置文件读取API密钥")
	assert.Equal(t, 20, cfg.TaskInterval, "应该从配置文件读取任务间隔")
	assert.Equal(t, 60, cfg.HTTPTimeout, "应该从配置文件读取HTTP超时")
	assert.Equal(t, 150, cfg.HTTPMaxIdleConns, "应该从配置文件读取最大空闲连接数")
	assert.True(t, cfg.HTTPInsecureTLS, "应该从配置文件读取TLS设置")
}

// TestGetArgs_WithConfigFile_OverrideByCLI tests CLI arguments override config file
func TestGetArgs_WithConfigFile_OverrideByCLI(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create temporary YAML config file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write valid YAML config
	yamlContent := `server:
  port: "8080"
redis:
  addr: "localhost:6380"
remote:
  url: "http://example.com/config"
  key: "config-key"
app:
  mode: "REMOTE_FIRST"
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	// CLI arguments should override config file values
	os.Args = []string{"test", "--config-file", tmpFile.Name(), "--port", "9090", "--redis", "127.0.0.1:6379", "--mode", "ONLY_LOCAL"}
	cfg := GetArgs()

	assert.Equal(t, "9090", cfg.Port, "CLI参数应该覆盖配置文件中的端口")
	assert.Equal(t, "127.0.0.1:6379", cfg.Redis, "CLI参数应该覆盖配置文件中的Redis地址")
	assert.Equal(t, "ONLY_LOCAL", cfg.Mode, "CLI参数应该覆盖配置文件中的模式")
	// These should still come from config file
	assert.Equal(t, "http://example.com/config", cfg.RemoteConfig, "未覆盖的配置应该来自配置文件")
	assert.Equal(t, "config-key", cfg.RemoteKey, "未覆盖的配置应该来自配置文件")
}

// TestGetArgs_WithConfigFile_InvalidFile tests GetArgs with invalid config file
func TestGetArgs_WithConfigFile_InvalidFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	// Use non-existent config file
	os.Args = []string{"test", "--config-file", "/nonexistent/config.yaml"}
	cfg := GetArgs()

	// Should fallback to default values
	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port, "无效配置文件应该回退到默认值")
	assert.Equal(t, define.DEFAULT_REDIS, cfg.Redis, "无效配置文件应该回退到默认值")
}

// TestLoadConfig tests LoadConfig function
func TestLoadConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create temporary YAML config file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write valid YAML config
	yamlContent := `server:
  port: "8080"
redis:
  addr: "localhost:6380"
remote:
  url: "http://example.com/config"
  key: "config-key"
  mode: "REMOTE_FIRST"
app:
  mode: "REMOTE_FIRST"
  api_key: "test-api-key"
task:
  interval: 20s
http:
  timeout: 60s
  max_idle_conns: 150
  insecure_tls: true
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE", "API_KEY", "HTTP_TIMEOUT", "HTTP_MAX_IDLE_CONNS", "HTTP_INSECURE_TLS"}
	for _, key := range envVarsToClear {
		if unsetErr := envMgr.Unset(key); unsetErr != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test"}
	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port, "应该从配置文件读取端口")
	assert.Equal(t, "localhost:6380", cfg.Redis, "应该从配置文件读取Redis地址")
	assert.Equal(t, "http://example.com/config", cfg.RemoteConfig, "应该从配置文件读取远程配置URL")
	assert.Equal(t, "config-key", cfg.RemoteKey, "应该从配置文件读取密钥")
	assert.Equal(t, "REMOTE_FIRST", cfg.Mode, "应该从配置文件读取模式")
	assert.Equal(t, "test-api-key", cfg.APIKey, "应该从配置文件读取API密钥")
	assert.Equal(t, 20, cfg.TaskInterval, "应该从配置文件读取任务间隔")
	assert.Equal(t, 60, cfg.HTTPTimeout, "应该从配置文件读取HTTP超时")
	assert.Equal(t, 150, cfg.HTTPMaxIdleConns, "应该从配置文件读取最大空闲连接数")
	assert.True(t, cfg.HTTPInsecureTLS, "应该从配置文件读取TLS设置")
}

// TestLoadConfig_NoFile tests LoadConfig without config file
func TestLoadConfig_NoFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test"}
	cfg, err := LoadConfig("")
	require.NoError(t, err)

	// Should use default values
	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port, "无配置文件时应该使用默认值")
	assert.Equal(t, define.DEFAULT_REDIS, cfg.Redis, "无配置文件时应该使用默认值")
}

// TestLoadConfig_InvalidFile tests LoadConfig with invalid YAML content
func TestLoadConfig_InvalidFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Create temporary file with invalid YAML
	tmpFile, err := os.CreateTemp("", "test-invalid-config-*.yaml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write invalid YAML content
	_, err = tmpFile.WriteString("invalid: yaml: content: [unclosed")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	os.Args = []string{"test"}
	cfg, err := LoadConfig(tmpFile.Name())
	assert.Error(t, err, "无效YAML内容应该返回错误")
	assert.Nil(t, cfg, "无效YAML内容应该返回nil")
}

// TestLoadConfig_NonExistentFile tests LoadConfig with non-existent file (should use defaults)
func TestLoadConfig_NonExistentFile(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	envVarsToClear := []string{"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL", "MODE"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test"}
	// Non-existent file should not return error, but use defaults
	cfg, err := LoadConfig("/nonexistent/config.yaml")
	require.NoError(t, err, "不存在的配置文件不应该返回错误，应该使用默认值")
	assert.NotNil(t, cfg, "应该返回默认配置")
	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port, "应该使用默认端口")
}

// TestLoadConfig_WithEnvOverride tests LoadConfig with environment variable override
func TestLoadConfig_WithEnvOverride(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create temporary YAML config file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write valid YAML config
	yamlContent := `server:
  port: "8080"
redis:
  addr: "localhost:6380"
app:
  mode: "REMOTE_FIRST"
`
	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Set environment variables (should override config file)
	require.NoError(t, envMgr.Set("PORT", "9999"))
	require.NoError(t, envMgr.Set("REDIS", "192.168.1.1:6379"))
	require.NoError(t, envMgr.Set("MODE", "ONLY_LOCAL"))

	os.Args = []string{"test"}
	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Environment variables should override config file
	assert.Equal(t, "9999", cfg.Port, "环境变量应该覆盖配置文件")
	assert.Equal(t, "192.168.1.1:6379", cfg.Redis, "环境变量应该覆盖配置文件")
	assert.Equal(t, "ONLY_LOCAL", cfg.Mode, "环境变量应该覆盖配置文件")
}

// TestGetArgs_RedisPassword_FromCLI tests Redis password from CLI argument
func TestGetArgs_RedisPassword_FromCLI(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	require.NoError(t, envMgr.Unset("REDIS_PASSWORD"))
	require.NoError(t, envMgr.Unset("REDIS_PASSWORD_FILE"))

	os.Args = []string{"test", "--redis-password", "cli-password-123"}
	cfg := GetArgs()

	assert.Equal(t, "cli-password-123", cfg.RedisPassword, "应该从CLI参数读取Redis密码")
}

// TestGetArgs_HTTPTimeout_FromCLI tests HTTP timeout from CLI argument
func TestGetArgs_HTTPTimeout_FromCLI(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	require.NoError(t, envMgr.Unset("HTTP_TIMEOUT"))

	os.Args = []string{"test", "--http-timeout", "120"}
	cfg := GetArgs()

	assert.Equal(t, 120, cfg.HTTPTimeout, "应该从CLI参数读取HTTP超时")
}

// TestGetArgs_HTTPInsecureTLS_FromCLI tests HTTP insecure TLS from CLI argument
func TestGetArgs_HTTPInsecureTLS_FromCLI(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	require.NoError(t, envMgr.Unset("HTTP_INSECURE_TLS"))

	os.Args = []string{"test", "--http-insecure-tls"}
	cfg := GetArgs()

	assert.True(t, cfg.HTTPInsecureTLS, "应该从CLI参数读取HTTP insecure TLS设置")
}

// TestGetArgs_RedisEnabled_FromCLI tests Redis enabled from CLI argument
func TestGetArgs_RedisEnabled_FromCLI(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear environment variables
	require.NoError(t, envMgr.Unset("REDIS_ENABLED"))

	// Test with --redis-enabled=false
	os.Args = []string{"test", "--redis-enabled=false"}
	cfg := GetArgs()
	assert.False(t, cfg.RedisEnabled, "CLI参数应该能够禁用Redis")

	// Test with --redis-enabled=true
	os.Args = []string{"test", "--redis-enabled=true"}
	cfg = GetArgs()
	assert.True(t, cfg.RedisEnabled, "CLI参数应该能够启用Redis")
}

// TestGetArgs_AllCLIFlags tests all CLI flags
func TestGetArgs_AllCLIFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Clear all environment variables
	envVarsToClear := []string{"PORT", "REDIS", "REDIS_PASSWORD", "REDIS_ENABLED", "CONFIG", "KEY", "MODE", "INTERVAL", "HTTP_TIMEOUT", "HTTP_MAX_IDLE_CONNS", "HTTP_INSECURE_TLS", "API_KEY"}
	for _, key := range envVarsToClear {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("清理环境变量失败: %s", key)
		}
	}

	os.Args = []string{"test",
		"--port", "7777",
		"--redis", "10.0.0.1:6379",
		"--redis-password", "test-pwd",
		"--redis-enabled=true",
		"--config", "http://test.com/config",
		"--key", "test-key",
		"--mode", "LOCAL_ONLY",
		"--interval", "30",
		"--http-timeout", "90",
		"--http-max-idle-conns", "200",
		"--http-insecure-tls=true",
		"--api-key", "cli-api-key",
	}

	cfg := GetArgs()

	assert.Equal(t, "7777", cfg.Port, "CLI参数应该设置端口")
	assert.Equal(t, "10.0.0.1:6379", cfg.Redis, "CLI参数应该设置Redis地址")
	assert.Equal(t, "test-pwd", cfg.RedisPassword, "CLI参数应该设置Redis密码")
	assert.True(t, cfg.RedisEnabled, "CLI参数应该启用Redis")
	assert.Equal(t, "http://test.com/config", cfg.RemoteConfig, "CLI参数应该设置远程配置URL")
	assert.Equal(t, "test-key", cfg.RemoteKey, "CLI参数应该设置密钥")
	assert.Equal(t, "LOCAL_ONLY", cfg.Mode, "CLI参数应该设置模式")
	assert.Equal(t, 30, cfg.TaskInterval, "CLI参数应该设置任务间隔")
	assert.Equal(t, 90, cfg.HTTPTimeout, "CLI参数应该设置HTTP超时")
	assert.Equal(t, 200, cfg.HTTPMaxIdleConns, "CLI参数应该设置最大空闲连接数")
	assert.True(t, cfg.HTTPInsecureTLS, "CLI参数应该启用不安全的TLS")
	assert.Equal(t, "cli-api-key", cfg.APIKey, "CLI参数应该设置API密钥")
}

// TestGetArgs_RedisPassword_Priority tests Redis password priority
func TestGetArgs_RedisPassword_Priority(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Use EnvManager to manage environment variables
	envMgr := testutil.NewEnvManager()
	defer envMgr.Cleanup()

	// Create password file
	tmpFile, err := os.CreateTemp("", "test-redis-password-*.txt")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("file-password")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("清理临时文件失败: %s", tmpFile.Name())
		}
	}()

	// Test priority: ENV > FILE > CLI
	// Set all three sources
	require.NoError(t, envMgr.Set("REDIS_PASSWORD", "env-password"))
	require.NoError(t, envMgr.Set("REDIS_PASSWORD_FILE", tmpFile.Name()))
	os.Args = []string{"test", "--redis-password", "cli-password"}

	cfg := GetArgs()
	assert.Equal(t, "env-password", cfg.RedisPassword, "环境变量应该有最高优先级")

	// Test priority: FILE > CLI (when ENV is not set)
	require.NoError(t, envMgr.Unset("REDIS_PASSWORD"))
	cfg = GetArgs()
	assert.Equal(t, "file-password", cfg.RedisPassword, "文件应该有第二优先级")

	// Test priority: CLI (when ENV and FILE are not set)
	require.NoError(t, envMgr.Unset("REDIS_PASSWORD_FILE"))
	cfg = GetArgs()
	assert.Equal(t, "cli-password", cfg.RedisPassword, "CLI参数应该有最低优先级")
}
