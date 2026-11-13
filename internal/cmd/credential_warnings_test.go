package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMissingCredentialWarnings(t *testing.T) {
	for _, environment := range []string{"", "development", "test"} {
		warnings := MissingCredentialWarnings(&Config{Environment: environment})
		require.Len(t, warnings, 1, environment)
		require.Contains(t, warnings[0], "API_KEY")
		require.Contains(t, warnings[0], "401")
	}
	for name, cfg := range map[string]*Config{
		"api key":    {Environment: "development", APIKey: "key"},
		"hmac keys":  {Environment: "development", HMACKeys: `{"gateway":"secret"}`},
		"client ca":  {Environment: "development", TLSCAFile: "/etc/warden/ca.pem"},
		"production": {Environment: "production"},
	} {
		require.Empty(t, MissingCredentialWarnings(cfg), name)
	}
	// A key set that does not parse provides no credential.
	require.Len(t, MissingCredentialWarnings(&Config{Environment: "development", HMACKeys: `{"gateway":""}`}), 1)
}
