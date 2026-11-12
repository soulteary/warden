package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/config"
)

func clearRedisCredentialEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"REDIS_PASSWORD", "REDIS_PASSWORD_FILE", "REDIS_USERNAME",
		"REDIS_TLS_ENABLED", "REDIS_TLS_CA_FILE", "REDIS_TLS_SERVER_NAME",
	} {
		t.Setenv(key, "")
	}
}

func TestValidateConfigRejectsUnusableRedisPasswordFile(t *testing.T) {
	clearRedisCredentialEnv(t)
	withCleanFlags(t)
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	empty := filepath.Join(dir, "empty")
	valid := filepath.Join(dir, "valid")
	require.NoError(t, os.WriteFile(empty, []byte(" \n"), 0o600))
	require.NoError(t, os.WriteFile(valid, []byte("file-password\n"), 0o600))

	for _, tc := range []struct {
		name, file, want string
	}{
		{"missing", missing, "is not readable"},
		{"empty", empty, "is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("REDIS_PASSWORD_FILE", tc.file)
			cfg := getArgsFromFlags()
			require.Empty(t, cfg.RedisPassword)
			err := ValidateConfig(cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), "REDIS_PASSWORD_FILE")
			require.Contains(t, err.Error(), tc.want)

			// The file is not consulted when Redis is disabled or REDIS_PASSWORD is set.
			cfg.RedisEnabled = false
			require.Empty(t, validateRedisPasswordFile(cfg))
			t.Setenv("REDIS_PASSWORD", "env-password")
			require.Empty(t, validateRedisPasswordFile(getArgsFromFlags()))
		})
	}

	t.Setenv("REDIS_PASSWORD_FILE", valid)
	cfg := getArgsFromFlags()
	require.Equal(t, "file-password", cfg.RedisPassword)
	require.Empty(t, validateRedisPasswordFile(cfg))
}

func TestConfigFileRejectsUnusableRedisPasswordFile(t *testing.T) {
	clearRedisCredentialEnv(t)
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	fileCfg := &config.Config{}
	fileCfg.Redis.PasswordFile = missing

	err := validateConfigFileRedisPassword(fileCfg, &Config{RedisEnabled: true})
	require.ErrorContains(t, err, "redis.password_file")
	require.ErrorContains(t, err, "is not readable")

	require.NoError(t, os.WriteFile(missing, nil, 0o600))
	require.ErrorContains(t, validateConfigFileRedisPassword(fileCfg, &Config{RedisEnabled: true}), "is empty")

	require.NoError(t, validateConfigFileRedisPassword(fileCfg, &Config{RedisEnabled: false}))
	require.NoError(t, validateConfigFileRedisPassword(fileCfg, &Config{RedisEnabled: true, RedisPassword: "resolved"}))
	require.NoError(t, validateConfigFileRedisPassword(&config.Config{}, &Config{RedisEnabled: true}))
}

func TestRedisTransportEnvironment(t *testing.T) {
	clearRedisCredentialEnv(t)
	withCleanFlags(t)
	for _, cfg := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Empty(t, cfg.RedisUsername)
		require.False(t, cfg.RedisTLSEnabled, "TLS must stay off unless requested")
		require.Empty(t, validateRedisTransport(cfg))
		tlsCfg, err := RedisTLSConfig(cfg)
		require.NoError(t, err)
		require.Nil(t, tlsCfg)
	}

	t.Setenv("REDIS_USERNAME", " warden ")
	t.Setenv("REDIS_TLS_ENABLED", "true")
	t.Setenv("REDIS_TLS_SERVER_NAME", " redis.internal ")
	for _, cfg := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Equal(t, "warden", cfg.RedisUsername)
		require.True(t, cfg.RedisTLSEnabled)
		require.Empty(t, validateRedisTransport(cfg))
		tlsCfg, err := RedisTLSConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, "redis.internal", tlsCfg.ServerName)
		require.False(t, tlsCfg.InsecureSkipVerify)
		require.Nil(t, tlsCfg.RootCAs, "system roots are used without REDIS_TLS_CA_FILE")
	}

	badCA := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(badCA, []byte("not a certificate"), 0o600))
	t.Setenv("REDIS_TLS_CA_FILE", badCA)
	require.NotEmpty(t, validateRedisTransport(getArgsFromFlags()))
	t.Setenv("REDIS_TLS_CA_FILE", badCA+".missing")
	require.NotEmpty(t, validateRedisTransport(getArgsFromFlags()))

	// TLS options without the switch would be silently ignored.
	t.Setenv("REDIS_TLS_ENABLED", "")
	require.NotEmpty(t, validateRedisTransport(getArgsFromFlags()))

	// A typo must not select a plaintext connection.
	t.Setenv("REDIS_TLS_CA_FILE", "")
	t.Setenv("REDIS_TLS_SERVER_NAME", "")
	t.Setenv("REDIS_TLS_ENABLED", "enable")
	cfg := getArgsFromFlags()
	require.False(t, cfg.RedisTLSEnabled)
	err := ValidateConfig(cfg)
	require.ErrorContains(t, err, "REDIS_TLS_ENABLED")
}
