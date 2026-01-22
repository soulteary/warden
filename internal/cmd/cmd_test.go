package cmd

import (
	"os"
	"strconv"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetArgs_DefaultValues(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	oldEnv := map[string]string{
		"PORT":     os.Getenv("PORT"),
		"REDIS":    os.Getenv("REDIS"),
		"CONFIG":   os.Getenv("CONFIG"),
		"KEY":      os.Getenv("KEY"),
		"INTERVAL": os.Getenv("INTERVAL"),
		"MODE":     os.Getenv("MODE"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Clear environment variables
	if err := os.Unsetenv("PORT"); err != nil {
		t.Logf("清理环境变量失败: PORT")
	}
	if err := os.Unsetenv("REDIS"); err != nil {
		t.Logf("清理环境变量失败: REDIS")
	}
	if err := os.Unsetenv("CONFIG"); err != nil {
		t.Logf("清理环境变量失败: CONFIG")
	}
	if err := os.Unsetenv("KEY"); err != nil {
		t.Logf("清理环境变量失败: KEY")
	}
	if err := os.Unsetenv("INTERVAL"); err != nil {
		t.Logf("清理环境变量失败: INTERVAL")
	}
	if err := os.Unsetenv("MODE"); err != nil {
		t.Logf("清理环境变量失败: MODE")
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
	oldEnv := map[string]string{
		"PORT":     os.Getenv("PORT"),
		"REDIS":    os.Getenv("REDIS"),
		"CONFIG":   os.Getenv("CONFIG"),
		"KEY":      os.Getenv("KEY"),
		"INTERVAL": os.Getenv("INTERVAL"),
		"MODE":     os.Getenv("MODE"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Clear environment variables
	for k := range oldEnv {
		if err := os.Unsetenv(k); err != nil {
			t.Logf("清理环境变量失败: %s", k)
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
	oldEnv := map[string]string{
		"PORT":     os.Getenv("PORT"),
		"REDIS":    os.Getenv("REDIS"),
		"CONFIG":   os.Getenv("CONFIG"),
		"KEY":      os.Getenv("KEY"),
		"INTERVAL": os.Getenv("INTERVAL"),
		"MODE":     os.Getenv("MODE"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Clear environment variables
	for k := range oldEnv {
		if err := os.Unsetenv(k); err != nil {
			t.Logf("清理环境变量失败: %s", k)
		}
	}

	// Set environment variables
	if err := os.Setenv("PORT", "8888"); err != nil {
		t.Fatalf("设置环境变量失败: PORT")
	}
	if err := os.Setenv("REDIS", "192.168.1.1:6379"); err != nil {
		t.Fatalf("设置环境变量失败: REDIS")
	}
	if err := os.Setenv("CONFIG", "http://test.com/data.json"); err != nil {
		t.Fatalf("设置环境变量失败: CONFIG")
	}
	if err := os.Setenv("KEY", "env-key"); err != nil {
		t.Fatalf("设置环境变量失败: KEY")
	}
	if err := os.Setenv("INTERVAL", "15"); err != nil {
		t.Fatalf("设置环境变量失败: INTERVAL")
	}
	if err := os.Setenv("MODE", "REMOTE_FIRST"); err != nil {
		t.Fatalf("设置环境变量失败: MODE")
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

// TestReadPasswordFromFile tests readPasswordFromFile function
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
	password, err := readPasswordFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "test-password-123", password, "密码应该被正确读取并去除空白字符")
}

// TestReadPasswordFromFile_NonExistent tests reading from non-existent file
func TestReadPasswordFromFile_NonExistent(t *testing.T) {
	_, err := readPasswordFromFile("/nonexistent/file/path.txt")
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

	password, err := readPasswordFromFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Empty(t, password, "空文件应该返回空字符串")
}

// TestGetArgs_RedisEnabled_OnlyLocal tests Redis enabled logic in ONLY_LOCAL mode
func TestGetArgs_RedisEnabled_OnlyLocal(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"MODE":          os.Getenv("MODE"),
		"REDIS":         os.Getenv("REDIS"),
		"REDIS_ENABLED": os.Getenv("REDIS_ENABLED"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Clear environment
	for k := range oldEnv {
		if err := os.Unsetenv(k); err != nil {
			t.Logf("清理环境变量失败: %s", k)
		}
	}

	// Test ONLY_LOCAL mode without explicit Redis address
	require.NoError(t, os.Setenv("MODE", "ONLY_LOCAL"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.False(t, cfg.RedisEnabled, "ONLY_LOCAL模式且未设置Redis地址时应该禁用Redis")

	// Test ONLY_LOCAL mode with explicit Redis address
	require.NoError(t, os.Setenv("REDIS", "localhost:6379"))
	cfg = GetArgs()
	assert.True(t, cfg.RedisEnabled, "ONLY_LOCAL模式但设置了Redis地址时应该启用Redis")
}

// TestGetArgs_RedisEnabled_Explicit tests explicit Redis enabled flag
func TestGetArgs_RedisEnabled_Explicit(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"REDIS_ENABLED": os.Getenv("REDIS_ENABLED"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Test with REDIS_ENABLED=true
	require.NoError(t, os.Setenv("REDIS_ENABLED", "true"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.True(t, cfg.RedisEnabled, "REDIS_ENABLED=true时应该启用Redis")

	// Test with REDIS_ENABLED=false
	require.NoError(t, os.Setenv("REDIS_ENABLED", "false"))
	cfg = GetArgs()
	assert.False(t, cfg.RedisEnabled, "REDIS_ENABLED=false时应该禁用Redis")

	// Test with REDIS_ENABLED=1
	require.NoError(t, os.Setenv("REDIS_ENABLED", "1"))
	cfg = GetArgs()
	assert.True(t, cfg.RedisEnabled, "REDIS_ENABLED=1时应该启用Redis")
}

// TestGetArgs_RedisPassword_FromFile tests reading Redis password from file
func TestGetArgs_RedisPassword_FromFile(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"REDIS_PASSWORD":      os.Getenv("REDIS_PASSWORD"),
		"REDIS_PASSWORD_FILE": os.Getenv("REDIS_PASSWORD_FILE"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Clear environment
	for k := range oldEnv {
		if err := os.Unsetenv(k); err != nil {
			t.Logf("清理环境变量失败: %s", k)
		}
	}

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
	require.NoError(t, os.Setenv("REDIS_PASSWORD_FILE", tmpFile.Name()))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, "file-password-123", cfg.RedisPassword, "应该从文件读取密码")
}

// TestGetArgs_HTTPTimeout tests HTTP timeout configuration
func TestGetArgs_HTTPTimeout(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"HTTP_TIMEOUT": os.Getenv("HTTP_TIMEOUT"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Test with integer seconds
	require.NoError(t, os.Setenv("HTTP_TIMEOUT", "30"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, 30, cfg.HTTPTimeout, "应该正确解析整数秒数")

	// Test with duration format
	require.NoError(t, os.Setenv("HTTP_TIMEOUT", "45s"))
	cfg = GetArgs()
	assert.Equal(t, 45, cfg.HTTPTimeout, "应该正确解析duration格式")
}

// TestGetArgs_HTTPMaxIdleConns tests HTTP max idle connections
func TestGetArgs_HTTPMaxIdleConns(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"HTTP_MAX_IDLE_CONNS": os.Getenv("HTTP_MAX_IDLE_CONNS"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	require.NoError(t, os.Setenv("HTTP_MAX_IDLE_CONNS", "200"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, 200, cfg.HTTPMaxIdleConns, "应该正确设置最大空闲连接数")
}

// TestGetArgs_HTTPInsecureTLS tests HTTP insecure TLS configuration
func TestGetArgs_HTTPInsecureTLS(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"HTTP_INSECURE_TLS": os.Getenv("HTTP_INSECURE_TLS"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Test with true
	require.NoError(t, os.Setenv("HTTP_INSECURE_TLS", "true"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.True(t, cfg.HTTPInsecureTLS, "HTTP_INSECURE_TLS=true时应该启用")

	// Test with 1
	require.NoError(t, os.Setenv("HTTP_INSECURE_TLS", "1"))
	cfg = GetArgs()
	assert.True(t, cfg.HTTPInsecureTLS, "HTTP_INSECURE_TLS=1时应该启用")
}

// TestGetArgs_APIKey tests API key configuration
func TestGetArgs_APIKey(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"API_KEY": os.Getenv("API_KEY"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	require.NoError(t, os.Setenv("API_KEY", "test-api-key-123"))
	os.Args = []string{"test"}
	cfg := GetArgs()
	assert.Equal(t, "test-api-key-123", cfg.APIKey, "应该正确设置API密钥")
}

// TestGetArgs_CommandLinePriority tests command-line arguments priority
func TestGetArgs_CommandLinePriority(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"PORT": os.Getenv("PORT"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Set environment variable
	require.NoError(t, os.Setenv("PORT", "8888"))
	// Set command-line argument (should override env var)
	os.Args = []string{"test", "--port", "9999"}

	cfg := GetArgs()
	assert.Equal(t, "9999", cfg.Port, "命令行参数应该覆盖环境变量")
}

// TestGetArgs_InvalidPortEnv tests invalid port in environment variable
func TestGetArgs_InvalidPortEnv(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"PORT": os.Getenv("PORT"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Set invalid port in environment
	require.NoError(t, os.Setenv("PORT", "invalid"))
	os.Args = []string{"test"}

	cfg := GetArgs()
	// Should fallback to default port
	assert.Equal(t, strconv.Itoa(define.DEFAULT_PORT), cfg.Port, "无效端口应该使用默认值")
}

// TestGetArgs_InvalidIntervalEnv tests invalid interval in environment variable
func TestGetArgs_InvalidIntervalEnv(t *testing.T) {
	oldArgs := os.Args
	oldEnv := map[string]string{
		"INTERVAL": os.Getenv("INTERVAL"),
	}

	defer func() {
		os.Args = oldArgs
		for k, v := range oldEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("恢复环境变量失败: %s", k)
				}
			}
		}
	}()

	// Set invalid interval in environment
	require.NoError(t, os.Setenv("INTERVAL", "invalid"))
	os.Args = []string{"test"}

	cfg := GetArgs()
	// Should fallback to default interval
	assert.Equal(t, define.DEFAULT_TASK_INTERVAL, cfg.TaskInterval, "无效间隔应该使用默认值")
}
