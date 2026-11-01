package main

import (
	"context"
	"github.com/soulteary/warden/internal/define"
	sdk "github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type sdkContractClock struct{ nanos atomic.Int64 }

func (c *sdkContractClock) Now() time.Time   { return time.Unix(0, c.nanos.Load()).UTC() }
func (c *sdkContractClock) set(at time.Time) { c.nanos.Store(at.UnixNano()) }

func TestServerSDKFreshnessMatrix(t *testing.T) {
	for _, enforce := range []bool{false, true} {
		for _, bound := range []bool{false, true} {
			for _, state := range []string{"memory_fresh", "memory_stale", "memory_unknown", "v2_fresh", "v2_stale", "legacy_unknown", "empty_fresh"} {
				t.Run(state+map[bool]string{true: "/enforce", false: "/observe"}[enforce]+map[bool]string{true: "/bound", false: "/default"}[bound], func(t *testing.T) {
					checkServerSDKFreshness(t, enforce, bound, state)
				})
			}
		}
	}
}

func TestServerSDKCachedSourceDeadlineAndDefaultCache(t *testing.T) {
	for _, bound := range []bool{false, true} {
		t.Run(map[bool]string{true: "bound", false: "default"}[bound], func(t *testing.T) {
			app := responseMetadataApp(t)
			app.snapshotReadEnforce = true
			now := time.Now().UTC()
			snap := *app.snapshots.Load()
			snap.SourceSucceededAt = now.Add(-time.Minute)
			app.snapshots.Store(&snap)
			clock := &sdkContractClock{}
			clock.set(now)
			var calls atomic.Int64
			routes := registerRoutes(app)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); routes.ServeHTTP(w, r) }))
			defer server.Close()
			opts := sdk.DefaultOptions().WithBaseURL(server.URL).WithAPIKey("key").WithCacheTTL(5 * time.Minute).WithClock(clock)
			if bound {
				opts = opts.WithSourceFreshness(2*time.Minute, 0)
			}
			client, err := sdk.NewClient(opts)
			require.NoError(t, err)
			defer client.Close()
			users, err := client.GetUsers(context.Background())
			require.NoError(t, err)
			require.Len(t, users, 1)
			deadline := snap.SourceSucceededAt.Add(app.snapshotMaxAge)
			clock.set(deadline)
			// Make the real server reject new requests without sleeping or changing its clock.
			stale := snap
			stale.SourceSucceededAt = now.Add(-time.Hour)
			app.snapshots.Store(&stale)
			users, err = client.GetUsers(context.Background())
			if bound {
				require.Error(t, err)
				require.Nil(t, users)
				require.EqualValues(t, 2, calls.Load())
			} else {
				require.NoError(t, err)
				require.Len(t, users, 1)
				require.EqualValues(t, 1, calls.Load())
			}
		})
	}
}

func TestServerSDKAuthenticationBeforeFreshness(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshotReadEnforce = true
	snap := *app.snapshots.Load()
	snap.SourceSucceededAt = time.Time{}
	app.snapshots.Store(&snap)
	server := httptest.NewServer(registerRoutes(app))
	defer server.Close()
	client, err := sdk.NewClient(sdk.DefaultOptions().WithBaseURL(server.URL).WithAPIKey("wrong").WithSourceFreshness(time.Minute, 0))
	require.NoError(t, err)
	defer client.Close()
	_, err = client.GetUsers(context.Background())
	var e *sdk.Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, sdk.ErrCodeUnauthorized, e.Code)
}

type sdkStepClock struct {
	initial time.Time
	after   time.Time
	calls   atomic.Int64
}

func (c *sdkStepClock) Now() time.Time {
	if c.calls.Add(1) == 1 {
		return c.initial
	}
	return c.after
}

func TestServerSDKRechecksDeadlineAfterDecode(t *testing.T) {
	app := responseMetadataApp(t)
	app.snapshotReadEnforce = true
	snap := app.snapshots.Load()
	now := time.Now().UTC()
	deadline := snap.SourceSucceededAt.Add(app.snapshotMaxAge)
	clock := &sdkStepClock{initial: now, after: deadline}
	var calls atomic.Int64
	routes := registerRoutes(app)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); routes.ServeHTTP(w, r) }))
	defer server.Close()
	client, err := sdk.NewClient(sdk.DefaultOptions().WithBaseURL(server.URL).WithAPIKey("key").WithClock(clock).WithSourceFreshness(2*time.Minute, 0))
	require.NoError(t, err)
	defer client.Close()
	users, err := client.GetUsers(context.Background())
	require.ErrorIs(t, err, sdk.ErrSourceStale)
	require.Nil(t, users)
	_, err = client.GetUsers(context.Background())
	require.ErrorIs(t, err, sdk.ErrSourceStale)
	require.EqualValues(t, 2, calls.Load(), "expired decoded rows must not enter the cache")
}

func TestServerSource503NegotiationAndAuthentication(t *testing.T) {
	for _, state := range []string{"expired", "unknown", "future", "no_snapshot", "view_unknown"} {
		t.Run(state, func(t *testing.T) {
			app := responseMetadataApp(t)
			app.snapshotReadEnforce = true
			snap := *app.snapshots.Load()
			switch state {
			case "expired":
				snap.SourceSucceededAt = time.Now().Add(-time.Hour)
			case "future":
				snap.SourceSucceededAt = time.Now().Add(time.Hour)
			case "unknown":
				snap.SourceSucceededAt = time.Time{}
			case "view_unknown":
				snap.view = nil
			}
			app.snapshots.Store(&snap)
			if state == "no_snapshot" {
				app.snapshots = newSnapshotStore()
			}
			routes := registerRoutes(app)
			for _, authenticated := range []bool{false, true} {
				for _, negotiated := range []bool{false, true} {
					req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
					if authenticated {
						req.Header.Set("X-API-Key", "key")
					}
					if negotiated {
						req.Header.Set(snapshotMetadataHeader, "v1")
					}
					rec := httptest.NewRecorder()
					routes.ServeHTTP(rec, req)
					if !authenticated {
						require.Equal(t, 401, rec.Code)
					} else {
						require.Equal(t, 503, rec.Code)
					}
					if authenticated && negotiated {
						expected := expectedSourceRefusal(state)
						require.Equal(t, expected, rec.Header().Get(sourceErrorHeader))
						require.Equal(t, "v1", rec.Header().Get(snapshotMetadataHeader))
					} else {
						require.Empty(t, rec.Header().Get(sourceErrorHeader))
						require.Empty(t, rec.Header().Get(snapshotMetadataHeader))
					}
				}
			}
		})
	}
}

func TestServerSDKSource503KeepsRetryPolicy(t *testing.T) {
	for _, source := range []bool{false, true} {
		app := responseMetadataApp(t)
		app.snapshotReadEnforce = true
		snap := *app.snapshots.Load()
		snap.SourceSucceededAt = time.Now().Add(-time.Hour)
		app.snapshots.Store(&snap)
		var calls atomic.Int64
		routes := registerRoutes(app)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if !source {
				http.Error(w, "ordinary failure", http.StatusServiceUnavailable)
				return
			}
			routes.ServeHTTP(w, r)
		}))
		opts := sdk.DefaultOptions().WithBaseURL(server.URL).WithAPIKey("key").WithSourceFreshness(time.Minute, 0).WithRetry(&sdk.RetryOptions{MaxRetries: 2, RetryDelay: time.Nanosecond, MaxRetryDelay: time.Nanosecond, BackoffMultiplier: 1, RetryableStatusCodes: []int{503}})
		client, err := sdk.NewClient(opts)
		require.NoError(t, err)
		_, err = client.GetUsers(context.Background())
		var e *sdk.Error
		require.ErrorAs(t, err, &e)
		if source {
			require.ErrorIs(t, err, sdk.ErrSourceStale)
			require.Equal(t, sdk.ErrCodeSourceFreshness, e.Code)
		} else {
			require.Equal(t, sdk.ErrCodeServerError, e.Code)
		}
		require.EqualValues(t, 3, calls.Load())
		client.Close()
		server.Close()
	}
}

func checkServerSDKFreshness(t *testing.T, enforce, bound bool, state string) {
	t.Helper()

	app := responseMetadataApp(t)
	app.snapshotReadEnforce = enforce
	now := time.Now().UTC()
	source := now.Add(-time.Second)
	stale := state == "memory_stale" || state == "v2_stale"
	unknown := state == "memory_unknown" || state == "legacy_unknown"
	if stale {
		source = now.Add(-time.Hour)
	}
	snap := *app.snapshots.Load()
	snap.SourceSucceededAt = source
	if unknown {
		snap.SourceSucceededAt = time.Time{}
	}
	if state == "empty_fresh" {
		_, empty := preparedTestSnapshot(t, []define.AllowListUser{})
		snap = *empty
		snap.SourceSucceededAt = source
	}
	app.snapshots.Store(&snap)
	if state == "legacy_unknown" {
		app.snapshots.Store(snapshotFromRedis(app.userCache.Get(), app.userCache.ReadView()))
	}
	if state == "v2_fresh" || state == "v2_stale" {
		backing, _ := migrationFixture(t, "v2")
		publishMigrationSource(t, backing, source, []define.AllowListUser{{Mail: "member@example.com", UserID: "member"}})
		app.redisSnapshotMode = "v2"
		app.redisSnapshots = backing.redisSnapshots
		require.True(t, app.bootstrapFromRedis())
	}
	server := httptest.NewServer(registerRoutes(app))
	defer server.Close()
	opts := sdk.DefaultOptions().WithBaseURL(server.URL).WithAPIKey("key")
	if bound {
		opts = opts.WithSourceFreshness(2*time.Minute, 0)
	}
	client, err := sdk.NewClient(opts)
	require.NoError(t, err)
	defer client.Close()
	for _, method := range []string{"list", "page", "user", "missing"} {
		checkSDKFreshnessMethod(t, client, method, state, enforce, bound, stale, unknown)
	}
}

func checkSDKFreshnessMethod(t *testing.T, client *sdk.Client, method, state string, enforce, bound, stale, unknown bool) {
	t.Helper()
	var callErr error
	switch method {
	case "list":
		var users []sdk.AllowListUser
		users, callErr = client.GetUsers(context.Background())
		if callErr == nil {
			if state == "empty_fresh" {
				require.Empty(t, users)
			} else {
				require.Equal(t, "member", users[0].UserID)
			}
		}
	case "page":
		_, callErr = client.GetUsersPaginated(context.Background(), 1, 1)
	case "user":
		_, callErr = client.GetUserByIdentifier(context.Background(), "", "", "member")
	case "missing":
		_, callErr = client.GetUserByIdentifier(context.Background(), "", "", "missing")
	}
	code := expectedSDKFreshnessCode(method, state, enforce, bound, stale, unknown)
	if code == "" {
		require.NoError(t, callErr, method)
	} else {
		var e *sdk.Error
		require.ErrorAs(t, callErr, &e, method)
		require.Equal(t, code, e.Code, method)
		if code == sdk.ErrCodeSourceFreshness {
			if stale {
				require.ErrorIs(t, callErr, sdk.ErrSourceStale)
			} else {
				require.ErrorIs(t, callErr, sdk.ErrSourceUnknown)
			}
		}
	}

}

func expectedSDKFreshnessCode(method, state string, enforce, bound, stale, unknown bool) string {
	code := ""
	switch {
	case enforce && (stale || unknown) && !bound:
		code = sdk.ErrCodeServerError
	case bound && (stale || unknown):
		code = sdk.ErrCodeSourceFreshness
	case method == "missing" || (method == "user" && state == "empty_fresh"):
		code = sdk.ErrCodeNotFound
	}

	return code
}

func expectedSourceRefusal(state string) string {
	expected := state
	if state == "no_snapshot" || state == "view_unknown" {
		expected = "unknown"
	}
	return expected
}
