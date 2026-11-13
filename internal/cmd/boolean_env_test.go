package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var booleanSettings = map[string]func(*Config) bool{
	"REDIS_ENABLED":                  func(cfg *Config) bool { return cfg.RedisEnabled },
	"REDIS_TLS_ENABLED":              func(cfg *Config) bool { return cfg.RedisTLSEnabled },
	"HTTP_INSECURE_TLS":              func(cfg *Config) bool { return cfg.HTTPInsecureTLS },
	"REMOTE_DECRYPT_ENABLED":         func(cfg *Config) bool { return cfg.RemoteDecryptEnabled },
	"REMOTE_ENCRYPTION_REQUIRED":     func(cfg *Config) bool { return cfg.RemoteEncryptionRequired },
	"REQUIRE_EXPLICIT_USER_ID":       func(cfg *Config) bool { return cfg.RequireExplicitUserID },
	"WARDEN_TLS_REQUIRE_CLIENT_CERT": func(cfg *Config) bool { return cfg.TLSRequireClientCert },
}

func clearBooleanEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CONFIG", "KEY", "MODE", "MERGE_MODE", "ENVIRONMENT", "CONFIG_FILE", "REDIS", "API_KEY", "DATA_DIR", "HTTP_TIMEOUT", "INTERVAL", "PORT"} {
		t.Setenv(key, "")
	}
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	os.Args = []string{"warden"}
	for name := range booleanSettings {
		t.Setenv(name, "")
	}
	for _, name := range []string{"WARDEN_HMAC_ALLOW_V1", "WARDEN_METRICS_REQUIRE_AUTH", "OTLP_ENABLED"} {
		t.Setenv(name, "")
	}
}

// Every boolean setting accepts the same spellings.
func TestBooleanSettingsAcceptTheSameSpellings(t *testing.T) {
	for name, read := range booleanSettings {
		for raw, want := range map[string]bool{"true": true, "1": true, "yes": true, "ON": true, "T": true, "false": false, "0": false, "no": false, "Off": false, "f": false} {
			clearBooleanEnv(t)
			t.Setenv(name, raw)
			cfg, err := GetArgsWithError()
			require.NoError(t, err)
			require.Equal(t, want, read(cfg), "%s=%s", name, raw)
		}
	}
}

// A value that is not a boolean stops startup and names the variable.
func TestUnrecognizedBooleanStopsStartup(t *testing.T) {
	names := []string{"WARDEN_HMAC_ALLOW_V1", "WARDEN_METRICS_REQUIRE_AUTH", "OTLP_ENABLED"}
	for name := range booleanSettings {
		names = append(names, name)
	}
	for _, name := range names {
		for _, raw := range []string{"enabled", "2", "ture"} {
			clearBooleanEnv(t)
			t.Setenv(name, raw)
			cfg, err := GetArgsWithError()
			require.NoError(t, err)
			err = ValidateConfig(cfg)
			require.Error(t, err, "%s=%s", name, raw)
			require.Contains(t, err.Error(), "Invalid "+name, "%s=%s", name, raw)
		}
	}
}
