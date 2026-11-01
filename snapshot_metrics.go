package main

import "github.com/soulteary/warden/internal/prommetrics"

// snapshotMetricSample captures one immutable root for the entire collection.
func (app *App) snapshotMetricSample() prommetrics.SnapshotSample {
	var sample prommetrics.SnapshotSample
	if app.snapshots == nil {
		return sample
	}
	state := app.snapshots.capture()
	if state == nil {
		return sample
	}
	sample.Degraded = state.failures > 0
	if snap := state.snapshot; snap != nil {
		sample.LoadedAt = snap.LoadedAt
		sample.SourceSucceededAt = snap.SourceSucceededAt
		sample.PublishedAt = snap.PublishedAt
		sample.RestoredAt = snap.RestoredAt
		sample.Degraded = sample.Degraded || snap.Degraded
	}
	return sample
}
