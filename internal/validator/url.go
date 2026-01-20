// Package validator provides configuration validation functionality.
// Includes URL validation, path validation and other security validation features.
package validator

import (
	// Standard library
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// ValidateRemoteURL validates remote configuration URL to prevent SSRF attacks
//
// This function performs strict validation on remote configuration URL, including:
// - Only allows http:// and https:// protocols
// - Prohibits access to private IP addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8)
// - Prohibits access to localhost
// - Validates URL format validity
//
// Parameters:
//   - urlStr: URL string to validate
//
// Returns:
//   - error: returns error if URL is invalid or has security risks; otherwise returns nil
func ValidateRemoteURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse URL
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow http and https protocols
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https protocols are allowed, current protocol: %s", u.Scheme)
	}

	// Validate host
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must contain a valid host")
	}

	// Prohibit localhost and 127.0.0.1
	hostLower := strings.ToLower(host)
	if hostLower == "localhost" || hostLower == "127.0.0.1" || hostLower == "::1" {
		return fmt.Errorf("access to localhost is not allowed")
	}

	// Parse IP address
	ip := net.ParseIP(host)
	if ip != nil {
		// Prohibit private IP addresses
		if isPrivateIP(ip) {
			return fmt.Errorf("access to private IP address is not allowed: %s", ip.String())
		}
		// Prohibit loopback addresses
		if ip.IsLoopback() {
			return fmt.Errorf("access to loopback address is not allowed: %s", ip.String())
		}
	}

	return nil
}

// isPrivateIP checks if IP is a private IP
//
// Private IP address ranges:
// - 10.0.0.0/8 (10.0.0.0 to 10.255.255.255)
// - 172.16.0.0/12 (172.16.0.0 to 172.31.255.255)
// - 192.168.0.0/16 (192.168.0.0 to 192.168.255.255)
// - 127.0.0.0/8 (127.0.0.0 to 127.255.255.255) - loopback address
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			ip4[0] == 127
	}
	// IPv6 private address check
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

// ValidateConfigPath validates configuration file path to prevent path traversal attacks
//
// This function validates configuration file path, including:
// - Checks if path contains path traversal characters (..)
// - Validates if path is absolute or relative path
// - Optional: restrict configuration files to be read only from specific directories
//
// Parameters:
//   - path: file path to validate
//   - allowedDirs: list of allowed directories (optional, if empty then no directory restriction)
//
// Returns:
//   - string: normalized absolute path
//   - error: returns error if path is invalid or has security risks; otherwise returns nil
func ValidateConfigPath(path string, allowedDirs []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("configuration file path cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to parse configuration file path: %w", err)
	}

	// Check if contains path traversal
	if strings.Contains(absPath, "..") {
		return "", fmt.Errorf("configuration file path cannot contain path traversal characters (..)")
	}

	// If allowed directories are specified, check if path is under allowed directories
	if len(allowedDirs) > 0 {
		allowed := false
		for _, allowedDir := range allowedDirs {
			allowedAbsDir, err := filepath.Abs(allowedDir)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, allowedAbsDir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("configuration file must be under allowed directories: %v", allowedDirs)
		}
	}

	return absPath, nil
}
