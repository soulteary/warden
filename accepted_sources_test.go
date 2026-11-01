package main

import (
	"context"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcceptedSourcesBindAndDetachFromLoaderResult(t *testing.T) {
	app := responseMetadataApp(t)
	file := filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(file, []byte(goodRuleSet), 0o600))
	rules, err := loader.NewRulesLoader(&cmd.Config{SourceSetID: "tenant-a"}, loader.ModeOnlyLocal)
	require.NoError(t, err)
	result := rules.LoadWithResult(context.Background(), file, "", "", "")
	require.NoError(t, result.Err)
	prepared, err := app.prepareUsers(result.Users)
	require.NoError(t, err)
	now := time.Now().UTC()
	app.snapshotNow = func() time.Time { return now }
	candidate := app.sourceSnapshot(&result, prepared.ReadView())
	require.NotNil(t, candidate.acceptedSources)
	app.snapshots.Store(candidate)
	pinned := captureHealthState(app.snapshots, app.userCache)
	evidence := pinned.state.snapshot.acceptedSources
	require.Equal(t, prepared.ReadView().GetHash(), evidence.EffectiveVersion)
	require.Equal(t, result.Version, evidence.InputVersion)
	require.Equal(t, now, evidence.AcceptedAt)
	require.Equal(t, "tenant-a", evidence.SourceSetID)
	originalID := evidence.Manifests[0].Sources[0].ID
	result.Manifests[0].Sources[0].ID = "mutated-result"
	result.Attempts[0].Outcome = "mutated-result"
	candidate.acceptedSources.Manifests[0].Sources[0].ID = "mutated-candidate"
	require.Equal(t, originalID, evidence.Manifests[0].Sources[0].ID)
	require.Equal(t, "success", evidence.Attempts[0].Outcome)
	app.snapshots.RecordRefreshFailure("timeout")
	require.Same(t, evidence, app.snapshots.Load().acceptedSources)
	next := app.sourceSnapshot(&result, prepared.ReadView())
	app.snapshots.Store(next)
	require.Greater(t, app.snapshots.Load().acceptedSources.Generation, evidence.Generation)
	require.Equal(t, originalID, pinned.state.snapshot.acceptedSources.Manifests[0].Sources[0].ID)
}
func TestRejectedRefreshKeepsAcceptedSourceEvidence(t *testing.T) {
	app, file, _ := policyTestApp(t, "availability-first")
	app.backgroundTask(file, "")
	known := app.snapshots.Load().acceptedSources
	require.NotNil(t, known)
	for _, content := range []string{rejectedRuleSet, "invalid JSON"} {
		writeRuleSet(t, file, content)
		app.backgroundTask(file, "")
		require.Same(t, known, app.snapshots.Load().acceptedSources)
	}
	failures, _ := app.snapshots.RefreshFailure()
	require.EqualValues(t, 2, failures)
	writeRuleSet(t, file, emptyRuleSet)
	app.backgroundTask(file, "")
	require.Empty(t, app.snapshots.Load().view.Get())
	require.Greater(t, app.snapshots.Load().acceptedSources.Generation, known.Generation)
}
func TestV2RecoveryCannotInventAcceptedSources(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	publishMigrationSource(t, app, time.Now().Add(-time.Second), []define.AllowListUser{{Mail: "member@example.com"}})
	app.snapshots = newSnapshotStore()
	require.True(t, app.bootstrapFromRedis())
	require.Nil(t, app.snapshots.Load().acceptedSources)
}
