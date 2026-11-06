package loader

import (
	"crypto/tls"
	"github.com/soulteary/warden/internal/cmd"
	"net/http"
)

// Local fixtures inject their transport explicitly; server startup never uses it.
func newTestRulesLoader(cfg *cmd.Config, mode string) (*RulesLoader, error) {
	insecure := cfg != nil && cfg.HTTPInsecureTLS
	// #nosec G402 -- local fixtures exercise the explicit development TLS option
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}}
	return NewRulesLoaderWithHTTPClient(cfg, mode, client)
}
