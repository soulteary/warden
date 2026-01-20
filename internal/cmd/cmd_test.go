package cmd

import (
	"os"
	"strconv"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
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
