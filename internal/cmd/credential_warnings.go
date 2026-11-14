package cmd

import (
	"strings"

	"github.com/soulteary/warden/internal/config"
)

// MissingCredentialWarnings reports a configuration without any service
// credential outside production. Production refuses to start in that state;
// the other environments start, and then every data route answers 401 with
// nothing in the log to say why. The warning names the settings that provide
// a credential.
func MissingCredentialWarnings(cfg *Config) []string {
	environment, _ := config.ParseEnvironment(cfg.Environment)
	if environment.IsProduction() {
		return nil
	}
	// A client CA alone is enough: a verified client certificate authenticates
	// even when certificates are optional.
	if strings.TrimSpace(cfg.APIKey) != "" || hasHMACKeys(cfg.HMACKeys) || strings.TrimSpace(cfg.TLSCAFile) != "" {
		return nil
	}
	return []string{"no service credential is configured (API_KEY, WARDEN_HMAC_KEYS or a client CA in WARDEN_TLS_CA): the data routes and /log/level will answer 401 to every request; set API_KEY to serve them"}
}
