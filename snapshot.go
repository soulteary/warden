package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/soulteary/warden/internal/cache"
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
	// view owns the effective users and indexes for EffectiveVersion/EffectiveCount.
	view            *cache.ReadView
	acceptedSources *acceptedSourceEvidence
	// EffectiveVersion and EffectiveCount describe the served set after filtering.
	EffectiveVersion string
	EffectiveCount   int
	// Users is the loaded input retained for provenance. Effective users live in view.
	// It must be treated as read-only by consumers.
	Users []define.AllowListUser
	// Count is the loaded input count. Keep this legacy health interpretation.
	Count int
	// Source records where the snapshot's data came from.
	Source loader.Source
	// Version is the loaded input hash used for change detection and legacy health reporting.
	Version string
	// LoadedAt retains legacy timing by default; V2 uses source validation time.
	LoadedAt time.Time
	// SourceSucceededAt is completion of an accepted resolved source validation.
	SourceSucceededAt time.Time
	// PublishedAt is successful V2 publication; TTL renewal leaves it unchanged.
	PublishedAt time.Time
	// RestoredAt records local recovery and is never persisted as source evidence.
	RestoredAt    time.Time
	Origin        loader.Source
	RedisRevision string
	versioned     bool
	// Degraded is true when the snapshot was served from local/last-known-good
	// because the preferred remote source failed.
	Degraded bool
	// DegradedReason is a short, non-sensitive reason code.
	DegradedReason string
}

// snapshotStore holds the current immutable snapshot plus refresh health counters.
// Reads are lock-free via atomic.Pointer; writes replace the whole snapshot.
//
//nolint:govet // fieldalignment: field order chosen for readability
type snapshotStore struct {
	current atomic.Pointer[snapshotState]
}

// snapshotState publishes rules, provenance and refresh diagnostics together.
// Every update replaces this object; readers can pin it without holding a lock.
type snapshotState struct {
	snapshot *Snapshot
	reason   string
	failures int64
}

func (s *snapshotStore) capture() *snapshotState { return s.current.Load() }

func freezeSnapshot(snap *Snapshot) *Snapshot {
	if snap == nil {
		return nil
	}
	frozen := *snap
	frozen.acceptedSources = cloneAcceptedSources(snap.acceptedSources)
	if snap.Users != nil {
		frozen.Users = append([]define.AllowListUser{}, snap.Users...)
		for i := range frozen.Users {
			if snap.Users[i].Scope != nil {
				frozen.Users[i].Scope = append([]string{}, snap.Users[i].Scope...)
			}
		}
	}
	return &frozen
}

// newSnapshotStore returns an empty store with a zero-value initial snapshot.
func newSnapshotStore() *snapshotStore {
	s := &snapshotStore{}
	s.Store(&Snapshot{Source: loader.SourceNone})
	return s
}

// Load returns the current immutable snapshot (never nil).
func (s *snapshotStore) Load() *Snapshot {
	state := s.capture()
	if state == nil {
		return nil
	}
	return state.snapshot
}

// HasKnownGood reports whether the store contains a successfully loaded snapshot. Count
// is deliberately not part of the check: an empty allow list can be a valid mass
// revocation, while SourceNone/zero LoadedAt represent a process that has loaded nothing.
func (s *snapshotStore) HasKnownGood() bool {
	if s == nil {
		return false
	}
	return isKnownGoodSnapshot(s.Load())
}

func isKnownGoodSnapshot(snap *Snapshot) bool {
	return snap != nil && snap.Source != loader.SourceNone && !snap.LoadedAt.IsZero()
}

// Store atomically replaces the current snapshot and resets failure counters.
func (s *snapshotStore) Store(snap *Snapshot) {
	s.current.Store(&snapshotState{snapshot: freezeSnapshot(snap)})
}

// StorePreservingFailures adopts a recovered baseline without counting a source
// refresh as successful. Diagnostics remain attached to the same published state.
func (s *snapshotStore) StorePreservingFailures(snap *Snapshot) {
	frozen := freezeSnapshot(snap)
	for {
		previous := s.capture()
		next := &snapshotState{snapshot: frozen}
		if previous != nil {
			next.failures = previous.failures
			next.reason = previous.reason
		}
		if s.current.CompareAndSwap(previous, next) {
			return
		}
	}
}

// RecordRefreshFailure retains the data generation while replacing its diagnostics.
func (s *snapshotStore) RecordRefreshFailure(reason string) int64 {
	for {
		previous := s.capture()
		next := &snapshotState{reason: reason, failures: 1}
		if previous != nil {
			next.snapshot = previous.snapshot
			next.failures += previous.failures
		}
		if s.current.CompareAndSwap(previous, next) {
			return next.failures
		}
	}
}

// RefreshFailure reads a consistent count/reason pair.
func (s *snapshotStore) RefreshFailure() (failures int64, reason string) {
	state := s.capture()
	if state == nil {
		return 0, ""
	}
	return state.failures, state.reason
}

// AgeSeconds returns the age of the current snapshot in seconds.
func (s *snapshotStore) AgeSeconds() float64 {
	snap := s.Load()
	if snap == nil || snap.LoadedAt.IsZero() {
		return 0
	}
	return time.Since(snap.LoadedAt).Seconds()
}

// snapshotFromResult builds an immutable Snapshot from a loader.LoadResult.
func snapshotFromResult(res *loader.LoadResult, view *cache.ReadView) *Snapshot {
	return &Snapshot{
		view:             view,
		EffectiveVersion: view.GetHash(),
		EffectiveCount:   view.Len(),
		Users:            res.Users,
		Count:            len(res.Users),
		Source:           res.Source,
		Version:          res.Version,
		LoadedAt:         res.LoadedAt,
		Degraded:         res.Degraded,
		DegradedReason:   res.DegradedReason,
	}
}

// snapshotFromRedis builds provenance for a rule set restored from the shared cache.
// Redis currently stores the effective users but not the original source timestamp, so
// LoadedAt records when this replica successfully verified and adopted the cached set.
// Marking it degraded keeps the fallback visible while giving strict modes a bounded
// freshness window instead of treating a usable bootstrap as SourceNone immediately.
func snapshotFromRedis(users []define.AllowListUser, view *cache.ReadView) *Snapshot {
	return &Snapshot{
		view:             view,
		EffectiveVersion: view.GetHash(),
		EffectiveCount:   view.Len(),
		Users:            users,
		Count:            len(users),
		Source:           loader.SourceRedis,
		Version:          cache.HashUserList(users),
		LoadedAt:         time.Now(),
		RestoredAt:       time.Now(),
		Degraded:         true,
		DegradedReason:   "redis_bootstrap",
	}
}

// hasKnownGoodSnapshot distinguishes a legitimately empty effective rule set from a
// process that has never obtained any valid data. Cache length alone cannot make that
// distinction, yet it determines whether publishing an empty slice to Redis is a valid
// renewal or could erase a shared last-known-good set after a bootstrap failure.
func (app *App) hasKnownGoodSnapshot() bool {
	return app.snapshots.HasKnownGood()
}

// Refresh failure reasons that are not derived from a load error. They share the same
// stable, low-cardinality label space as classifyRefreshReason, which is what keeps the
// reason label on refresh_failures_total bounded.
const (
	// reasonIdentityConflict: identity validation rejected the set (duplicate identifier
	// or a record missing a required user_id).
	reasonIdentityConflict = "identity_conflict"
	// reasonAllRecordsRejected: the load succeeded and returned records, but every one of
	// them failed per-record validation. Only recorded as a failure under the
	// availability-first empty-ruleset policy; consistency-first applies the empty set and
	// counts the refresh as successful.
	reasonAllRecordsRejected = "all_records_rejected"
)

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
	failures, _ := app.snapshots.RefreshFailure()
	if failures > 0 || (snap != nil && snap.Degraded) {
		prommetrics.SnapshotDegraded.Set(1)
	} else {
		prommetrics.SnapshotDegraded.Set(0)
	}
}
