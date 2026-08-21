package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/prommetrics"
	"github.com/soulteary/warden/internal/remote"
)

// Snapshot is an immutable point-in-time view of the loaded rule set plus
// provenance metadata. Snapshots are swapped atomically; a failed refresh keeps
// the previous last-known-good snapshot instead of overwriting it with empty or
// partial data.
//
//nolint:govet // fieldalignment: readability over a few bytes for a small, rarely-allocated struct
type Snapshot struct {
	// Users is the resolved rule set backing the current cache. It must be treated
	// as read-only by consumers.
	Users []define.AllowListUser
	// Count is len(Users), cached for cheap health reporting.
	Count int
	// Source records where the snapshot's data came from.
	Source loader.Source
	// Version is a short content hash used for change detection and reporting.
	Version string
	// LoadedAt is when this snapshot was produced.
	LoadedAt time.Time
	// Degraded is true when the snapshot was served from local/last-known-good
	// because the preferred remote source failed.
	Degraded bool
	// DegradedReason is a short, non-sensitive reason code.
	DegradedReason string
}

// snapshotStore holds the current immutable snapshot plus refresh health counters.
// Reads are lock-free via atomic.Pointer; writes replace the whole snapshot.
type snapshotStore struct {
	current atomic.Pointer[Snapshot]
	// refreshFailures counts consecutive background refresh failures since the last
	// successful refresh. It is advisory and only mutated from the single background
	// task goroutine plus atomic reads for reporting.
	refreshFailures atomic.Int64
	// lastRefreshErr stores the most recent refresh error string (non-sensitive code).
	lastRefreshReason atomic.Pointer[string]
}

// newSnapshotStore returns an empty store with a zero-value initial snapshot.
func newSnapshotStore() *snapshotStore {
	s := &snapshotStore{}
	s.current.Store(&Snapshot{Source: loader.SourceNone, LoadedAt: time.Now()})
	return s
}

// Load returns the current immutable snapshot (never nil).
func (s *snapshotStore) Load() *Snapshot {
	return s.current.Load()
}

// Store atomically replaces the current snapshot and resets failure counters.
func (s *snapshotStore) Store(snap *Snapshot) {
	s.current.Store(snap)
	s.refreshFailures.Store(0)
	empty := ""
	s.lastRefreshReason.Store(&empty)
}

// RecordRefreshFailure increments the failure counter and records a reason code,
// keeping the existing last-known-good snapshot untouched.
func (s *snapshotStore) RecordRefreshFailure(reason string) int64 {
	r := reason
	s.lastRefreshReason.Store(&r)
	return s.refreshFailures.Add(1)
}

// AgeSeconds returns the age of the current snapshot in seconds.
func (s *snapshotStore) AgeSeconds() float64 {
	snap := s.current.Load()
	if snap == nil || snap.LoadedAt.IsZero() {
		return 0
	}
	return time.Since(snap.LoadedAt).Seconds()
}

// snapshotFromResult builds an immutable Snapshot from a loader.LoadResult.
func snapshotFromResult(res loader.LoadResult) *Snapshot {
	return &Snapshot{
		Users:          res.Users,
		Count:          len(res.Users),
		Source:         res.Source,
		Version:        res.Version,
		LoadedAt:       res.LoadedAt,
		Degraded:       res.Degraded,
		DegradedReason: res.DegradedReason,
	}
}

// classifyRefreshReason maps a load error to a stable, low-cardinality, non-sensitive
// reason code suitable for logs and metric labels. It never includes error contents,
// URLs, keys, or user data.
func classifyRefreshReason(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, remote.ErrEncryptionRequired):
		return "encryption_required"
	case errors.Is(err, remote.ErrIntegrityCheckFailed):
		return "integrity_failed"
	case errors.Is(err, remote.ErrUnsupportedEnvelopeVersion), errors.Is(err, remote.ErrUnsupportedAlgorithm):
		return "unsupported_format"
	case errors.Is(err, remote.ErrEnvelopeMalformed):
		return "malformed"
	case errors.Is(err, remote.ErrPlaintextTooLarge):
		return "too_large"
	}
	// Coarse classification from the message without leaking specifics.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "timeout"
	case strings.Contains(msg, "status "):
		return "http_status"
	case strings.Contains(msg, "json") || strings.Contains(msg, "parse") || strings.Contains(msg, "decode"):
		return "parse"
	case strings.Contains(msg, "no sources") || strings.Contains(msg, "no local"):
		return "no_sources"
	default:
		return "load_failed"
	}
}

// updateSnapshotMetrics refreshes the snapshot age and degraded gauges from the store.
func (app *App) updateSnapshotMetrics() {
	if app.snapshots == nil {
		return
	}
	prommetrics.SnapshotAgeSeconds.Set(app.snapshots.AgeSeconds())
	snap := app.snapshots.Load()
	if snap != nil && snap.Degraded {
		prommetrics.SnapshotDegraded.Set(1)
	} else {
		prommetrics.SnapshotDegraded.Set(0)
	}
}
