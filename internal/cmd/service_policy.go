package cmd

import (
	"os"
	"strings"

	"github.com/soulteary/warden/internal/serviceauth"
)

func validateServicePolicies(cfg *Config) []string {
	raw := os.Getenv("WARDEN_SERVICE_POLICIES")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	keys, err := ParseHMACKeys(cfg.HMACKeys)
	if err != nil {
		return []string{"WARDEN_SERVICE_POLICIES requires valid HMAC keys"}
	}
	if _, err = serviceauth.Parse(raw, keys); err != nil {
		return []string{err.Error()}
	}
	allowV1, err := ParseHMACAllowV1(os.Getenv("WARDEN_HMAC_ALLOW_V1"))
	if err != nil || allowV1 {
		return []string{"WARDEN_SERVICE_POLICIES requires HMAC v2 without legacy v1"}
	}
	if cfg.APIKey != "" || cfg.TLSCAFile != "" || cfg.TLSRequireClientCert {
		return []string{"WARDEN_SERVICE_POLICIES requires HMAC-only authentication; remove API key and client-certificate auth"}
	}
	return nil
}
