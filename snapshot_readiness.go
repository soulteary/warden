package main

import (
	"time"

	"github.com/soulteary/warden/internal/cache"
)

const (
	readAllowed       = "allowed"
	readNoSnapshot    = "no_snapshot"
	readViewUnknown   = "view_unknown"
	readSourceUnknown = "source_unknown"
	readInvalidMaxAge = "invalid_max_age"
	readSourceFuture  = "source_future"
	readSourceExpired = "source_expired"
)

type snapshotReadDecision struct {
	validUntil time.Time
	reason     string
}

// evaluateSnapshotRead uses only the pinned publication and observation time.
// Publication, lease renewal and recovery do not establish new source evidence.
// A source time ahead of the observation by no more than the allowance the
// snapshot store applies is a writer whose clock runs ahead, not a refusal:
// the store has already accepted that snapshot for every replica.
func evaluateSnapshotRead(pinned *healthReadState, maxAge time.Duration) snapshotReadDecision {
	if pinned == nil || pinned.state == nil || pinned.state.snapshot == nil {
		return snapshotReadDecision{reason: readNoSnapshot}
	}
	snap := pinned.state.snapshot
	if snap.view == nil || snap.view != pinned.view {
		return snapshotReadDecision{reason: readViewUnknown}
	}
	if maxAge <= 0 {
		return snapshotReadDecision{reason: readInvalidMaxAge}
	}
	if snap.SourceSucceededAt.IsZero() {
		return snapshotReadDecision{reason: readSourceUnknown}
	}
	decision := snapshotReadDecision{validUntil: snap.SourceSucceededAt.Add(maxAge), reason: readAllowed}
	if snap.SourceSucceededAt.After(pinned.observedAt.Add(cache.SNAPSHOT_FUTURE_ALLOWANCE)) {
		decision.reason = readSourceFuture
	} else if !pinned.observedAt.Before(decision.validUntil) {
		decision.reason = readSourceExpired
	}
	return decision
}

func snapshotReadFresh(pinned *healthReadState, maxAge time.Duration) bool {
	return evaluateSnapshotRead(pinned, maxAge).reason == readAllowed
}
