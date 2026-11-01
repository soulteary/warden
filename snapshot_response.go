package main

import (
	"github.com/soulteary/warden/internal/cache"
	"net/http"
	"time"
)

const snapshotMetadataHeader = "Warden-Snapshot-Metadata"
const sourceErrorHeader = "Warden-Source-Error"
const snapshotVersionHeader = "Warden-Snapshot-Version"
const sourceSucceededHeader = "Warden-Source-Succeeded-At"
const snapshotValidUntilHeader = "Warden-Snapshot-Valid-Until"

type fixedSnapshotView struct{ view *cache.ReadView }

func (s fixedSnapshotView) ReadView() *cache.ReadView { return s.view }

type snapshotDataFactory func(cache.ViewSource) func(http.ResponseWriter, *http.Request)

// withSnapshotMetadata runs inside authentication. Enforced reads and negotiated
// responses check provenance and serve indexes from the same publication.
func (app *App) withSnapshotMetadata(build snapshotDataFactory) func(http.ResponseWriter, *http.Request) {
	ordinary := build(snapshotUsers{snapshots: app.snapshots, fallback: app.userCache})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", snapshotMetadataHeader)
		negotiated := r.Header.Get(snapshotMetadataHeader) == "v1"
		if !negotiated && !app.snapshotReadEnforce {
			ordinary(w, r)
			return
		}
		pinned := captureHealthState(app.snapshots, app.userCache)
		if app.snapshotReadEnforce {
			w.Header().Set("Cache-Control", "no-store")
			decision := evaluateSnapshotRead(pinned, app.snapshotMaxAge)
			if decision.reason != readAllowed {
				if negotiated {
					w.Header().Set(snapshotMetadataHeader, "v1")
					if reason := sourceErrorReason(decision.reason); reason != "" {
						w.Header().Set(sourceErrorHeader, reason)
					}
				}
				http.Error(w, "snapshot source freshness unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		if negotiated {
			w.Header().Set(snapshotMetadataHeader, "v1")
			if !app.snapshotReadEnforce {
				w.Header().Set("Cache-Control", "private")
			}
			if pinned.view != nil {
				w.Header().Set(snapshotVersionHeader, pinned.view.GetHash())
			}
			snap := pinned.state.snapshot
			// A fallback view is usable for compatibility, but cannot establish source evidence.
			if snap != nil && snap.view != nil && snap.view == pinned.view && !snap.SourceSucceededAt.IsZero() && app.snapshotMaxAge > 0 {
				w.Header().Set(sourceSucceededHeader, snap.SourceSucceededAt.UTC().Format(time.RFC3339Nano))
				w.Header().Set(snapshotValidUntilHeader, snap.SourceSucceededAt.Add(app.snapshotMaxAge).UTC().Format(time.RFC3339Nano))
			}
		}
		build(fixedSnapshotView{view: pinned.view})(w, r)
	}
}

// Only negotiated source refusals carry this bounded protocol marker. A bad
// max-age configuration is not classified as missing source-age evidence.
func sourceErrorReason(reason string) string {
	switch reason {
	case readSourceExpired:
		return "expired"
	case readSourceFuture:
		return "future"
	case readNoSnapshot, readViewUnknown, readSourceUnknown:
		return "unknown"
	default:
		return ""
	}
}
