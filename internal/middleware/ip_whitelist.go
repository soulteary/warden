// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	"net/http"
	"os"

	middlewarekit "github.com/soulteary/middleware-kit"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// IPWhitelistMiddleware creates IP whitelist middleware.
//
// This middleware only allows IP addresses in the whitelist to access protected endpoints.
// Whitelist is taken from the whitelist argument, or from environment variable IP_WHITELIST when whitelist is empty.
// Supports comma-separated multiple IPs or CIDR networks.
// Uses middleware-kit IPAllowlistMiddleware under the hood. Client IP is resolved via TrustedProxyConfig
// from TRUSTED_PROXY_IPS so behaviour matches other middleware-kit usage.
//
// Parameters:
//   - whitelist: IP whitelist (comma-separated IP addresses or CIDR networks). Empty means use IP_WHITELIST env.
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func IPWhitelistMiddleware(whitelist string) func(http.Handler) http.Handler {
	if whitelist == "" {
		whitelist = os.Getenv("IP_WHITELIST")
	}
	trustedCfg := middlewarekit.NewTrustedProxyConfig(define.ParseTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS")))
	return middlewarekit.IPAllowlistMiddlewareFromConfig(middlewarekit.IPAllowlistConfig{
		Allowlist:          whitelist,
		TrustedProxyConfig: trustedCfg,
		OnDenied: func(w http.ResponseWriter, r *http.Request) {
			clientIP := middlewarekit.GetClientIP(r, trustedCfg)
			logger.FromRequest(r).Warn().
				Str("ip", clientIP).
				Str("path", r.URL.Path).
				Msg("IP not in whitelist, access denied")
			http.Error(w, "Forbidden", http.StatusForbidden)
		},
	})
}
