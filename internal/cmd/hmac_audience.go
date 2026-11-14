package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/soulteary/warden/internal/define"
)

// maxHMACAudiences bounds WARDEN_HMAC_AUDIENCES. Every v3 request is checked
// against each entry, and an instance has few names.
const maxHMACAudiences = 16

// maxHMACAudienceLength is the longest audience accepted; a DNS name with a
// port fits.
const maxHMACAudienceLength = 255

// ParseHMACAudiences parses WARDEN_HMAC_AUDIENCES: a comma-separated list of
// the names this instance accepts in a v3 signature. An entry is 1 to 255
// printable ASCII characters without spaces or commas; it is lower-cased, and
// duplicates are dropped. The empty string yields no audience.
func ParseHMACAudiences(raw string) ([]string, error) {
	var audiences []string
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if len(entry) > maxHMACAudienceLength {
			return nil, fmt.Errorf("an audience is longer than %d characters", maxHMACAudienceLength)
		}
		for i := 0; i < len(entry); i++ {
			if entry[i] <= ' ' || entry[i] > '~' {
				return nil, fmt.Errorf("an audience must consist of printable ASCII characters without spaces")
			}
		}
		if seen[entry] {
			continue
		}
		seen[entry] = true
		audiences = append(audiences, entry)
	}
	if len(audiences) > maxHMACAudiences {
		return nil, fmt.Errorf("at most %d audiences are accepted", maxHMACAudiences)
	}
	return audiences, nil
}

// ParseHMACRequireAudience parses WARDEN_HMAC_REQUIRE_AUDIENCE. The empty
// value is false: unbound signatures stay accepted unless the operator asks
// otherwise.
func ParseHMACRequireAudience(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	required, ok := define.ParseBool(raw)
	if !ok {
		return false, fmt.Errorf("expected a boolean (%s)", define.BooleanSpellings)
	}
	return required, nil
}

// validateHMACAudience checks the audience settings. An unrecognized
// WARDEN_HMAC_REQUIRE_AUDIENCE is reported by the boolean validation.
func validateHMACAudience() []string {
	var errs []string
	audiences, err := ParseHMACAudiences(os.Getenv("WARDEN_HMAC_AUDIENCES"))
	if err != nil {
		errs = append(errs, fmt.Sprintf("Invalid WARDEN_HMAC_AUDIENCES: %v", err))
	}
	if required, requireErr := ParseHMACRequireAudience(os.Getenv("WARDEN_HMAC_REQUIRE_AUDIENCE")); requireErr == nil && required && err == nil && len(audiences) == 0 {
		errs = append(errs, "WARDEN_HMAC_REQUIRE_AUDIENCE requires WARDEN_HMAC_AUDIENCES: without an audience every HMAC request would be rejected")
	}
	return errs
}
