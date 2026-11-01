package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	health "github.com/soulteary/health-kit/v4"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/require"
)

func preparedTestSnapshot(t *testing.T, users []define.AllowListUser) (*cache.PreparedRules, *Snapshot) {
	t.Helper()
	prepared, err := cache.PrepareRules(users, identity.Options{})
	require.NoError(t, err)
	res := &loader.LoadResult{Users: users, Version: cache.HashUserList(users), LoadedAt: time.Now(), Source: loader.SourceRemote}
	return prepared, snapshotFromResult(res, prepared.ReadView())
}

func TestSnapshotPublicationIsReadCommitPoint(t *testing.T) {
	old, oldSnap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "old@example.com", UserID: "old"}})
	newRules, newSnap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "new@example.com", UserID: "new"}})
	users := cache.NewSafeUserCache()
	users.ApplyPrepared(old)
	store := newSnapshotStore()
	store.Store(oldSnap)
	source := snapshotUsers{snapshots: store, fallback: users}
	users.ApplyPrepared(newRules)
	require.Equal(t, "old", source.ReadView().Get()[0].UserID, "write-side staging is not publication")
	store.Store(newSnap)
	require.Equal(t, "new", source.ReadView().Get()[0].UserID)
	require.Equal(t, newSnap.EffectiveVersion, source.ReadView().GetHash())
	require.Equal(t, "old", old.ReadView().Get()[0].UserID)
}

func TestSnapshotStateIsImmutableAndPreservesFailureSemantics(t *testing.T) {
	_, snap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "a@example.com", UserID: "a", Scope: []string{"read"}}})
	store := newSnapshotStore()
	store.Store(snap)
	before := store.capture()
	snap.Users[0].Scope[0] = "changed"
	require.Equal(t, []string{"read"}, before.snapshot.Users[0].Scope)
	require.EqualValues(t, 1, store.RecordRefreshFailure("timeout"))
	require.Zero(t, before.failures)
	require.Equal(t, "timeout", store.capture().reason)
	require.Same(t, before.snapshot.view, store.capture().snapshot.view)
	store.StorePreservingFailures(snap)
	failures, reason := store.RefreshFailure()
	require.EqualValues(t, 1, failures)
	require.Equal(t, "timeout", reason)
	store.Store(snap)
	failures, reason = store.RefreshFailure()
	require.Zero(t, failures)
	require.Empty(t, reason)
}

func TestConcurrentSnapshotReadsKeepEffectiveMetadata(t *testing.T) {
	_, first := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "a@example.com", UserID: "a"}})
	_, second := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "b@example.com", UserID: "b"}, {Mail: "c@example.com", UserID: "c"}, {Mail: "invalid", UserID: "rejected"}})
	store := newSnapshotStore()
	store.Store(first)
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			store.Store(second)
			store.RecordRefreshFailure("timeout")
			store.Store(first)
		}
	}()
	for i := 0; i < 100; i++ {
		state := store.capture()
		snap := state.snapshot
		require.Equal(t, snap.EffectiveCount, snap.view.Len())
		require.Equal(t, snap.EffectiveVersion, cache.HashUserList(snap.view.Get()))
		require.Equal(t, snap.Version, cache.HashUserList(snap.Users))
		if state.failures > 0 {
			require.Equal(t, "timeout", state.reason)
		}
	}
}

func TestHealthRequestPinsDataAndProvenance(t *testing.T) {
	old, oldSnap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "old@example.com", UserID: "old"}})
	newRules, newSnap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "invalid", UserID: "rejected"}})
	users := cache.NewSafeUserCache()
	users.ApplyPrepared(old)
	store := newSnapshotStore()
	store.Store(oldSnap)
	aggregator := setupHealthChecker(nil, users, store, time.Minute, "ONLY_REMOTE", "development", false, false, "")
	observed := health.StatusUnhealthy
	handler := pinHealthRead(store, users, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users.ApplyPrepared(newRules)
		store.Store(newSnap)
		store.RecordRefreshFailure("timeout")
		result := aggregator.Check(r.Context())
		observed = result.Status
		require.Equal(t, oldSnap.Version, result.Checks["snapshot"].Metadata["version"])
		require.Equal(t, health.StatusHealthy, result.Checks["data"].Status)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Errorf("encode health: %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))
	require.Equal(t, health.StatusHealthy, observed)
	require.Equal(t, health.StatusUnhealthy, aggregator.Check(httptest.NewRequest(http.MethodGet, "/health", http.NoBody).Context()).Status, "next check observes the new empty effective generation with its legacy non-empty input count")
}
