package main

import (
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	rediskitlock "github.com/soulteary/redis-kit/lock"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
)

// legacyReplica is a replica in the default snapshot mode on a shared Redis,
// with the real refresh lock.
func legacyReplica(t *testing.T, mr *miniredis.Miniredis) *App {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	app := &App{
		redisClient: client, redisSnapshotMode: "legacy", redisUserCache: cache.NewRedisUserCache(client),
		redisRefreshLocker: rediskitlock.NewRedisLockerWithLockTime(client, redisRefreshLockTTL),
		userCache:          cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: logger.GetLoggerKit(),
	}
	app.loadFromRedis, app.redisCacheExists = app.readRedisUserCache, app.readRedisUserCacheExists
	return app
}

func legacyStoredMails(t *testing.T, mr *miniredis.Miniredis) []string {
	t.Helper()
	// What a replica that starts now adopts.
	starting := legacyReplica(t, mr)
	require.True(t, starting.bootstrapFromRedis())
	mails := []string{}
	users := starting.userCache.Get()
	for i := range users {
		mails = append(mails, users[i].Mail)
	}
	return mails
}

var (
	legacyOlderSet = []define.AllowListUser{{Mail: "kept@example.com"}, {Mail: "removed@example.com"}}
	legacyNewerSet = []define.AllowListUser{{Mail: "kept@example.com"}}
)

// Replica B holds a set loaded before a user was removed; replica A published
// the set without that user. When the source then returns only records that
// are rejected, B keeps its set and wins the lock: it must not write the older
// set back, or a replica that starts meanwhile serves the removed user.
func TestLegacyRetainedSetDoesNotReplaceNewerStoredSet(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := legacyReplica(t, mr), legacyReplica(t, mr)
	base := time.Now().Add(-2 * time.Minute).UTC()
	loadWithoutPublishing(t, b, base, legacyOlderSet)
	publishMigrationSource(t, a, base.Add(time.Minute), legacyNewerSet)
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))

	rejectedCycle(t, b)
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr), "the later source result stays stored")
	require.Len(t, b.userCache.Get(), 2, "the replica itself keeps serving the set it retained")

	// The source stays unusable for both. The stored set is renewed by the
	// replica that holds it and left alone by the other, in either order.
	for cycle := 0; cycle < 3; cycle++ {
		mr.FastForward(40 * time.Minute)
		require.Equal(t, 20*time.Minute, mr.TTL(cache.REDIS_CACHE_KEY))
		rejectedCycle(t, b)
		require.Equal(t, 20*time.Minute, mr.TTL(cache.REDIS_CACHE_KEY), "a set the replica does not hold is not renewed")
		rejectedCycle(t, a)
		require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(cache.REDIS_CACHE_KEY), "the stored set does not expire while its holder runs")
		require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))
	}

	// Only the replica with the older set is left. Once the stored set has
	// expired there is nothing newer to protect, and Redis is not left empty.
	mr.FastForward(cache.REDIS_CACHE_TTL + time.Second)
	require.False(t, mr.Exists(cache.REDIS_CACHE_KEY))
	rejectedCycle(t, b)
	require.ElementsMatch(t, []string{"kept@example.com", "removed@example.com"}, legacyStoredMails(t, mr))
	require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(cache.REDIS_CACHE_KEY))
}

// The opposite order: the stored set is the older one, and the replica that
// loaded later without being the writer keeps the newer one. Its retained set
// replaces the stored one, and the older set is not written back afterwards.
func TestLegacyRetainedSetReplacesOlderStoredSet(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := legacyReplica(t, mr), legacyReplica(t, mr)
	base := time.Now().Add(-2 * time.Minute).UTC()
	publishMigrationSource(t, a, base, legacyOlderSet)
	loadWithoutPublishing(t, b, base.Add(time.Minute), legacyNewerSet)

	rejectedCycle(t, a)
	require.Len(t, legacyStoredMails(t, mr), 2, "nothing newer is stored yet")
	rejectedCycle(t, b)
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))
	for cycle := 0; cycle < 2; cycle++ {
		rejectedCycle(t, a)
		require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr), "the stored set does not alternate between the replicas")
		rejectedCycle(t, b)
		require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))
	}
}

// A successful load is new information and is written whatever is stored,
// including over a set recorded with a later source time.
func TestLegacyCurrentLoadIsAlwaysWritten(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := legacyReplica(t, mr), legacyReplica(t, mr)
	base := time.Now().Add(-2 * time.Minute).UTC()
	publishMigrationSource(t, a, base.Add(time.Minute), legacyNewerSet)
	publishMigrationSource(t, b, base, legacyOlderSet)
	require.Len(t, legacyStoredMails(t, mr), 2)
	publishMigrationSource(t, b, base, legacyOlderSet)
	require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(cache.REDIS_CACHE_KEY))
}

// During a rolling upgrade a replica of an earlier release writes the array
// without a source record. The stored set then has no known source time and a
// retained set is written over it, as before this record existed.
func TestLegacyRetainedSetReplacesSetWithoutSourceRecord(t *testing.T) {
	for name, previous := range map[string]func(t *testing.T, a *App, base time.Time){
		"no_record": func(*testing.T, *App, time.Time) {},
		"record_of_a_replaced_value": func(t *testing.T, a *App, base time.Time) {
			publishMigrationSource(t, a, base.Add(time.Hour), []define.AllowListUser{{Mail: "earlier@example.com"}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			a, b := legacyReplica(t, mr), legacyReplica(t, mr)
			base := time.Now().Add(-2 * time.Hour).UTC()
			previous(t, a, base)
			// What a release without the record does: the array and its version only.
			require.NoError(t, a.redisUserCache.Set(legacyNewerSet))
			loadWithoutPublishing(t, b, base, legacyOlderSet)
			rejectedCycle(t, b)
			require.Len(t, legacyStoredMails(t, mr), 2)

			// A replica of the earlier release still reads what this one wrote.
			raw, err := mr.Get(cache.REDIS_CACHE_KEY)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(raw, `[{"phone":`), "the stored value is the plain user array")
		})
	}
}

// A set restored from Redis has no source time. It renews the stored set it
// was restored from and does not replace a different one that has a record.
func TestLegacyRestoredSetDoesNotReplaceRecordedSet(t *testing.T) {
	mr := miniredis.RunT(t)
	a := legacyReplica(t, mr)
	base := time.Now().Add(-2 * time.Minute).UTC()
	publishMigrationSource(t, a, base, legacyOlderSet)
	restored := legacyReplica(t, mr)
	require.True(t, restored.bootstrapFromRedis())
	require.True(t, restored.snapshots.Load().SourceSucceededAt.IsZero())

	mr.FastForward(40 * time.Minute)
	rejectedCycle(t, restored)
	require.Equal(t, cache.REDIS_CACHE_TTL, mr.TTL(cache.REDIS_CACHE_KEY), "the set it restored is renewed")

	publishMigrationSource(t, a, base.Add(time.Minute), legacyNewerSet)
	rejectedCycle(t, restored)
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))

	mr.Del(cache.REDIS_CACHE_KEY)
	rejectedCycle(t, restored)
	require.Len(t, legacyStoredMails(t, mr), 2, "a missing key is written")
}

// When the comparison cannot be made the retained set is not written and the
// failure is reported after the usual retries.
func TestLegacyRetainedSetIsNotWrittenWhenRedisFails(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := legacyReplica(t, mr), legacyReplica(t, mr)
	base := time.Now().Add(-2 * time.Minute).UTC()
	loadWithoutPublishing(t, b, base, legacyOlderSet)
	publishMigrationSource(t, a, base.Add(time.Minute), legacyNewerSet)
	b.snapshots.RecordRefreshFailure(reasonAllRecordsRejected)
	mr.SetError("LOADING Redis is loading the dataset in memory")
	err := b.publishRedisSnapshots(b.userCache.Get())
	mr.SetError("")
	require.ErrorContains(t, err, "failed to update Redis cache")
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))
}

// Shadow mode writes the legacy array under the same rule.
func TestShadowRetainedSetDoesNotReplaceNewerLegacySet(t *testing.T) {
	a, mr := migrationFixture(t, "shadow")
	a.redisRefreshLocker = rediskitlock.NewRedisLockerWithLockTime(a.redisClient, redisRefreshLockTTL)
	b := &App{
		redisClient: a.redisClient, redisSnapshotMode: "shadow", redisSnapshots: a.redisSnapshots, redisUserCache: a.redisUserCache,
		redisRefreshLocker: a.redisRefreshLocker, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: a.log,
	}
	base := time.Now().Add(-2 * time.Minute).UTC()
	loadWithoutPublishing(t, b, base, legacyOlderSet)
	publishMigrationSource(t, a, base.Add(time.Minute), legacyNewerSet)
	rejectedCycle(t, b)
	require.Equal(t, []string{"kept@example.com"}, legacyStoredMails(t, mr))
	require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, a))
}
