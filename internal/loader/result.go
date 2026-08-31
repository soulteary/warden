package loader

import (
	"strings"
	"time"

	"github.com/soulteary/warden/internal/define"
)

// Source identifies where a successfully loaded rule set originated.
type Source string

const (
	// SourceRemote indicates the rules came from the remote configuration endpoint.
	SourceRemote Source = "remote"
	// SourceLocal indicates the rules came from local file(s)/directory.
	SourceLocal Source = "local"
	// SourceMerged indicates the rules came from a merge of remote and local.
	SourceMerged Source = "merged"
	// SourceNone indicates no rules were loaded.
	SourceNone Source = "none"
)

// Merge/environment modes. These mirror the historical MODE values so behavior
// is preserved; the values are matched case-insensitively after trimming.
const (
	ModeOnlyLocal                  = "ONLY_LOCAL"
	ModeOnlyRemote                 = "ONLY_REMOTE"
	ModeRemoteFirst                = "REMOTE_FIRST"
	ModeRemoteFirstAllowRemoteFail = "REMOTE_FIRST_ALLOW_REMOTE_FAILED"
	ModeLocalFirst                 = "LOCAL_FIRST"
	ModeLocalFirstAllowRemoteFail  = "LOCAL_FIRST_ALLOW_REMOTE_FAILED"
)

// LoadResult is the outcome of a load attempt. It separates "what was read" from
// "how the mode decided to use it" so callers (and the background refresher) can
// reason about degraded operation without inspecting the concrete error.
//
//nolint:govet // fieldalignment: readability over a few bytes for a rarely-allocated struct
type LoadResult struct {
	// Users is the resolved rule set. It is only meaningful when Err is nil.
	Users []define.AllowListUser
	// Source records where the resolved rule set came from.
	Source Source
	// Version is a short content hash of the resolved rule set (empty when Err != nil).
	Version string
	// LoadedAt is when the result was produced.
	LoadedAt time.Time
	// Degraded is true when the mode allowed a remote failure and the result was
	// served from local/last-known-good instead of the preferred remote source.
	Degraded bool
	// DegradedReason is a short, non-sensitive reason code (e.g. "remote_failed").
	DegradedReason string
	// Err is the combined root cause when loading failed. It never carries secrets.
	Err error
}

// normalizeMode upper-cases and trims a mode string.
func normalizeMode(mode string) string {
	return strings.ToUpper(strings.TrimSpace(mode))
}

// allowsRemoteFailure reports whether the mode tolerates a remote failure by
// falling back to local/last-known-good and marking the result degraded.
func allowsRemoteFailure(mode string) bool {
	switch normalizeMode(mode) {
	case define.DEFAULT_MODE, ModeLocalFirst, ModeLocalFirstAllowRemoteFail, ModeRemoteFirstAllowRemoteFail, ModeOnlyLocal:
		return true
	default:
		// ONLY_REMOTE and REMOTE_FIRST are strict: a remote failure is fatal.
		return false
	}
}
