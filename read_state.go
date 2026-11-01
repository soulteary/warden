package main

import (
	"context"
	"net/http"
	"time"

	"github.com/soulteary/warden/internal/cache"
)

// snapshotUsers reads from the published snapshot, not the write-side cache.
// The fallback supports callers that construct legacy snapshots without a view.
type snapshotUsers struct {
	snapshots *snapshotStore
	fallback  *cache.SafeUserCache
}

func (s snapshotUsers) ReadView() *cache.ReadView {
	if s.snapshots != nil {
		if snap := s.snapshots.Load(); snap != nil && snap.view != nil {
			return snap.view
		}
	}
	return s.fallback.ReadView()
}

type healthReadKey struct{}

type healthReadState struct {
	state      *snapshotState
	view       *cache.ReadView
	observedAt time.Time
}

func captureHealthState(snapshots *snapshotStore, users *cache.SafeUserCache) *healthReadState {
	state := &snapshotState{}
	if snapshots != nil {
		if current := snapshots.capture(); current != nil {
			state = current
		}
	}
	var view *cache.ReadView
	if state.snapshot != nil {
		view = state.snapshot.view
	}
	if view == nil && users != nil {
		view = users.ReadView()
	}
	return &healthReadState{state: state, view: view, observedAt: time.Now()}
}

func readHealthState(ctx context.Context, snapshots *snapshotStore, users *cache.SafeUserCache) *healthReadState {
	if state, ok := ctx.Value(healthReadKey{}).(*healthReadState); ok {
		return state
	}
	return captureHealthState(snapshots, users)
}

// pinHealthRead binds all concurrently evaluated data/provenance checks in one
// HTTP request to one publication. Redis probing remains a live dependency check.
func pinHealthRead(snapshots *snapshotStore, users *cache.SafeUserCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), healthReadKey{}, captureHealthState(snapshots, users))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
