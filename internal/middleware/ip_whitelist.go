// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net"
	"net/http"
	"os"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"
)

// IPWhitelistMiddleware creates IP whitelist middleware
//
// This middleware only allows IP addresses in the whitelist to access protected endpoints.
// Whitelist is configured via environment variable IP_WHITELIST, supports comma-separated multiple IPs or CIDR networks.
//
// Parameters:
//   - whitelist: IP whitelist (comma-separated IP addresses or CIDR networks)
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func IPWhitelistMiddleware(whitelist string) func(http.Handler) http.Handler {
	// If whitelist is not configured, allow all IPs (backward compatibility)
	if whitelist == "" {
		whitelist = os.Getenv("IP_WHITELIST")
	}
	if whitelist == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Parse whitelist
	allowedIPs, allowedNetworks := parseIPWhitelist(whitelist)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			// Check if IP is in whitelist
			if !isIPAllowed(clientIP, allowedIPs, allowedNetworks) {
				hlog.FromRequest(r).Warn().
					Str("ip", clientIP).
					Str("path", r.URL.Path).
					Msg("IP not in whitelist, access denied")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseIPWhitelist parses IP whitelist
func parseIPWhitelist(whitelist string) (map[string]bool, []*net.IPNet) {
	allowedIPs := make(map[string]bool)
	var allowedNetworks []*net.IPNet

	ips := strings.Split(whitelist, ",")
	for _, ipStr := range ips {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// Try to parse as CIDR network
		if _, network, err := net.ParseCIDR(ipStr); err == nil {
			allowedNetworks = append(allowedNetworks, network)
			continue
		}

		// Try to parse as single IP
		if ip := net.ParseIP(ipStr); ip != nil {
			allowedIPs[ip.String()] = true
		}
	}

	return allowedIPs, allowedNetworks
}

// isIPAllowed checks if IP is in whitelist
func isIPAllowed(ipStr string, allowedIPs map[string]bool, allowedNetworks []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check if in single IP whitelist
	if allowedIPs[ip.String()] {
		return true
	}

	// Check if in any CIDR network
	for _, network := range allowedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
