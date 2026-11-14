package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The configuration-file path reads the boolean environment variables with the
// same spellings as the flag path.
func TestConfigFileBooleanEnvSpellings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("http:\n  insecure_tls: false\n"), 0o600))
	names := []string{"REDIS_ENABLED", "HTTP_INSECURE_TLS", "REMOTE_DECRYPT_ENABLED", "REMOTE_ENCRYPTION_REQUIRED", "REQUIRE_EXPLICIT_USER_ID", "WARDEN_TLS_REQUIRE_CLIENT_CERT", "OTLP_ENABLED"}
	for raw, want := range map[string]bool{"yes": true, "On": true, "1": true, "TRUE": true, "no": false, "off": false, "0": false, "False": false} {
		for _, name := range names {
			t.Setenv(name, raw)
		}
		cfg, err := ParseFromFile(path)
		require.NoError(t, err)
		data := cfg.ToCmdConfig()
		got := map[string]bool{
			"REDIS_ENABLED":                  data.RedisEnabled,
			"HTTP_INSECURE_TLS":              data.HTTPInsecureTLS,
			"REMOTE_DECRYPT_ENABLED":         data.RemoteDecryptEnabled,
			"REMOTE_ENCRYPTION_REQUIRED":     data.RemoteEncryptionRequired,
			"REQUIRE_EXPLICIT_USER_ID":       data.RequireExplicitUserID,
			"WARDEN_TLS_REQUIRE_CLIENT_CERT": data.TLSRequireClientCert,
			"OTLP_ENABLED":                   cfg.Tracing.Enabled,
		}
		for _, name := range names {
			require.Equal(t, want, got[name], "%s=%s", name, raw)
		}
	}
}
