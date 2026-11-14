package main

import (
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// A stored value the renewal cannot use at all (over the size limit, or not
// JSON) is replaced by the lease holder even when it holds a revision, as a
// replica without a revision already does. Otherwise the key keeps a value no
// replica can restore for as long as the source keeps the writer on its
// retained set.
func TestV2UnusableSnapshotIsReplacedByRevisionHolder(t *testing.T) {
	for name, tc := range map[string]struct{ stored, outcome string }{
		"not_json":       {"{", "invalid_envelope"},
		"empty_value":    {"", "invalid_envelope"},
		"json_scalar":    {"7", "invalid_envelope"},
		"over_the_limit": {strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1), "too_large"},
	} {
		t.Run(name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			sourceTime := time.Now().Add(-time.Minute).UTC()
			published := publishMigrationSource(t, app, sourceTime, []define.AllowListUser{{Mail: "kept@example.com", UserID: "kept"}})
			rejectedCycle(t, app)
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)

			require.NoError(t, mr.Set(migrationSnapshotKey, tc.stored))
			renewals := snapshotOutcomeCount(t, "v2", "renew", tc.outcome)
			publications := snapshotOutcomeCount(t, "v2", "publish", "published")
			rejectedCycle(t, app)
			require.Equal(t, renewals+1, snapshotOutcomeCount(t, "v2", "renew", tc.outcome))
			require.Equal(t, publications+1, snapshotOutcomeCount(t, "v2", "publish", "published"))

			stored, err := app.redisSnapshots.Read(t.Context())
			require.NoError(t, err, "the key is usable again after the next cycle")
			require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, app))
			require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(migrationSnapshotKey))
			require.Equal(t, sourceTime, stored.SourceSucceededAt, "the replacement does not renew the source evidence")
			current := app.snapshots.Load()
			require.NotEqual(t, published.RedisRevision, current.RedisRevision)
			require.Equal(t, stored.Revision, current.RedisRevision)

			// The following cycle renews the replacement instead of writing again.
			rejectedCycle(t, app)
			require.Equal(t, stored.Revision, app.snapshots.Load().RedisRevision)

			restarted := &App{redisClient: app.redisClient, redisSnapshotMode: "v2", redisSnapshots: app.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: app.log}
			require.True(t, restarted.bootstrapFromRedis())
		})
	}
}

// A stored value that decodes but carries another revision stays a conflict,
// whatever else it holds: it may be a publication this replica cannot judge.
func TestV2DecodableSnapshotWithAnotherRevisionIsLeftAlone(t *testing.T) {
	for name, stored := range map[string]string{
		"other_schema":      `{"schema_version":3,"revision":"` + strings.Repeat("b", 32) + `"}`,
		"other_revision":    `{"revision":"` + strings.Repeat("c", 32) + `"}`,
		"object_no_version": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			published := publishMigrationSource(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
			require.NoError(t, mr.Set(migrationSnapshotKey, stored))
			conflicts := snapshotOutcomeCount(t, "v2", "renew", "revision_conflict")
			publications := snapshotOutcomeCount(t, "v2", "publish", "published")
			rejectedCycle(t, app)
			require.Equal(t, conflicts+1, snapshotOutcomeCount(t, "v2", "renew", "revision_conflict"))
			require.Equal(t, publications, snapshotOutcomeCount(t, "v2", "publish", "published"))
			after, err := mr.Get(migrationSnapshotKey)
			require.NoError(t, err)
			require.Equal(t, stored, after)
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)
		})
	}
}

// The replacement of an unusable value follows the rule for a retained set: a
// lease is still required.
func TestV2UnusableSnapshotIsNotReplacedWithoutTheLease(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	published := publishMigrationSource(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
	app.snapshots.RecordRefreshFailure(reasonAllRecordsRejected)
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	require.NoError(t, mr.Set(migrationSnapshotKey, "{"))
	mr.Del(migrationSnapshotKey + ":lease")
	require.ErrorIs(t, app.publishRedisSnapshots(app.userCache.Get()), cache.ErrSnapshotLeaseLost)
	after, err := mr.Get(migrationSnapshotKey)
	require.NoError(t, err)
	require.Equal(t, "{", after)
	require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)
}
