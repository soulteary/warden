// Package logger provides logging functionality.
// Implements structured logging based on zerolog, supports log level control and sensitive information sanitization.
package logger

import (
	// Standard library
	"os"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog"
)

var globalLevel zerolog.Level = zerolog.InfoLevel

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Read log level from environment variable
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
		if err == nil {
			globalLevel = level
		}
	}

	zerolog.SetGlobalLevel(globalLevel)
}

// GetLogger gets logger instance
func GetLogger() zerolog.Logger {
	logger := zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger().
		Level(globalLevel)

	return logger
}

// SetLevel sets log level (for runtime adjustment)
func SetLevel(level zerolog.Level) {
	globalLevel = level
	zerolog.SetGlobalLevel(level)
}

// SanitizeString sanitizes sensitive information
// Performs partial sanitization on strings that may contain sensitive information
func SanitizeString(s string) string {
	if s == "" {
		return s
	}

	// If string is short, only show first and last characters
	if len(s) <= 4 {
		return "***"
	}

	// Show first 2 characters and last 2 characters, replace middle with *
	prefix := s[:2]
	suffix := s[len(s)-2:]
	masked := strings.Repeat("*", len(s)-4)
	return prefix + masked + suffix
}

// SanitizeHeader sanitizes HTTP header information
func SanitizeHeader(header string) string {
	// Sanitize sensitive headers like Authorization
	lowerHeader := strings.ToLower(header)
	if strings.Contains(lowerHeader, "authorization") ||
		strings.Contains(lowerHeader, "token") ||
		strings.Contains(lowerHeader, "api-key") {
		return SanitizeString(header)
	}
	return header
}
