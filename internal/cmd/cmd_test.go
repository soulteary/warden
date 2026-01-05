package cmd

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestGetArgs_DefaultValues(t *testing.T) {
	// 保存原始状态
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
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	// 清理环境变量
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("REDIS")
	_ = os.Unsetenv("CONFIG")
	_ = os.Unsetenv("KEY")
	_ = os.Unsetenv("INTERVAL")
	_ = os.Unsetenv("MODE")

	os.Args = []string{"test"}

	cfg := GetArgs()

	assert.Equal(t, strconv.Itoa(define.DefaultPort), cfg.Port)
	assert.Equal(t, define.DefaultRedis, cfg.Redis)
	assert.Equal(t, define.DefaultRemoteConfig, cfg.RemoteConfig)
	assert.Equal(t, define.DefaultRemoteKey, cfg.RemoteKey)
	assert.Equal(t, define.DefaultTaskInterval, cfg.TaskInterval)
	assert.Equal(t, define.DefaultMode, cfg.Mode)
}

func TestGetArgs_WithCommandLineArgs(t *testing.T) {
	// 保存原始状态
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
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	// 清理环境变量
	for k := range oldEnv {
		_ = os.Unsetenv(k)
	}

	// 设置命令行参数
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
	// 保存原始状态
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
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	// 清理环境变量
	for k := range oldEnv {
		_ = os.Unsetenv(k)
	}

	// 设置环境变量
	_ = os.Setenv("PORT", "8888")
	_ = os.Setenv("REDIS", "192.168.1.1:6379")
	_ = os.Setenv("CONFIG", "http://test.com/config.json")
	_ = os.Setenv("KEY", "env-key")
	_ = os.Setenv("INTERVAL", "15")
	_ = os.Setenv("MODE", "REMOTE_FIRST")

	os.Args = []string{"test"}

	cfg := GetArgs()

	assert.Equal(t, "8888", cfg.Port, "端口应该匹配环境变量")
	assert.Equal(t, "192.168.1.1:6379", cfg.Redis, "Redis地址应该匹配环境变量")
	assert.Equal(t, "http://test.com/config.json", cfg.RemoteConfig, "配置URL应该匹配环境变量")
	assert.Equal(t, "env-key", cfg.RemoteKey, "密钥应该匹配环境变量")
	assert.Equal(t, "REMOTE_FIRST", cfg.Mode, "模式应该匹配环境变量")
	assert.Equal(t, 15, cfg.TaskInterval, "间隔应该匹配环境变量")
}
