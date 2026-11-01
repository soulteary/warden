package main

import (
	"github.com/soulteary/warden/internal/cache"
	"net/http"
	"time"
)

const snapshotMetadataHeader = "Warden-Snapshot-Metadata"
const snapshotVersionHeader = "Warden-Snapshot-Version"
const sourceSucceededHeader = "Warden-Source-Succeeded-At"
const snapshotValidUntilHeader = "Warden-Snapshot-Valid-Until"

type fixedSnapshotView struct{ view *cache.ReadView }

func (s fixedSnapshotView) ReadView() *cache.ReadView { return s.view }

type snapshotDataFactory func(cache.ViewSource) func(http.ResponseWriter, *http.Request)

// withSnapshotMetadata runs inside authentication. Only negotiated responses pin
// metadata and served indexes together; ordinary requests retain their existing query path.
func (app *App) withSnapshotMetadata(build snapshotDataFactory) func(http.ResponseWriter, *http.Request) {
	ordinary := build(snapshotUsers{snapshots: app.snapshots, fallback: app.userCache})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", snapshotMetadataHeader)
		if r.Header.Get(snapshotMetadataHeader) != "v1" {
			ordinary(w, r)
			return
		}
		pinned := captureHealthState(app.snapshots, app.userCache)
		w.Header().Set(snapshotMetadataHeader, "v1")
		w.Header().Set("Cache-Control", "private")
		if pinned.view != nil {
			w.Header().Set(snapshotVersionHeader, pinned.view.GetHash())
		}
		snap := pinned.state.snapshot
		// A fallback view is usable for compatibility, but cannot establish source evidence.
		if snap != nil && snap.view != nil && snap.view == pinned.view && !snap.SourceSucceededAt.IsZero() && app.snapshotMaxAge > 0 {
			w.Header().Set(sourceSucceededHeader, snap.SourceSucceededAt.UTC().Format(time.RFC3339Nano))
			w.Header().Set(snapshotValidUntilHeader, snap.SourceSucceededAt.Add(app.snapshotMaxAge).UTC().Format(time.RFC3339Nano))
		}
		build(fixedSnapshotView{view: pinned.view})(w, r)
	}
}
