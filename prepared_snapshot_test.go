package main

import (
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/stretchr/testify/require"
)

func TestPreparedSnapshotSeparatesInputAndEffectiveVersions(t *testing.T) {
	app, path, published := policyTestApp(t, "consistency-first")
	writeRuleSet(t, path, `[{"mail":"member@example.com","user_id":"member"},{"mail":"not-an-email","user_id":"rejected"}]`)
	app.backgroundTask(path, "")
	snap := app.snapshots.Load()
	require.Equal(t, 2, snap.Count, "legacy health count still describes loaded input")
	require.Equal(t, 1, snap.EffectiveCount)
	require.NotEqual(t, snap.Version, snap.EffectiveVersion)
	require.Equal(t, cache.HashUserList(app.userCache.Get()), snap.EffectiveVersion)
	require.Equal(t, app.userCache.ReadView(), snap.view)
	require.Equal(t, snap.view.Get(), (*published)[0])
	view := snap.view
	app.backgroundTask(path, "")
	require.Same(t, view, app.snapshots.Load().view, "unchanged input reuses effective indexes")
	require.Equal(t, snap.EffectiveVersion, app.snapshots.Load().EffectiveVersion)

	writeRuleSet(t, path, rejectedRuleSet)
	app.backgroundTask(path, "")
	snap = app.snapshots.Load()
	require.NotZero(t, snap.Count, "do not silently change the legacy empty-health decision")
	require.Zero(t, snap.EffectiveCount)
	require.Equal(t, cache.HashUserList(nil), snap.EffectiveVersion)
}
