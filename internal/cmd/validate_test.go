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

func TestValidateConfig_RemoteDecryptEnabled_KeyFileNotSet(t *testing.T) {
	cfg := &Config{
		Port:                    "8081",
		Redis:                   "localhost:6379",
		TaskInterval:            5,
		Mode:                    "DEFAULT",
		RemoteDecryptEnabled:    true,
		RemoteRSAPrivateKeyFile: "", // 未设置
		RemoteRSAPrivateKey:     "", // 未设置
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "启用远程解密但未配置私钥文件或 PEM 应返回错误")
	assert.Contains(t, err.Error(), "REMOTE_RSA_PRIVATE_KEY")
}

func TestValidateConfig_RemoteDecryptEnabled_OnlyPEMSet(t *testing.T) {
	cfg := &Config{ // #nosec G101 -- test fixture only, not a real key
		Port:                    "8081",
		Redis:                   "localhost:6379",
		TaskInterval:            5,
		Mode:                    "DEFAULT",
		RemoteDecryptEnabled:    true,
		RemoteRSAPrivateKeyFile: "",
		RemoteRSAPrivateKey:     "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0\n-----END RSA PRIVATE KEY-----",
	}

	err := ValidateConfig(cfg)
	assert.NoError(t, err, "仅设置 REMOTE_RSA_PRIVATE_KEY 时校验应通过，PEM 在加载时解析")
}

func TestValidateConfig_RemoteDecryptEnabled_KeyFileNotExist(t *testing.T) {
	cfg := &Config{
		Port:                    "8081",
		Redis:                   "localhost:6379",
		TaskInterval:            5,
		Mode:                    "DEFAULT",
		RemoteDecryptEnabled:    true,
		RemoteRSAPrivateKeyFile: "/nonexistent/path/to/private.pem",
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "启用远程解密但私钥文件不存在应返回错误")
	assert.Contains(t, err.Error(), "does not exist")
}

// TestValidateConfig_InvalidEnvironment ensures an unrecognized ENVIRONMENT is rejected.
func TestValidateConfig_InvalidEnvironment(t *testing.T) {
	cfg := &Config{
		Port:         "8081",
		Redis:        "localhost:6379",
		RemoteConfig: "http://example.com/data.json",
		TaskInterval: 5,
		Mode:         "DEFAULT",
		Environment:  "staging", // not a recognized environment
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "无效部署环境应返回错误")
}

// TestValidateConfig_ProductionForbidsInsecureTLS ensures production hardening keys off
// ENVIRONMENT (not the merge mode) and rejects disabled TLS verification.
func TestValidateConfig_ProductionForbidsInsecureTLS(t *testing.T) {
	cfg := &Config{
		Port:            "8081",
		Redis:           "localhost:6379",
		TaskInterval:    5,
		Mode:            "DEFAULT",
		Environment:     "production",
		HTTPInsecureTLS: true,
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "生产环境禁止禁用 TLS 验证")
}

// TestValidateConfig_ProductionRemoteRequiresEncryptionAndTimeout ensures production
// remote fetches must be bounded and encrypted (fail closed).
func TestValidateConfig_ProductionRemoteRequiresEncryptionAndTimeout(t *testing.T) {
	cfg := &Config{
		Port:                     "8081",
		Redis:                    "localhost:6379",
		TaskInterval:             5,
		Mode:                     "DEFAULT",
		Environment:              "production",
		RemoteConfig:             "http://example.com/data.json",
		HTTPTimeout:              0,     // unbounded
		RemoteEncryptionRequired: false, // not fail-closed
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "生产环境远程数据源必须设置超时并要求加密")
}

// TestValidateConfig_ProductionRemoteOK ensures a properly hardened production remote
// config passes validation.
func TestValidateConfig_ProductionRemoteOK(t *testing.T) {
	cfg := &Config{
		Port:                     "8081",
		Redis:                    "localhost:6379",
		TaskInterval:             5,
		Mode:                     "DEFAULT",
		Environment:              "production",
		RemoteConfig:             "http://example.com/data.json",
		HTTPTimeout:              30,
		RemoteEncryptionRequired: true,
	}

	err := ValidateConfig(cfg)
	assert.NoError(t, err, "满足生产硬化要求的配置应通过校验")
}

// TestValidateConfig_DevelopmentAllowsInsecureTLS ensures the development environment
// does NOT trigger production hardening (explicitly allowed for local workflows).
func TestValidateConfig_DevelopmentAllowsInsecureTLS(t *testing.T) {
	cfg := &Config{
		Port:            "8081",
		Redis:           "localhost:6379",
		TaskInterval:    5,
		Mode:            "DEFAULT",
		Environment:     "development",
		HTTPInsecureTLS: true,
	}

	err := ValidateConfig(cfg)
	assert.NoError(t, err, "开发环境显式允许禁用 TLS 验证")
}

// TestValidateConfig_ProdAliasNormalized ensures the "prod" alias is treated as production.
func TestValidateConfig_ProdAliasNormalized(t *testing.T) {
	cfg := &Config{
		Port:            "8081",
		Redis:           "localhost:6379",
		TaskInterval:    5,
		Mode:            "DEFAULT",
		Environment:     "prod",
		HTTPInsecureTLS: true,
	}

	err := ValidateConfig(cfg)
	assert.Error(t, err, "prod 别名应规范化为 production 并触发硬化校验")
}
