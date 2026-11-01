package main

import "time"

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
	if pinned.observedAt.Before(snap.SourceSucceededAt) {
		decision.reason = readSourceFuture
	} else if !pinned.observedAt.Before(decision.validUntil) {
		decision.reason = readSourceExpired
	}
	return decision
}

func snapshotReadFresh(pinned *healthReadState, maxAge time.Duration) bool {
	return evaluateSnapshotRead(pinned, maxAge).reason == readAllowed
}
