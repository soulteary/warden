package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "localhost:6379",
		RemoteConfig: "http://example.com/data.json",
		RemoteKey:    "test-key",
		TaskInterval: 5,
		Mode:         "DEFAULT",
	}

	err := ValidateConfig(cfg)
	assert.NoError(t, err, "有效配置应该通过验证")
}

func TestValidateConfig_InvalidPort(t *testing.T) {
	cfg := &Config{
		Port:         "99999", // 无效端口
		Redis:        "localhost:6379",
		RemoteConfig: "http://example.com/data.json",
		RemoteKey:    "test-key",
		TaskInterval: 5,
		Mode:         "DEFAULT",
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效端口应该返回错误")
	assert.Contains(t, err.Error(), "端口")
}

func TestValidateConfig_InvalidRedis(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "invalid", // 无效格式
		RemoteConfig: "http://example.com/data.json",
		RemoteKey:    "test-key",
		TaskInterval: 5,
		Mode:         "DEFAULT",
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效Redis地址应该返回错误")
	assert.Contains(t, err.Error(), "Redis")
}

func TestValidateConfig_InvalidURL(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "localhost:6379",
		RemoteConfig: "not-a-valid-url", // 无效URL
		RemoteKey:    "test-key",
		TaskInterval: 5,
		Mode:         "DEFAULT",
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效URL应该返回错误")
	assert.Contains(t, err.Error(), "URL")
}

func TestValidateConfig_InvalidMode(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "localhost:6379",
		RemoteConfig: "http://example.com/data.json",
		RemoteKey:    "test-key",
		TaskInterval: 5,
		Mode:         "INVALID_MODE", // 无效模式
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效模式应该返回错误")
	assert.Contains(t, err.Error(), "模式")
}

func TestValidateConfig_InvalidTaskInterval(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "localhost:6379",
		RemoteConfig: "http://example.com/data.json",
		RemoteKey:    "test-key",
		TaskInterval: 0, // 无效间隔
		Mode:         "DEFAULT",
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效任务间隔应该返回错误")
	assert.Contains(t, err.Error(), "任务间隔")
}
