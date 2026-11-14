package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// MaxHTTPTimeoutSeconds is the largest HTTP timeout accepted from any source.
// The timeout is multiplied into a time.Duration and added to the refresh
// deadline and to the lifetime of the Redis writer lock; an unbounded value
// overflows that arithmetic. An hour is far above any useful request timeout.
const MaxHTTPTimeoutSeconds = 3600

// ParseHTTPTimeout parses HTTP_TIMEOUT: a whole number of seconds ("30") or a
// Go duration ("30s", "1m30s"), from one second to MaxHTTPTimeoutSeconds.
func ParseHTTPTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	timeout, err := time.ParseDuration(raw)
	if err != nil {
		seconds, convErr := strconv.Atoi(raw)
		if convErr != nil {
			return 0, fmt.Errorf("must be a number of seconds or a duration such as 30s, got %q", raw)
		}
		if seconds > MaxHTTPTimeoutSeconds {
			return 0, fmt.Errorf("must be at most %d seconds, got %d", MaxHTTPTimeoutSeconds, seconds)
		}
		timeout = time.Duration(seconds) * time.Second
	}
	if timeout < time.Second {
		return 0, fmt.Errorf("must be at least 1s, got %s", timeout)
	}
	if timeout > MaxHTTPTimeoutSeconds*time.Second {
		return 0, fmt.Errorf("must be at most %s, got %s", MaxHTTPTimeoutSeconds*time.Second, timeout)
	}
	return timeout, nil
}

// integerEnvRule describes an environment variable that holds an integer.
type integerEnvRule struct {
	name     string
	expected string
	minimum  int
	maximum  int // zero means no upper bound
}

// integerEnvRules are the integer settings read from the environment. Each was
// read with a fallback: text that is not a number selected the default without
// a word, so PORT=abc listened on 8081.
var integerEnvRules = []integerEnvRule{
	{name: "PORT", minimum: 1, maximum: 65535, expected: "an integer between 1 and 65535"},
	{name: "INTERVAL", minimum: 1, expected: "a positive integer (seconds)"},
	{name: "HTTP_MAX_IDLE_CONNS", minimum: 0, expected: "a non-negative integer"},
	{name: "WARDEN_HMAC_TIMESTAMP_TOLERANCE", minimum: 1, expected: "a positive integer (seconds)"},
}

// validateNumericEnv rejects numeric environment variables that hold text the
// service cannot use, and an HTTP timeout outside the accepted range from any
// source. An empty variable is unset. The variables are checked as given, also
// when a command-line flag overrides them.
func validateNumericEnv(cfg *Config) []string {
	var errs []string
	for _, rule := range integerEnvRules {
		raw := strings.TrimSpace(os.Getenv(rule.name))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < rule.minimum || (rule.maximum > 0 && value > rule.maximum) {
			errs = append(errs, fmt.Sprintf("Invalid %s %q: expected %s", rule.name, raw, rule.expected))
		}
	}
	if raw := strings.TrimSpace(os.Getenv("HTTP_TIMEOUT")); raw != "" {
		if _, err := ParseHTTPTimeout(raw); err != nil {
			errs = append(errs, fmt.Sprintf("Invalid HTTP_TIMEOUT: %v", err))
		}
	}
	// The flag and the configuration file reach the same field.
	if cfg.HTTPTimeout < 0 || cfg.HTTPTimeout > MaxHTTPTimeoutSeconds {
		errs = append(errs, fmt.Sprintf("Invalid HTTP timeout: %d seconds (must be between 0 and %d)", cfg.HTTPTimeout, MaxHTTPTimeoutSeconds))
	}
	return errs
}
