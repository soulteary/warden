package main

import (
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFrozenSnapshotOwnsOverlappingScopesAndPreservesTimes(t *testing.T) {
	backing := []string{"read", "write"}
	empty := []string{"spare"}
	rows := []define.AllowListUser{{UserID: "a", Mail: "a@example.com", Scope: backing[:2]}, {UserID: "b", Mail: "b@example.com", Scope: backing[1:]}, {UserID: "c", Mail: "c@example.com", Scope: empty[:0]}, {UserID: "d", Mail: "d@example.com"}}
	prepared, err := cache.PrepareRules(rows, identity.Options{})
	require.NoError(t, err)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	original := &Snapshot{Users: rows, view: prepared.ReadView(), Version: prepared.InputVersion(), EffectiveVersion: prepared.ReadView().GetHash(), SourceSucceededAt: at, PublishedAt: at.Add(time.Minute), RestoredAt: at.Add(time.Hour)}
	frozen := freezeSnapshot(original)
	backing[0] = "caller"
	backing[1] = "caller"
	empty[0] = "caller"
	require.Equal(t, []string{"read", "write"}, frozen.Users[0].Scope)
	require.Equal(t, []string{"write"}, frozen.Users[1].Scope)
	require.NotNil(t, frozen.Users[2].Scope)
	require.Empty(t, frozen.Users[2].Scope)
	require.Nil(t, frozen.Users[3].Scope)
	frozen.Users[0].Scope[1] = "first frozen record"
	require.Equal(t, []string{"write"}, frozen.Users[1].Scope)
	require.Equal(t, []string{"read", "write"}, frozen.view.Get()[0].Scope)
	require.Same(t, original.view, frozen.view)
	require.Equal(t, original.Version, frozen.Version)
	require.Equal(t, original.EffectiveVersion, frozen.EffectiveVersion)
	require.Equal(t, at, frozen.SourceSucceededAt)
	require.Equal(t, original.PublishedAt, frozen.PublishedAt)
	require.Equal(t, original.RestoredAt, frozen.RestoredAt)
}
