package define

import (
	"fmt"
	"os"
	"strings"
)

// BooleanEnvVars are the environment variables that hold a boolean. Startup
// validation rejects an unrecognized value in any of them (see
// ValidateBooleanEnv), so a typo cannot silently turn a setting off.
var BooleanEnvVars = []string{
	"REDIS_ENABLED",
	"REDIS_TLS_ENABLED",
	"HTTP_INSECURE_TLS",
	"REMOTE_DECRYPT_ENABLED",
	"REMOTE_ENCRYPTION_REQUIRED",
	"REQUIRE_EXPLICIT_USER_ID",
	"WARDEN_TLS_REQUIRE_CLIENT_CERT",
	"WARDEN_HMAC_ALLOW_V1",
	"WARDEN_HMAC_REQUIRE_AUDIENCE",
	"WARDEN_METRICS_REQUIRE_AUTH",
	"OTLP_ENABLED",
}

// BooleanSpellings describes the accepted boolean values in error messages.
const BooleanSpellings = "true/false, 1/0, yes/no, on/off or t/f"

// ParseBool parses a boolean setting. It accepts true/false, 1/0, yes/no,
// on/off and t/f, ignoring case and surrounding space. ok is false for any
// other text, including the empty string, and value is then false.
func ParseBool(raw string) (value, ok bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on", "t":
		return true, true
	case "false", "0", "no", "off", "f":
		return false, true
	default:
		return false, false
	}
}

// EnvBool reads a boolean environment variable. An unset or empty variable
// yields fallback. An unrecognized value yields false: startup validation has
// already rejected it, and a caller that skipped validation must not have a
// typo read as "enabled".
func EnvBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, _ := ParseBool(raw)
	return value
}

// ValidateBooleanEnv returns one message for every boolean environment
// variable that is set to something ParseBool does not recognize.
func ValidateBooleanEnv() []string {
	var errs []string
	for _, name := range BooleanEnvVars {
		raw := strings.TrimSpace(os.Getenv(name))
		if raw == "" {
			continue
		}
		if _, ok := ParseBool(raw); !ok {
			errs = append(errs, fmt.Sprintf("Invalid %s %q: expected %s", name, raw, BooleanSpellings))
		}
	}
	return errs
}
