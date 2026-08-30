// Package validator provides configuration validation functionality.
// This package wraps cli-kit/validator for backward compatibility.
package validator

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/soulteary/cli-kit/validator"
)

const remoteURLResolveTimeout = 5 * time.Second

type ipResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// ValidateRemoteURL validates remote configuration URL to prevent SSRF attacks
//
// This function performs strict validation on remote configuration URL, including:
// - Only allows http:// and https:// protocols
// - Prohibits access to private IP addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8)
// - Prohibits access to localhost
// - Validates URL format validity
//
// This function delegates to cli-kit/validator.ValidateURL with secure defaults.
//
// Parameters:
//   - urlStr: URL string to validate
//
// Returns:
//   - error: returns error if URL is invalid or has security risks; otherwise returns nil
func ValidateRemoteURL(urlStr string) error {
	return validateRemoteURL(urlStr, net.DefaultResolver)
}

func validateRemoteURL(urlStr string, resolver ipResolver) error {
	// First validate syntax, scheme, localhost, and literal IPs without touching
	// DNS. Hostnames are resolved below through the injected resolver so tests
	// can exercise the same SSRF checks without external network access.
	opts := &validator.URLOptions{ResolveHostTimeout: 0}
	if err := validator.ValidateURL(urlStr, opts); err != nil {
		return err
	}

	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	host := u.Hostname()
	if net.ParseIP(host) != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), remoteURLResolveTimeout)
	defer cancel()
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("host %q resolved to no addresses", host)
	}

	for _, addr := range addrs {
		resolved := *u
		resolved.Host = resolvedURLHost(addr.IP.String(), u.Port())
		if err := validator.ValidateURL(resolved.String(), opts); err != nil {
			return err
		}
	}
	return nil
}

func resolvedURLHost(ip, port string) string {
	if port != "" {
		return net.JoinHostPort(ip, port)
	}
	if strings.Contains(ip, ":") {
		return "[" + ip + "]"
	}
	return ip
}

// ValidateConfigPath validates configuration file path to prevent path traversal attacks
//
// This function validates configuration file path, including:
// - Checks if path contains path traversal characters (..)
// - Validates if path is absolute or relative path
// - Optional: restrict configuration files to be read only from specific directories
//
// This function delegates to cli-kit/validator.ValidatePath.
//
// Parameters:
//   - path: file path to validate
//   - allowedDirs: list of allowed directories (optional, if empty then no directory restriction)
//
// Returns:
//   - string: normalized absolute path
//   - error: returns error if path is invalid or has security risks; otherwise returns nil
func ValidateConfigPath(path string, allowedDirs []string) (string, error) {
	opts := &validator.PathOptions{
		AllowRelative:  true,
		CheckTraversal: true,
		AllowedDirs:    allowedDirs,
	}
	return validator.ValidatePath(path, opts)
}
