// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/metrics"
)

// RateLimiter implements simple in-memory rate limiting
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type RateLimiter struct {
	mu           sync.RWMutex        // 24 bytes
	wg           sync.WaitGroup      // 12 bytes (padding to 16)
	window       time.Duration       // 8 bytes
	rate         int                 // 8 bytes
	maxVisitors  int                 // 8 bytes
	maxWhitelist int                 // 8 bytes
	visitors     map[string]*visitor // 8 bytes pointer
	whitelist    map[string]bool     // 8 bytes pointer
	cleanup      *time.Ticker        // 8 bytes pointer
	stopCh       chan struct{}       // 8 bytes pointer
	stopOnce     sync.Once           // Ensures Stop is executed only once
}

type visitor struct {
	lastSeen time.Time // 24 bytes
	count    int       // 8 bytes
}

// NewRateLimiter creates a new rate limiter
// rate: number of allowed requests
// window: time window (e.g., 1 * time.Minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:     make(map[string]*visitor),
		rate:         rate,
		window:       window,
		cleanup:      time.NewTicker(define.RATE_LIMIT_CLEANUP_INTERVAL), // Periodically clean up expired records
		stopCh:       make(chan struct{}),
		whitelist:    make(map[string]bool),
		maxVisitors:  define.MAX_VISITORS_MAP_SIZE,
		maxWhitelist: define.MAX_WHITELIST_SIZE,
	}

	// Start cleanup goroutine
	rl.wg.Add(1)
	go rl.cleanupVisitors()

	return rl
}

// cleanupVisitors periodically cleans up expired access records
func (rl *RateLimiter) cleanupVisitors() {
	defer rl.wg.Done()
	for {
		select {
		case <-rl.cleanup.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			// If still exceeds limit, clean up oldest records
			if len(rl.visitors) > rl.maxVisitors {
				rl.cleanupOldestVisitors()
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// cleanupOldestVisitors cleans up oldest access records (when exceeding maximum limit)
func (rl *RateLimiter) cleanupOldestVisitors() {
	// Find oldest records
	type visitorWithTime struct {
		lastSeen time.Time // 24 bytes
		ip       string    // 16 bytes
	}
	visitors := make([]visitorWithTime, 0, len(rl.visitors))
	for ip, v := range rl.visitors {
		visitors = append(visitors, visitorWithTime{ip: ip, lastSeen: v.lastSeen})
	}

	// Sort by time, delete oldest
	sort.Slice(visitors, func(i, j int) bool {
		return visitors[i].lastSeen.Before(visitors[j].lastSeen)
	})

	// Delete oldest records until below limit
	toRemove := len(rl.visitors) - rl.maxVisitors
	for i := 0; i < toRemove && i < len(visitors); i++ {
		delete(rl.visitors, visitors[i].ip)
	}
}

// Allow checks if request is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.RLock()
	// Check whitelist
	if rl.whitelist[ip] {
		rl.mu.RUnlock()
		return true
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		// Check if exceeds maximum limit
		if len(rl.visitors) >= rl.maxVisitors {
			// Clean up oldest records
			rl.cleanupOldestVisitors()
		}
		rl.visitors[ip] = &visitor{
			count:    1,
			lastSeen: now,
		}
		return true
	}

	// If exceeds time window, reset count
	if now.Sub(v.lastSeen) > rl.window {
		v.count = 1
		v.lastSeen = now
		return true
	}

	// Check if exceeds limit
	if v.count >= rl.rate {
		// Record rate limit metrics
		metrics.RateLimitHits.WithLabelValues(ip).Inc()
		return false
	}

	v.count++
	v.lastSeen = now
	return true
}

// AddToWhitelist adds IP to whitelist
func (rl *RateLimiter) AddToWhitelist(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if already exists
	if rl.whitelist[ip] {
		return true
	}

	// Check if exceeds maximum limit
	if len(rl.whitelist) >= rl.maxWhitelist {
		return false // Whitelist is full
	}

	rl.whitelist[ip] = true
	return true
}

// RemoveFromWhitelist removes IP from whitelist
func (rl *RateLimiter) RemoveFromWhitelist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.whitelist, ip)
}

// IsWhitelisted checks if IP is in whitelist
func (rl *RateLimiter) IsWhitelisted(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.whitelist[ip]
}

// Stop stops the rate limiter
// Uses sync.Once to ensure it's executed only once, avoiding panic from repeatedly closing channel
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		rl.cleanup.Stop()
		close(rl.stopCh)
		rl.wg.Wait() // Wait for goroutine to exit
	})
}

// RateLimitMiddlewareWithLimiter creates rate limiting middleware (using specified RateLimiter instance)
// Recommended to use this method to avoid global variables
func RateLimitMiddlewareWithLimiter(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP (secure method)
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				hlog.FromRequest(r).Warn().
					Str("ip", ip).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Str("user_agent", r.UserAgent()).
					Str("referer", r.Referer()).
					Msg(i18n.T(r, "error.rate_limit_exceeded"))
				http.Error(w, i18n.T(r, "http.rate_limit_exceeded"), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP safely gets client IP address
// Improved IP retrieval logic with stricter validation
func getClientIP(r *http.Request) string {
	// Prefer X-Real-IP (usually set by reverse proxy, more reliable)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		ip := strings.TrimSpace(realIP)
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			// Verify if it's a private IP (if so, may indicate configuration issue)
			if !isPrivateIP(parsedIP) || isTrustedProxy(r) {
				return ip
			}
		}
	}

	// Next use X-Forwarded-For, but needs validation
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP (may be proxy chain)
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			// Validate IP format
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				// Only use X-Forwarded-For if proxy is trusted
				if isTrustedProxy(r) {
					return ip
				}
			}
		}
	}

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	// Validate IP in RemoteAddr
	if parsedIP := net.ParseIP(host); parsedIP != nil {
		return host
	}

	return r.RemoteAddr
}

// isPrivateIP checks if IP is a private IP
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			ip4[0] == 127
	}
	return false
}

// trustedProxyIPs trusted proxy IP list (read from environment variable)
var trustedProxyIPs = loadTrustedProxyIPs()

// loadTrustedProxyIPs loads trusted proxy IP list from environment variable
func loadTrustedProxyIPs() map[string]bool {
	trustedIPs := make(map[string]bool)

	// Read from environment variable TRUSTED_PROXY_IPS, supports comma-separated multiple IPs
	trustedIPsEnv := os.Getenv("TRUSTED_PROXY_IPS")
	if trustedIPsEnv != "" {
		ips := strings.Split(trustedIPsEnv, ",")
		for _, ipStr := range ips {
			ipStr = strings.TrimSpace(ipStr)
			if ip := net.ParseIP(ipStr); ip != nil {
				trustedIPs[ip.String()] = true
			}
		}
	}

	return trustedIPs
}

// isTrustedProxy checks if request is from a trusted proxy
// In production environment, should configure trusted proxy IPs according to actual deployment
func isTrustedProxy(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check if in trusted proxy IP list
	if len(trustedProxyIPs) > 0 {
		if trustedProxyIPs[ip.String()] {
			return true
		}
	}

	// If no trusted proxy IP list is configured, default to allow private IPs (backward compatibility)
	// In production environment, recommend explicitly configuring trusted proxy IPs via environment variable
	return isPrivateIP(ip)
}
