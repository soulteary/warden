package warden

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestListOlderResponseCannotReplaceNewerEmptyCache(t *testing.T) {
	for _, bound := range []bool{false, true} {
		t.Run(map[bool]string{true: "source", false: "default"}[bound], func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int64
			clock := newSourceTestClock()
			now := clock.Now()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					close(started)
					<-release
				}
				if bound {
					source := now
					if n == 1 {
						source = now.Add(-time.Minute)
					}
					setSourceTestHeaders(w.Header(), source, source.Add(2*time.Minute))
				}
				body := `[]`
				if n == 1 {
					body = `[{"user_id":"revoked","scope":["read"]}]`
				}
				_, err := io.WriteString(w, body)
				require.NoError(t, err)
			}))
			defer server.Close()
			opts := DefaultOptions().WithBaseURL(server.URL).WithClock(clock)
			if bound {
				opts = opts.WithSourceFreshness(2*time.Minute, 0)
			}
			client, err := NewClient(opts)
			require.NoError(t, err)
			defer client.Close()
			type result struct {
				err   error
				users []AllowListUser
			}
			done := make(chan result, 1)
			go func() { u, e := client.GetUsers(context.Background()); done <- result{err: e, users: u} }()
			<-started
			users, err := client.GetUsers(context.Background())
			require.NoError(t, err)
			require.Empty(t, users)
			close(release)
			old := <-done
			require.ErrorIs(t, old.err, ErrCacheSuperseded)
			require.Nil(t, old.users)
			users, err = client.GetUsers(context.Background())
			require.NoError(t, err)
			require.NotNil(t, users)
			require.Empty(t, users)
			require.EqualValues(t, 2, calls.Load())
		})
	}
}

func TestCacheResponseOrderingAndEpochs(t *testing.T) {
	cache := NewCache(time.Minute)
	_, first := cache.beginRead()
	_, second := cache.beginRead()
	// Starting a later request alone must not prevent an earlier success.
	require.NoError(t, cache.commitRead([]AllowListUser{{UserID: "first"}}, time.Time{}, first))
	require.NoError(t, cache.commitRead([]AllowListUser{{UserID: "second"}}, time.Time{}, second))
	require.ErrorIs(t, cache.commitRead([]AllowListUser{}, time.Time{}, first), ErrCacheSuperseded)
	require.Equal(t, "second", cache.Get()[0].UserID)
	cache.Clear()
	_, third := cache.beginRead()
	require.NoError(t, cache.commitRead([]AllowListUser{}, time.Time{}, third))
	require.ErrorIs(t, cache.commitRead([]AllowListUser{{UserID: "old"}}, time.Time{}, second), ErrCacheInvalidated)
	require.Empty(t, cache.Get())
}

func TestExpiredLaterResponseDoesNotFenceEarlierValidResponse(t *testing.T) {
	cache := NewCache(time.Minute)
	clock := newSourceTestClock()
	cache.now = clock.Now
	_, first := cache.beginRead()
	_, second := cache.beginRead()
	require.ErrorIs(t, cache.commitRead([]AllowListUser{}, clock.Now(), second), ErrSourceStale)
	require.NoError(t, cache.commitRead([]AllowListUser{{UserID: "valid"}}, clock.Now().Add(time.Minute), first))
	require.Equal(t, "valid", cache.Get()[0].UserID)
}
