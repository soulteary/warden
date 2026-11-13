package middleware

import (
	middlewarekit "github.com/soulteary/middleware-kit/v3"

	"github.com/soulteary/warden/internal/define"
)

// noTrustedProxy is the trusted-proxy list used for TRUSTED_PROXY_IPS=none.
// middleware-kit treats an empty list as "trust every private or loopback peer"
// and has no setting for "trust nobody", so the list holds the unspecified
// address instead: it is never the source address of an accepted connection,
// so no peer matches, and the list is not empty, so the private-address
// default does not apply.
var noTrustedProxy = []string{"0.0.0.0/32"}

// NewTrustedProxyConfig builds the client-address policy from a TRUSTED_PROXY_IPS
// value. An empty value keeps the middleware-kit default of trusting forwarded
// headers from private and loopback peers; "none" trusts no peer.
func NewTrustedProxyConfig(raw string) *middlewarekit.TrustedProxyConfig {
	if define.TrustedProxyDisabled(raw) {
		return middlewarekit.NewTrustedProxyConfig(noTrustedProxy)
	}
	return middlewarekit.NewTrustedProxyConfig(define.ParseTrustedProxyIPs(raw))
}
