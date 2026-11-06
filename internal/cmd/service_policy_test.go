package cmd

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateServicePolicies(t *testing.T) {
	cfg := &Config{HMACKeys: `{"gateway":"secret"}`}
	t.Setenv("WARDEN_SERVICE_POLICIES", "")
	require.Empty(t, validateServicePolicies(cfg))
	t.Setenv("WARDEN_SERVICE_POLICIES", `{"gateway":{"actions":["directory:lookup"]}}`)
	t.Setenv("WARDEN_HMAC_ALLOW_V1", "")
	require.Empty(t, validateServicePolicies(cfg))
	cfg.HMACKeys = "bad"
	require.NotEmpty(t, validateServicePolicies(cfg))
	cfg.HMACKeys = `{"gateway":"secret"}`
	t.Setenv("WARDEN_SERVICE_POLICIES", `{}`)
	require.NotEmpty(t, validateServicePolicies(cfg))
	t.Setenv("WARDEN_SERVICE_POLICIES", `{"gateway":{"actions":["directory:lookup"]}}`)
	for _, value := range []string{"true", "bad"} {
		t.Setenv("WARDEN_HMAC_ALLOW_V1", value)
		require.NotEmpty(t, validateServicePolicies(cfg))
	}
	t.Setenv("WARDEN_HMAC_ALLOW_V1", "")
	cfg.APIKey = "key"
	require.NotEmpty(t, validateServicePolicies(cfg))
	cfg.APIKey = ""
	cfg.TLSCAFile = "ca.pem"
	require.NotEmpty(t, validateServicePolicies(cfg))
	cfg.TLSCAFile = ""
	cfg.TLSRequireClientCert = true
	require.NotEmpty(t, validateServicePolicies(cfg))
}
