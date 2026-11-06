package cmd

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProductionRejectsHTTPRuleOrigins(t *testing.T) {
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "http://10.0.0.2:8080")
	for _, origin := range []string{"http://93.184.216.34/data.json", "http://10.0.0.2:8080/data.json", define.DEFAULT_REMOTE_CONFIG} {
		cfg := &Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_REMOTE", Environment: "production", RemoteConfig: origin, HTTPTimeout: 30, RemoteEncryptionRequired: true, RemoteDecryptEnabled: true, RemoteRSAPrivateKey: "test-key-pem", RemoteEncryptionFormat: "v2", APIKey: "service-key"}
		require.ErrorContains(t, ValidateConfig(cfg), "requires HTTPS", origin)
		cfg.Environment = "development"
		require.NoError(t, ValidateConfig(cfg), "development HTTP remains explicit compatibility")
	}
}
