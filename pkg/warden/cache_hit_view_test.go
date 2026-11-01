package warden

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCacheCapturedViewSurvivesClearAndSet(t *testing.T) {
	clock := newSourceTestClock()
	now := clock.Now()
	cache := NewCache(time.Minute)
	cache.now = clock.Now
	require.True(t, cache.setUntil([]AllowListUser{{UserID: "old", Scope: []string{"read"}}}, now.Add(time.Minute)))
	clock.set(now.Add(time.Minute - time.Nanosecond))
	captured := cache.captureUsers()
	require.NotNil(t, captured)
	// The hit was valid when captured. Later expiry and replacement cannot mutate it.
	clock.set(now.Add(time.Minute))
	cache.Clear()
	cache.Set([]AllowListUser{{UserID: "new", Scope: []string{"write"}}})
	copied := cloneCachedUsers(captured)
	require.Equal(t, []AllowListUser{{UserID: "old", Scope: []string{"read"}}}, copied)
	copied[0].Scope[0] = "changed"
	require.Equal(t, "read", captured[0].Scope[0])
	require.Equal(t, "new", cache.Get()[0].UserID)
	require.Equal(t, "write", cache.Get()[0].Scope[0])
}

func TestCacheMissRechecksConcurrentPublication(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "member", true: "empty"}[empty], func(t *testing.T) {
			cache := NewCache(time.Minute)
			require.Nil(t, cache.captureUsers())
			rows := []AllowListUser{{UserID: "new", Scope: []string{"read"}}}
			if empty {
				rows = nil
			}
			cache.Set(rows)
			captured, request := cache.beginMiss()
			require.NotNil(t, captured)
			require.Equal(t, cache.Get(), cloneCachedUsers(captured))
			require.Equal(t, cacheRequest{}, request)
			require.Zero(t, cache.nextRequest)
		})
	}
}

func TestCacheMissRechecksExpiryAndEpoch(t *testing.T) {
	clock := newSourceTestClock()
	now := clock.Now()
	cache := NewCache(time.Minute)
	cache.now = clock.Now
	cache.Set([]AllowListUser{{UserID: "expired"}})
	clock.set(now.Add(time.Minute + time.Nanosecond))
	require.Nil(t, cache.captureUsers())
	cache.Clear()
	rows, request := cache.beginMiss()
	require.Nil(t, rows)
	require.Equal(t, cacheRequest{epoch: 1, sequence: 1}, request)
	require.ErrorIs(t, cache.commitRead(nil, time.Time{}, cacheRequest{sequence: 1}), ErrCacheInvalidated)
	require.NoError(t, cache.commitRead([]AllowListUser{{UserID: "current"}}, time.Time{}, request))
	require.Equal(t, "current", cache.Get()[0].UserID)
}

func TestCacheBeginReadCapturesHitsConcurrently(t *testing.T) {
	cache := NewCache(time.Hour)
	cache.Set([]AllowListUser{{UserID: "member", Scope: []string{"read"}}})
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan []AllowListUser, 2)
	// Both readers must reach the clock while holding shared locks. An exclusive
	// hit lock would block the second reader until the first one was released.
	cache.now = func() time.Time { entered <- struct{}{}; <-release; return time.Now() }
	go func() { rows, _ := cache.beginRead(); done <- rows }()
	go func() { rows, _ := cache.beginRead(); done <- rows }()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-timer.C:
			close(release)
			t.Fatal("cache hits did not acquire shared locks")
		}
	}
	close(release)
	for i := 0; i < 2; i++ {
		require.Equal(t, "member", (<-done)[0].UserID)
	}
}
