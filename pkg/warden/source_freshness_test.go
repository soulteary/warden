package warden

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type sourceTestClock struct {
	onNow func()
	nanos atomic.Int64
}

func newSourceTestClock() *sourceTestClock {
	c := &sourceTestClock{}
	c.set(time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC))
	return c
}
func (c *sourceTestClock) Now() time.Time {
	at := time.Unix(0, c.nanos.Load()).UTC()
	if c.onNow != nil {
		c.onNow()
	}
	return at
}
func (c *sourceTestClock) set(at time.Time) { c.nanos.Store(at.UnixNano()) }

func setSourceTestHeaders(h http.Header, source, until time.Time) {
	h.Set(HeaderSnapshotMetadata, "v1")
	h.Set(HeaderSnapshotVersion, strings.Repeat("a", 64))
	h.Set(HeaderSourceSucceededAt, source.Format(time.RFC3339Nano))
	h.Set(HeaderSnapshotValidUntil, until.Format(time.RFC3339Nano))
}

func TestSourceFreshnessCacheUsesOriginalDeadline(t *testing.T) {
	clock := newSourceTestClock()
	initial := clock.Now()
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "v1", r.Header.Get(HeaderSnapshotMetadata))
		setSourceTestHeaders(w.Header(), initial.Add(-time.Minute), initial.Add(20*time.Second))
		_, err := io.WriteString(w, `[{"user_id":"member","scope":["read"]}]`)
		require.NoError(t, err)
	}))
	defer server.Close()
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithCacheTTL(5*time.Minute).WithClock(clock).WithSourceFreshness(2*time.Minute, 5*time.Second))
	require.NoError(t, err)
	defer client.Close()
	first, err := client.GetUsers(context.Background())
	require.NoError(t, err)
	first[0].Scope[0] = "changed"
	clock.set(initial.Add(14 * time.Second))
	cached, err := client.GetUsers(context.Background())
	require.NoError(t, err)
	require.Equal(t, "read", cached[0].Scope[0])
	require.EqualValues(t, 1, calls.Load())
	clock.set(initial.Add(15 * time.Second))
	_, err = client.GetUsers(context.Background())
	require.ErrorIs(t, err, ErrSourceStale)
	require.EqualValues(t, 2, calls.Load())
}

func TestSourceFreshnessLocalAgeAndTTLBounds(t *testing.T) {
	clock := newSourceTestClock()
	initial := clock.Now()
	client, err := NewClient(DefaultOptions().WithBaseURL("https://example.com").WithClock(clock).WithSourceFreshness(time.Minute, 5*time.Second))
	require.NoError(t, err)
	headers := http.Header{}
	setSourceTestHeaders(headers, initial.Add(-30*time.Second), initial.Add(time.Hour))
	deadline, err := client.sourceDeadline(headers)
	require.NoError(t, err)
	require.Equal(t, initial.Add(25*time.Second), deadline)
	client.cache.ttl = 10 * time.Second
	require.True(t, client.cache.setUntil([]AllowListUser{}, deadline))
	clock.set(initial.Add(10 * time.Second))
	require.Nil(t, client.cache.Get(), "local TTL remains a tighter bound")
	// Cache invalidation clears both the data and its generation-specific bound.
	clock.set(initial)
	require.True(t, client.cache.setUntil([]AllowListUser{}, deadline))
	client.ClearCache()
	require.Nil(t, client.cache.Get())
}

func TestSourceFreshnessRejectsUnknownMalformedAndFutureEvidence(t *testing.T) {
	clock := newSourceTestClock()
	now := clock.Now()
	client, err := NewClient(DefaultOptions().WithBaseURL("https://example.com").WithClock(clock).WithSourceFreshness(time.Minute, 5*time.Second))
	require.NoError(t, err)
	for _, tc := range []struct {
		change func(http.Header)
		want   error
		name   string
	}{
		{name: "old_server", change: func(h http.Header) {
			for k := range h {
				h.Del(k)
			}
		}, want: ErrSourceUnknown},
		{name: "version", change: func(h http.Header) { h.Set(HeaderSnapshotVersion, "invalid") }, want: ErrSourceUnknown},
		{name: "time", change: func(h http.Header) { h.Set(HeaderSourceSucceededAt, "invalid") }, want: ErrSourceUnknown},
		{name: "duplicate", change: func(h http.Header) { h.Add(HeaderSourceSucceededAt, now.Format(time.RFC3339Nano)) }, want: ErrSourceUnknown},
		{name: "unknown", change: func(h http.Header) { h.Del(HeaderSourceSucceededAt) }, want: ErrSourceUnknown},
		{name: "invalid_until", change: func(h http.Header) { h.Set(HeaderSnapshotValidUntil, now.Add(-time.Minute).Format(time.RFC3339Nano)) }, want: ErrSourceUnknown},
		{name: "future", change: func(h http.Header) { h.Set(HeaderSourceSucceededAt, now.Add(6*time.Second).Format(time.RFC3339Nano)) }, want: ErrSourceFuture},
		{name: "expired", change: func(h http.Header) { setSourceTestHeaders(h, now.Add(-time.Hour), now.Add(-time.Minute)) }, want: ErrSourceStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{}
			setSourceTestHeaders(headers, now.Add(-time.Second), now.Add(time.Minute))
			tc.change(headers)
			_, freshErr := client.sourceDeadline(headers)
			require.ErrorIs(t, freshErr, tc.want)
			var sdkErr *Error
			require.ErrorAs(t, freshErr, &sdkErr)
			require.Equal(t, ErrCodeSourceFreshness, sdkErr.Code)
		})
	}
	headers := http.Header{}
	setSourceTestHeaders(headers, now.Add(5*time.Second), now.Add(time.Minute))
	_, err = client.sourceDeadline(headers)
	require.NoError(t, err, "future allowance boundary is inclusive")
}

func TestSourceFreshnessPreservesDefaultResponsesAndStatusErrors(t *testing.T) {
	clock := newSourceTestClock()
	var status atomic.Int64
	status.Store(200)
	var include atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if include.Load() {
			setSourceTestHeaders(w.Header(), clock.Now().Add(-time.Second), clock.Now().Add(time.Minute))
		}
		w.WriteHeader(int(status.Load()))
		body := `[]`
		if r.URL.Query().Get("page") != "" {
			body = `{"data":[],"pagination":{"page":1,"page_size":1,"total":0,"total_pages":0}}`
		}
		_, err := io.WriteString(w, body)
		require.NoError(t, err)
	}))
	defer server.Close()
	compatible, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	_, err = compatible.GetUsers(context.Background())
	require.NoError(t, err, "old servers stay compatible by default")
	strict, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithClock(clock).WithSourceFreshness(time.Minute, 0))
	require.NoError(t, err)
	_, err = strict.GetUsers(context.Background())
	require.ErrorIs(t, err, ErrSourceUnknown)
	status.Store(503)
	_, err = strict.GetUsers(context.Background())
	var sdkErr *Error
	require.ErrorAs(t, err, &sdkErr)
	require.Equal(t, ErrCodeServerError, sdkErr.Code)
	status.Store(404)
	_, err = strict.GetUserByIdentifier(context.Background(), "", "", "missing")
	require.ErrorIs(t, err, ErrSourceUnknown)
	include.Store(true)
	_, err = strict.GetUserByIdentifier(context.Background(), "", "", "missing")
	require.ErrorAs(t, err, &sdkErr)
	require.Equal(t, ErrCodeNotFound, sdkErr.Code)
	status.Store(200)
	_, err = strict.GetUsersPaginated(context.Background(), 1, 1)
	require.NoError(t, err, "all successful read methods check source evidence")
}

func TestSourceFreshnessRechecksAfterBodyRead(t *testing.T) {
	methods := []struct {
		read func(context.Context, *Client) error
		name string
		body string
	}{
		{name: "list", body: `[]`, read: func(ctx context.Context, c *Client) error { _, err := c.GetUsers(ctx); return err }},
		{name: "page", body: `{"data":[],"pagination":{"page":1,"page_size":1,"total":0,"total_pages":0}}`, read: func(ctx context.Context, c *Client) error { _, err := c.GetUsersPaginated(ctx, 1, 1); return err }},
		{name: "user", body: `{"user_id":"member"}`, read: func(ctx context.Context, c *Client) error {
			_, err := c.GetUserByIdentifier(ctx, "", "", "member")
			return err
		}},
	}
	for _, method := range methods {
		for _, elapsed := range []time.Duration{time.Second - time.Nanosecond, time.Second, time.Second + time.Nanosecond} {
			t.Run(method.name+"/"+elapsed.String(), func(t *testing.T) {
				clock := newSourceTestClock()
				now := clock.Now()
				validated := make(chan struct{})
				var once sync.Once
				clock.onNow = func() { once.Do(func() { close(validated) }) }
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					setSourceTestHeaders(w.Header(), now, now.Add(time.Second))
					w.WriteHeader(http.StatusOK)
					if err := http.NewResponseController(w).Flush(); err != nil {
						t.Errorf("flush response headers: %v", err)
						return
					}
					select {
					case <-validated:
					case <-r.Context().Done():
						return
					}
					clock.set(now.Add(elapsed))
					if _, err := io.WriteString(w, method.body); err != nil {
						t.Errorf("write response: %v", err)
					}
				}))
				t.Cleanup(server.Close)
				client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithClock(clock).WithMaxResponseBytes(-1).WithSourceFreshness(time.Minute, 0))
				require.NoError(t, err)
				t.Cleanup(client.Close)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err = method.read(ctx, client)
				if elapsed < time.Second {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, ErrSourceStale)
					require.Nil(t, client.cache.Get(), "expired response must not populate the list cache")
				}
			})
		}
	}
}

func TestSourceFreshnessExpiredResponseCannotEraseConcurrentFreshCache(t *testing.T) {
	clock := newSourceTestClock()
	now := clock.Now()
	cache := NewCache(time.Minute)
	cache.now = clock.Now
	require.True(t, cache.setUntil([]AllowListUser{{UserID: "new"}}, now.Add(time.Minute)))
	require.False(t, cache.setUntil([]AllowListUser{{UserID: "old"}}, now))
	require.Equal(t, "new", cache.Get()[0].UserID)
}

func TestSourceFreshnessOptions(t *testing.T) {
	for _, bounds := range [][2]time.Duration{{-1, 0}, {time.Minute, -1}, {time.Second, time.Second}} {
		_, err := NewClient(DefaultOptions().WithBaseURL("https://example.com").WithSourceFreshness(bounds[0], bounds[1]))
		require.Error(t, err)
	}
	_, err := NewClient(DefaultOptions().WithBaseURL("https://example.com").WithSourceFreshness(0, 0))
	require.NoError(t, err)
	require.False(t, errors.Is(ErrSourceUnknown, ErrSourceStale))
}
