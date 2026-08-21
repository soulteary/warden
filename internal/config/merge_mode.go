// Package config: merge_mode.go defines the data MergeMode type.
//
// MergeMode controls how local and remote rule sources are combined. It is distinct
// from Environment (which drives security policy). The legacy overloaded "MODE" env
// maps onto MergeMode only; it no longer influences production decisions.
package config

import "strings"

// MergeMode selects how local files and the remote endpoint are combined.
type MergeMode string

const (
	// MergeDefault is the historical default merge behavior (remote-first, tolerant),
	// preserved so existing deployments that never set MODE keep working unchanged.
	MergeDefault MergeMode = "DEFAULT"
	// MergeOnlyLocal loads only local files (no remote).
	MergeOnlyLocal MergeMode = "ONLY_LOCAL"
	// MergeOnlyRemote loads only the remote endpoint (strict: remote failure is fatal).
	MergeOnlyRemote MergeMode = "ONLY_REMOTE"
	// MergeRemoteFirst prefers remote, then local (strict on remote failure).
	MergeRemoteFirst MergeMode = "REMOTE_FIRST"
	// MergeRemoteFirstAllowRemoteFail prefers remote but tolerates remote failure.
	MergeRemoteFirstAllowRemoteFail MergeMode = "REMOTE_FIRST_ALLOW_REMOTE_FAILED"
	// MergeLocalFirst prefers local, then remote (tolerant of remote failure).
	MergeLocalFirst MergeMode = "LOCAL_FIRST"
	// MergeLocalFirstAllowRemoteFail prefers local and tolerates remote failure.
	MergeLocalFirstAllowRemoteFail MergeMode = "LOCAL_FIRST_ALLOW_REMOTE_FAILED"
)

// DefaultMergeMode preserves the historical default merge behavior.
func DefaultMergeMode() MergeMode { return MergeDefault }

// ParseMergeMode normalizes a merge-mode string (upper-cased, trimmed). The bool is
// false when the input is non-empty but unrecognized (callers should warn/fall back).
func ParseMergeMode(s string) (MergeMode, bool) {
	v := strings.ToUpper(strings.TrimSpace(s))
	if v == "" {
		return DefaultMergeMode(), true
	}
	m := MergeMode(v)
	if m.Validate() {
		return m, true
	}
	return DefaultMergeMode(), false
}

// Validate reports whether the MergeMode is a recognized value.
func (m MergeMode) Validate() bool {
	switch m {
	case MergeDefault, MergeOnlyLocal, MergeOnlyRemote, MergeRemoteFirst,
		MergeRemoteFirstAllowRemoteFail, MergeLocalFirst, MergeLocalFirstAllowRemoteFail:
		return true
	default:
		return false
	}
}

// String returns the canonical (upper-case) string form.
func (m MergeMode) String() string { return string(m) }

// IsOnlyLocal reports whether remote loading is disabled.
func (m MergeMode) IsOnlyLocal() bool { return m == MergeOnlyLocal }
