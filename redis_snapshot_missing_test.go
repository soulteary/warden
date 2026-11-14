package main

import (
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

const migrationSnapshotKey = "warden:{migration-test}:snapshot:v2"

// rejectedCycle runs one refresh cycle whose load was rejected in full, as the
// writer for that cycle.
func rejectedCycle(t *testing.T, app *App) {
	t.Helper()
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	app.keepRejectedRefresh(writer, 1)
	release()
}

// The replica that published the snapshot stores it again when the key has
// gone and the source keeps it on its retained set, so a replica that starts
// afterwards still has something to restore.
func TestV2MissingSnapshotIsPublishedAgainByItsPublisher(t *testing.T) {
	for name, restoredFirst := range map[string]bool{"published_here": false, "restored_here": true} {
		t.Run(name, func(t *testing.T) {
			app, mr := migrationFixture(t, "v2")
			sourceTime := time.Now().Add(-time.Minute).UTC()
			users := []define.AllowListUser{{Mail: "kept@example.com", UserID: "kept"}}
			published := publishMigrationSource(t, app, sourceTime, users)
			if restoredFirst {
				// A replica that restored the snapshot holds its revision as well.
				app.snapshots, app.userCache = newSnapshotStore(), cache.NewSafeUserCache()
				require.True(t, app.bootstrapFromRedis())
			}
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)

			rejectedCycle(t, app)
			require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision, "a present key is renewed, not rewritten")

			mr.Del(migrationSnapshotKey)
			missing := snapshotOutcomeCount(t, "v2", "renew", "missing")
			republished := snapshotOutcomeCount(t, "v2", "publish", "published")
			rejectedCycle(t, app)
			require.Equal(t, missing+1, snapshotOutcomeCount(t, "v2", "renew", "missing"))
			require.Equal(t, republished+1, snapshotOutcomeCount(t, "v2", "publish", "published"))

			require.True(t, mr.Exists(migrationSnapshotKey), "the key is present again after the next cycle")
			require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(migrationSnapshotKey))
			require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, app))
			stored, err := app.redisSnapshots.Read(t.Context())
			require.NoError(t, err)
			require.Equal(t, sourceTime, stored.SourceSucceededAt, "publishing again does not renew the source evidence")
			current := app.snapshots.Load()
			require.NotEqual(t, published.RedisRevision, current.RedisRevision)
			require.Equal(t, stored.Revision, current.RedisRevision, "the local revision follows the new publication")
			require.Equal(t, sourceTime, current.SourceSucceededAt)

			// The following cycle renews the new publication instead of writing again.
			rejectedCycle(t, app)
			require.Equal(t, stored.Revision, app.snapshots.Load().RedisRevision)

			restarted := &App{redisClient: app.redisClient, redisSnapshotMode: "v2", redisSnapshots: app.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: app.log}
			require.True(t, restarted.bootstrapFromRedis())
			require.Equal(t, sourceTime, restarted.snapshots.Load().SourceSucceededAt)
		})
	}
}

// A different stored revision is another writer's publication. The replica
// holding the old revision neither renews nor replaces it.
func TestV2ConflictingSnapshotIsNotReplacedByOldRevisionHolder(t *testing.T) {
	older, mr := migrationFixture(t, "v2")
	newer := &App{redisClient: older.redisClient, redisSnapshotMode: "v2", redisSnapshots: older.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: older.log}
	base := time.Now().Add(-2 * time.Minute).UTC()
	first := publishMigrationSource(t, older, base, []define.AllowListUser{{Mail: "revoked@example.com"}, {Mail: "kept@example.com"}})
	publishMigrationSource(t, newer, base.Add(time.Minute), []define.AllowListUser{{Mail: "kept@example.com"}})
	raw, err := mr.Get(migrationSnapshotKey)
	require.NoError(t, err)

	conflicts := snapshotOutcomeCount(t, "v2", "renew", "revision_conflict")
	publications := snapshotOutcomeCount(t, "v2", "publish", "published")
	rejectedCycle(t, older)
	require.Equal(t, conflicts+1, snapshotOutcomeCount(t, "v2", "renew", "revision_conflict"))
	require.Equal(t, publications, snapshotOutcomeCount(t, "v2", "publish", "published"))

	after, err := mr.Get(migrationSnapshotKey)
	require.NoError(t, err)
	require.Equal(t, raw, after, "the newer publication is byte-for-byte unchanged")
	require.Equal(t, first.RedisRevision, older.snapshots.Load().RedisRevision)

	writer, release := older.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	require.ErrorIs(t, older.publishRedisSnapshots(older.userCache.Get()), cache.ErrSnapshotConflict)
	require.NotErrorIs(t, older.publishRedisSnapshots(older.userCache.Get()), cache.ErrSnapshotMissing)
}

// Publishing again needs the lease like any other publication: a writer whose
// lease has gone reports that and writes nothing.
func TestV2MissingSnapshotIsNotPublishedWithoutTheLease(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	published := publishMigrationSource(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
	app.snapshots.RecordRefreshFailure(reasonAllRecordsRejected)
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	mr.Del(migrationSnapshotKey)
	mr.Del(migrationSnapshotKey + ":lease")
	require.ErrorIs(t, app.publishRedisSnapshots(app.userCache.Get()), cache.ErrSnapshotLeaseLost)
	require.False(t, mr.Exists(migrationSnapshotKey))
	require.Equal(t, published.RedisRevision, app.snapshots.Load().RedisRevision)
}
