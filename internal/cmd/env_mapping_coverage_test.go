package cmd

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEnvironmentMappingPreservesAbsentValuesAndParsesExplicitValues(t *testing.T) {
	values := map[string]string{"DATA_DIR": " /rules ", "RESPONSE_FIELDS": " user_id, , mail ,", "REMOTE_DECRYPT_ENABLED": "1", "REMOTE_RSA_PRIVATE_KEY_FILE": " /keys/rsa.pem ", "REMOTE_RSA_PRIVATE_KEY": " inline-key ", "REMOTE_ENCRYPTION_REQUIRED": "TRUE", "REMOTE_ENCRYPTION_FORMAT": " V2 ", "USER_ID_STRATEGY": " CANONICAL ", "REQUIRE_EXPLICIT_USER_ID": "1", "WARDEN_HMAC_KEYS": `{"key":"secret"}`, "WARDEN_HMAC_TIMESTAMP_TOLERANCE": "75", "WARDEN_TLS_CERT": " /tls/cert ", "WARDEN_TLS_KEY": " /tls/key ", "WARDEN_TLS_CA": " /tls/ca ", "WARDEN_TLS_REQUIRE_CLIENT_CERT": "true"}
	for key := range values {
		t.Setenv(key, "")
	}
	unchanged := &Config{DataDir: "retained", ResponseFields: []string{"retained"}, UserIDStrategy: "retained", HMACToleranceSec: 90}
	processDataDirFromEnv(unchanged)
	processResponseFieldsFromEnv(unchanged)
	processRemoteDecryptFromEnv(unchanged)
	processIdentityFromEnv(unchanged)
	processServiceAuthFromEnv(unchanged)
	require.Equal(t, "retained", unchanged.DataDir)
	require.Equal(t, []string{"retained"}, unchanged.ResponseFields)
	require.Equal(t, "retained", unchanged.UserIDStrategy)
	require.Equal(t, 90, unchanged.HMACToleranceSec)
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg := &Config{}
	processDataDirFromEnv(cfg)
	processResponseFieldsFromEnv(cfg)
	processRemoteDecryptFromEnv(cfg)
	processIdentityFromEnv(cfg)
	processServiceAuthFromEnv(cfg)
	require.Equal(t, "/rules", cfg.DataDir)
	require.Equal(t, []string{"user_id", "mail"}, cfg.ResponseFields)
	require.True(t, cfg.RemoteDecryptEnabled)
	require.True(t, cfg.RemoteEncryptionRequired)
	require.Equal(t, "v2", cfg.RemoteEncryptionFormat)
	require.Equal(t, "/keys/rsa.pem", cfg.RemoteRSAPrivateKeyFile)
	require.Equal(t, "inline-key", cfg.RemoteRSAPrivateKey)
	require.Equal(t, "canonical", cfg.UserIDStrategy)
	require.True(t, cfg.RequireExplicitUserID)
	require.Equal(t, 75, cfg.HMACToleranceSec)
	require.Equal(t, "/tls/cert", cfg.TLSCertFile)
	require.Equal(t, "/tls/key", cfg.TLSKeyFile)
	require.Equal(t, "/tls/ca", cfg.TLSCAFile)
	require.True(t, cfg.TLSRequireClientCert)
	t.Setenv("WARDEN_HMAC_TIMESTAMP_TOLERANCE", "invalid")
	fallback := &Config{HMACKeys: `{"key":"secret"}`}
	processServiceAuthFromEnv(fallback)
	require.Equal(t, defaultHMACToleranceSec, fallback.HMACToleranceSec)
	for _, key := range []string{"REMOTE_DECRYPT_ENABLED", "REMOTE_ENCRYPTION_REQUIRED", "REQUIRE_EXPLICIT_USER_ID", "WARDEN_TLS_REQUIRE_CLIENT_CERT"} {
		t.Setenv(key, "false")
	}
	cfg.RemoteDecryptEnabled = true
	cfg.RemoteEncryptionRequired = true
	cfg.RequireExplicitUserID = true
	cfg.TLSRequireClientCert = true
	processRemoteDecryptFromEnv(cfg)
	processIdentityFromEnv(cfg)
	processServiceAuthFromEnv(cfg)
	require.False(t, cfg.RemoteDecryptEnabled)
	require.False(t, cfg.RemoteEncryptionRequired)
	require.False(t, cfg.RequireExplicitUserID)
	require.False(t, cfg.TLSRequireClientCert)
}
