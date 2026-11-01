package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentSecurityAndCredentialPrecedence(t *testing.T) {
	cfg := &Config{Redis: RedisConfig{Password: "yaml", PasswordFile: "yaml-file"}}
	// Test-only values exercise credential precedence.
	// #nosec G101 -- fixture credentials are never used outside this test.
	for k, v := range map[string]string{"REDIS_PASSWORD": " env-secret ", "REDIS_PASSWORD_FILE": "env-file", "DATA_DIR": "/data/rules", "REMOTE_DECRYPT_ENABLED": "1", "REMOTE_RSA_PRIVATE_KEY_FILE": "/keys/key.pem"} {
		t.Setenv(k, v)
	}
	overrideFromEnv(cfg)
	require.Equal(t, "env-secret", cfg.Redis.Password)
	require.Empty(t, cfg.Redis.PasswordFile)
	require.Equal(t, "/data/rules", cfg.App.DataDir)
	require.True(t, cfg.Remote.DecryptEnabled)
	require.Equal(t, "/keys/key.pem", cfg.Remote.RSAPrivateKeyFile)
	t.Setenv("REDIS_PASSWORD", "")
	overrideFromEnv(cfg)
	require.Equal(t, "env-file", cfg.Redis.PasswordFile)
}

func TestCmdConfigSecurityOverrides(t *testing.T) {
	for k, v := range map[string]string{"ENVIRONMENT": "test", "MERGE_MODE": "ONLY_LOCAL", "REDIS_ENABLED": "false", "RESPONSE_FIELDS": "mail,user_id", "WARDEN_HMAC_KEYS": "service:secret", "WARDEN_HMAC_TIMESTAMP_TOLERANCE": "25", "WARDEN_TLS_REQUIRE_CLIENT_CERT": "1", "REMOTE_DECRYPT_ENABLED": "true", "REMOTE_RSA_PRIVATE_KEY_FILE": "/env.pem", "REMOTE_ENCRYPTION_REQUIRED": "1", "REMOTE_ENCRYPTION_FORMAT": "V2", "REQUIRE_EXPLICIT_USER_ID": "true"} {
		t.Setenv(k, v)
	}
	cfg := (&Config{}).ToCmdConfig()
	require.Equal(t, "test", cfg.Environment)
	require.Equal(t, "ONLY_LOCAL", cfg.Mode)
	require.False(t, cfg.RedisEnabled)
	require.Equal(t, []string{"mail", "user_id"}, cfg.ResponseFields)
	require.Equal(t, 25, cfg.HMACToleranceSec)
	require.True(t, cfg.TLSRequireClientCert)
	require.True(t, cfg.RemoteDecryptEnabled)
	require.Equal(t, "/env.pem", cfg.RemoteRSAPrivateKeyFile)
	require.True(t, cfg.RemoteEncryptionRequired)
	require.Equal(t, "v2", cfg.RemoteEncryptionFormat)
	require.True(t, cfg.RequireExplicitUserID)
	t.Setenv("WARDEN_HMAC_TIMESTAMP_TOLERANCE", "invalid")
	require.Equal(t, 60, (&Config{}).ToCmdConfig().HMACToleranceSec)
}

func TestLegacyConfigFallbackAndUnreadablePassword(t *testing.T) {
	for _, key := range []string{"REDIS_PASSWORD", "REDIS_PASSWORD_FILE", "ENVIRONMENT", "MERGE_MODE"} {
		t.Setenv(key, "")
	}
	cfg := &Config{Redis: RedisConfig{PasswordFile: filepath.Join(t.TempDir(), "missing")}}
	require.Empty(t, cfg.ToLegacyConfig().RedisPassword)
	require.NotEmpty(t, cfg.ToLegacyConfig().Mode)
	require.Empty(t, cfg.ToCmdConfig().RedisPassword)
	require.NotEmpty(t, cfg.ToCmdConfig().Mode)
	cfg.App.Mode = "prod"
	result := cfg.ToCmdConfig()
	require.Equal(t, "production", result.Environment)
	require.True(t, result.LegacyModeUsedForEnv)
	require.Equal(t, "DEFAULT", result.Mode)
	cfg.Redis.PasswordFile = "bad..path"
	_, err := cfg.GetRedisPassword()
	require.Error(t, err)
}

func TestConfigFileParseAndValidationFailures(t *testing.T) {
	dir := t.TempDir()
	_, err := ParseFromFile(dir)
	require.Error(t, err)
	_, err = ParseFromFile(filepath.Join(dir, "bad..yaml"))
	require.Error(t, err)
	for _, content := range []string{"server:\n  port: '9000'\n", "[invalid"} {
		path := filepath.Join(dir, "config.unknown")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		cfg, parseErr := ParseFromFile(path)
		if content[0] == '[' {
			require.Error(t, parseErr)
		} else {
			require.NoError(t, parseErr)
			require.Equal(t, "9000", cfg.Server.Port)
		}
	}
	t.Setenv("TASK_INTERVAL", "")
	path := filepath.Join(dir, "invalid.yaml")
	require.NoError(t, os.WriteFile(path, []byte("task:\n  interval: 1ns\n"), 0o600))
	_, err = LoadFromFile(path)
	require.Error(t, err)
	require.Error(t, validate(&Config{Task: TaskConfig{Interval: time.Nanosecond}}))
}
