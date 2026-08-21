package define

import (
	"strings"
	"sync/atomic"

	secure "github.com/soulteary/secure-kit"
)

// UserIDStrategy selects how an empty user_id is derived during Normalize.
type UserIDStrategy string

const (
	// UserIDStrategyLegacy preserves the historical behavior: the first 16 hex
	// characters (~64 bits) of SHA-256(identifier). Kept as the default so existing
	// deployments do not silently change user IDs.
	UserIDStrategyLegacy UserIDStrategy = "legacy"
	// UserIDStrategySHA256_128 derives a 128-bit (32 hex char) id from SHA-256 over a
	// domain-separated, normalized input. Opt-in via USER_ID_STRATEGY=sha256-128.
	UserIDStrategySHA256_128 UserIDStrategy = "sha256-128"

	// userIDDomainPhone / userIDDomainMail provide domain separation so a phone and a
	// mail that happen to share a string cannot collide across fields.
	userIDDomainPhone = "warden:user-id:v1:phone:"
	userIDDomainMail  = "warden:user-id:v1:mail:"
)

// identityConfig holds process-wide identity behavior. It is configured once at
// startup (before serving traffic). Reads are atomic so background refresh and
// request paths observe a consistent value.
var (
	userIDStrategy        atomic.Pointer[UserIDStrategy]
	requireExplicitUserID atomic.Bool
)

func init() {
	def := UserIDStrategyLegacy
	userIDStrategy.Store(&def)
}

// ParseUserIDStrategy parses a strategy string, defaulting to legacy for empty input.
func ParseUserIDStrategy(s string) (UserIDStrategy, bool) {
	switch UserIDStrategy(strings.ToLower(strings.TrimSpace(s))) {
	case "", UserIDStrategyLegacy:
		return UserIDStrategyLegacy, true
	case UserIDStrategySHA256_128:
		return UserIDStrategySHA256_128, true
	default:
		return UserIDStrategyLegacy, false
	}
}

// SetUserIDStrategy sets the process-wide user_id derivation strategy.
func SetUserIDStrategy(s UserIDStrategy) {
	v := s
	userIDStrategy.Store(&v)
}

// GetUserIDStrategy returns the current user_id derivation strategy.
func GetUserIDStrategy() UserIDStrategy {
	if p := userIDStrategy.Load(); p != nil {
		return *p
	}
	return UserIDStrategyLegacy
}

// SetRequireExplicitUserID toggles whether Normalize refuses to derive a missing id.
func SetRequireExplicitUserID(v bool) { requireExplicitUserID.Store(v) }

// RequireExplicitUserID reports whether an explicit user_id is required.
func RequireExplicitUserID() bool { return requireExplicitUserID.Load() }

// deriveUserID computes a user_id from the identifier fields per the active strategy.
// It returns an empty string when there is no usable identifier. It never changes an
// already-present id (callers only invoke this when UserID is empty).
func deriveUserID(phone, mail string) string {
	p := strings.TrimSpace(phone)
	m := strings.TrimSpace(strings.ToLower(mail))
	switch GetUserIDStrategy() {
	case UserIDStrategySHA256_128:
		switch {
		case p != "":
			return secure.GetSHA256Hash(userIDDomainPhone + p)[:32]
		case m != "":
			return secure.GetSHA256Hash(userIDDomainMail + m)[:32]
		default:
			return ""
		}
	default: // legacy
		identifier := p
		if identifier == "" {
			identifier = m
		}
		if identifier == "" {
			return ""
		}
		return secure.GetSHA256Hash(identifier)[:16]
	}
}
