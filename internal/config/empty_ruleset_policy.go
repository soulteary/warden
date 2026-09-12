// Package config defines the EmptyRulesetPolicy type (empty_ruleset_policy.go).
//
// A load can succeed, return records, and still leave the effective rule set empty:
// identity validation runs over the whole set, but per-record format validation lives one
// layer down in the cache and silently drops records it rejects. An upstream that changes
// its phone-number formatting turns every record invalid at once while the load itself
// reports success.
//
// That state is genuinely ambiguous and the service cannot tell the two readings apart:
//
//	"the upstream revoked everyone"  -> the empty set is correct and must propagate
//	"the upstream changed format"    -> the empty set is an artifact and must not propagate
//
// EmptyRulesetPolicy makes the operator choose which reading to assume. A load that
// genuinely returns zero records is NOT ambiguous and always propagates under both
// policies — otherwise revoking every user would become impossible.
package config

import "strings"

// EmptyRulesetPolicy selects what a refresh does when a successful load returns records
// but none of them survive validation.
type EmptyRulesetPolicy string

const (
	// EmptyRulesetConsistencyFirst applies and publishes the empty effective set, keeping
	// every replica and the shared cache identical to what this process serves.
	//
	// Choose it when the upstream is the single source of truth and divergence is the
	// worse failure: a legitimate mass revocation propagates immediately, and an upstream
	// format regression surfaces at once as "everything is denied" instead of hiding
	// behind stale data. This is the default because failing closed is the right bias for
	// an allow list — serving users the source of truth no longer contains is a security
	// failure, while denying users is an availability failure that is loud and obvious.
	EmptyRulesetConsistencyFirst EmptyRulesetPolicy = "consistency-first"

	// EmptyRulesetAvailabilityFirst treats the all-rejected load as a failed refresh: the
	// last known good rule set is kept in memory and in the shared cache, the refresh
	// failure counter advances, and health reports degraded.
	//
	// Choose it when the availability loss outweighs the risk of briefly honouring stale
	// entries — typically an unstable data source combined with replicas that restart
	// often, where a transient upstream glitch would otherwise empty every replica and
	// leave restarted ones with nothing to bootstrap from.
	EmptyRulesetAvailabilityFirst EmptyRulesetPolicy = "availability-first"
)

// DefaultEmptyRulesetPolicy preserves the historical behavior.
const DefaultEmptyRulesetPolicy = EmptyRulesetConsistencyFirst

// ParseEmptyRulesetPolicy normalizes a policy string. The bool is false when the input is
// non-empty but unrecognized (callers should warn and fall back to the default).
func ParseEmptyRulesetPolicy(s string) (EmptyRulesetPolicy, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return DefaultEmptyRulesetPolicy, true
	case "consistency-first", "consistency":
		return EmptyRulesetConsistencyFirst, true
	case "availability-first", "availability":
		return EmptyRulesetAvailabilityFirst, true
	default:
		return DefaultEmptyRulesetPolicy, false
	}
}

// Validate reports whether the policy is a recognized value.
func (p EmptyRulesetPolicy) Validate() bool {
	switch p {
	case EmptyRulesetConsistencyFirst, EmptyRulesetAvailabilityFirst:
		return true
	default:
		return false
	}
}

// KeepsLastKnownGood reports whether an all-rejected load must leave the current rule set
// in place instead of replacing it with the empty one.
func (p EmptyRulesetPolicy) KeepsLastKnownGood() bool {
	return p == EmptyRulesetAvailabilityFirst
}

// String returns the canonical string form.
func (p EmptyRulesetPolicy) String() string { return string(p) }
