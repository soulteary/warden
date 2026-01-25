// Package validator provides configuration validation functionality.
// This package wraps cli-kit/validator for backward compatibility.
package validator

import (
	"github.com/soulteary/cli-kit/validator"
)

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
	// Use cli-kit validator with default options (SSRF protection enabled)
	return validator.ValidateURL(urlStr, nil)
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
