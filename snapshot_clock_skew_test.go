package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/stretchr/testify/require"
)

// A writer whose clock runs ahead by less than the allowance publishes a
// snapshot the store accepts. A reader that bootstraps it must serve it: the
// read decision applies the same allowance as the store.
func TestEnforcedReadServesSnapshotFromWriterWithClockAhead(t *testing.T) {
	for _, key := range []string{"IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS"} {
		t.Setenv(key, "")
	}
	writer, _ := migrationFixture(t, "v2")
	ahead := time.Now().Add(3 * time.Second).UTC()
	publishMigrationSource(t, writer, ahead, []define.AllowListUser{{UserID: "member", Mail: "member@example.com"}})

	limiter := middleware.NewRateLimits(0, 0, 0)
	t.Cleanup(limiter.Stop)
	reader := &App{
		redisClient: writer.redisClient, redisSnapshotMode: "v2", redisSnapshots: writer.redisSnapshots,
		userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: writer.log,
		snapshotMaxAge: 2 * time.Minute, snapshotReadEnforce: true, appMode: "DEFAULT", environment: "development",
		apiKey: "key", rateLimiter: limiter,
	}
	require.True(t, reader.bootstrapFromRedis())
	require.Equal(t, ahead, reader.snapshots.Load().SourceSucceededAt)

	pinned := captureHealthState(reader.snapshots, reader.userCache)
	decision := evaluateSnapshotRead(pinned, reader.snapshotMaxAge)
	require.Equal(t, readAllowed, decision.reason)
	require.Equal(t, ahead.Add(reader.snapshotMaxAge), decision.validUntil)
	require.Zero(t, reader.snapshots.AgeSeconds(), "a load time ahead of the local clock is not a negative age")

	mux := registerRoutes(reader)
	for _, path := range []string{"/", "/user?user_id=member", "/health"} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		req.Header.Set("X-API-Key", "key")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, path+rec.Body.String())
	}
}

// Beyond the allowance the store refuses the publication, so no reader can
// bootstrap it, and the read decision refuses the same source time.
func TestSourceTimeBeyondAllowanceIsRefusedByStoreAndReadDecision(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	users := []define.AllowListUser{{UserID: "member", Mail: "member@example.com"}}
	prepared, err := app.prepareUsers(users)
	require.NoError(t, err)
	beyond := time.Now().Add(cache.SNAPSHOT_FUTURE_ALLOWANCE + time.Second).UTC()
	app.snapshotNow = func() time.Time { return beyond }
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.Store(app.sourceSnapshot(&loader.LoadResult{Users: users, Version: cache.HashUserList(users), Source: loader.SourceRemote}, prepared.ReadView()))
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	require.ErrorIs(t, app.publishRedisSnapshots(app.userCache.Get()), cache.ErrSnapshotTimestamp)
	stored, err := app.redisSnapshots.Read(t.Context())
	require.NoError(t, err)
	require.Nil(t, stored, "nothing was stored")

	pinned := captureHealthState(app.snapshots, app.userCache)
	require.Equal(t, readSourceFuture, evaluateSnapshotRead(pinned, time.Minute).reason)
}

// The allowance is inclusive on both sides, as it is in the store, and a
// source time inside it never makes the deadline arithmetic report expiry
// before the source time has been reached.
func TestReadDecisionFutureAllowanceBoundary(t *testing.T) {
	app := responseMetadataApp(t)
	pinned := captureHealthState(app.snapshots, app.userCache)
	now := pinned.observedAt
	for _, tc := range []struct {
		name   string
		reason string
		ahead  time.Duration
		maxAge time.Duration
	}{
		{name: "one_nanosecond", ahead: time.Nanosecond, maxAge: time.Minute, reason: readAllowed},
		{name: "at_allowance", ahead: cache.SNAPSHOT_FUTURE_ALLOWANCE, maxAge: time.Minute, reason: readAllowed},
		{name: "past_allowance", ahead: cache.SNAPSHOT_FUTURE_ALLOWANCE + time.Nanosecond, maxAge: time.Minute, reason: readSourceFuture},
		{name: "max_age_below_skew", ahead: 3 * time.Second, maxAge: time.Second, reason: readAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := *app.snapshots.Load()
			snap.SourceSucceededAt = now.Add(tc.ahead)
			p := *pinned
			p.state = &snapshotState{snapshot: &snap}
			decision := evaluateSnapshotRead(&p, tc.maxAge)
			require.Equal(t, tc.reason, decision.reason)
			require.Equal(t, snap.SourceSucceededAt.Add(tc.maxAge), decision.validUntil)
		})
	}
}
