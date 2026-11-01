package warden

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListInvalidationFencesInFlightResponse(t *testing.T) {
	for _, sourceBound := range []bool{false, true} {
		for _, empty := range []bool{false, true} {
			t.Run(map[bool]string{true: "source", false: "default"}[sourceBound]+map[bool]string{true: "/empty", false: "/member"}[empty], func(t *testing.T) {
				started, release := make(chan struct{}), make(chan struct{})
				clock := newSourceTestClock()
				now := clock.Now()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(started)
					<-release
					if sourceBound {
						setSourceTestHeaders(w.Header(), now, now.Add(time.Minute))
					}
					body := `[{"user_id":"old","scope":["read"]}]`
					if empty {
						body = `[]`
					}
					_, err := io.WriteString(w, body)
					require.NoError(t, err)
				}))
				defer server.Close()
				opts := DefaultOptions().WithBaseURL(server.URL).WithClock(clock)
				if sourceBound {
					opts = opts.WithSourceFreshness(time.Minute, 0)
				}
				client, err := NewClient(opts)
				require.NoError(t, err)
				defer client.Close()
				type result struct {
					err   error
					users []AllowListUser
				}
				done := make(chan result, 1)
				go func() { users, e := client.GetUsers(context.Background()); done <- result{users: users, err: e} }()
				<-started
				client.InvalidateCache()
				close(release)
				got := <-done
				require.ErrorIs(t, got.err, ErrCacheInvalidated)
				require.Nil(t, got.users)
				require.Nil(t, client.cache.Get())
			})
		}
	}
}

func TestCacheReadCapturesInvalidationGeneration(t *testing.T) {
	cache := NewCache(time.Minute)
	_, old := cache.beginRead()
	cache.Clear()
	_, current := cache.beginRead()
	require.NoError(t, cache.commitRead([]AllowListUser{{UserID: "new", Scope: []string{"read"}}}, time.Time{}, current))
	require.ErrorIs(t, cache.commitRead([]AllowListUser{}, time.Time{}, old), ErrCacheInvalidated)
	require.Equal(t, "new", cache.Get()[0].UserID)
	cache.Clear()
	_, after := cache.beginRead()
	require.NoError(t, cache.commitRead([]AllowListUser{}, time.Time{}, after))
	require.NotNil(t, cache.Get())
	require.Empty(t, cache.Get())
}

func TestInvalidationChannelFencesInFlightResponse(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, err := io.WriteString(w, `[{"user_id":"old"}]`)
		require.NoError(t, err)
	}))
	defer server.Close()
	invalidations := make(chan struct{})
	opts := DefaultOptions().WithBaseURL(server.URL)
	opts.CacheInvalidationChannel = invalidations
	client, err := NewClient(opts)
	require.NoError(t, err)
	defer client.Close()
	done := make(chan error, 1)
	go func() { _, e := client.GetUsers(context.Background()); done <- e }()
	<-started
	invalidations <- struct{}{}
	// Receiving the second signal requires the listener to finish clearing the first.
	invalidations <- struct{}{}
	close(release)
	require.ErrorIs(t, <-done, ErrCacheInvalidated)
	require.Nil(t, client.cache.Get())
}
