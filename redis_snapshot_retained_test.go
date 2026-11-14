package main

import (
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/require"
)

// loadWithoutPublishing gives a replica a successful source load it did not
// publish, as happens on every cycle to the replicas that are not the writer.
func loadWithoutPublishing(t *testing.T, app *App, at time.Time, users []define.AllowListUser) {
	t.Helper()
	prepared, err := app.prepareUsers(users)
	require.NoError(t, err)
	app.snapshotNow = func() time.Time { return at }
	res := &loader.LoadResult{Users: users, Version: cache.HashUserList(users), Source: loader.SourceRemote, LoadedAt: at}
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.Store(app.sourceSnapshot(res, prepared.ReadView()))
}

func storedSnapshotMails(t *testing.T, app *App) []string {
	t.Helper()
	envelope, err := app.redisSnapshots.Read(t.Context())
	require.NoError(t, err)
	require.NotNil(t, envelope)
	mails := make([]string, 0, len(envelope.Users))
	for i := range envelope.Users {
		mails = append(mails, envelope.Users[i].Mail)
	}
	return mails
}

// A replica that keeps its last known good rule set after an all-rejected load
// must not replace a publication made from a later successful source load.
func TestV2RetainedSnapshotDoesNotReplaceNewerPublication(t *testing.T) {
	newer, _ := migrationFixture(t, "v2")
	older := &App{redisClient: newer.redisClient, redisSnapshotMode: "v2", redisSnapshots: newer.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: newer.log}

	base := time.Now().Add(-2 * time.Minute).UTC()
	loadWithoutPublishing(t, older, base, []define.AllowListUser{{Mail: "revoked@example.com"}, {Mail: "kept@example.com"}})
	publishMigrationSource(t, newer, base.Add(time.Minute), []define.AllowListUser{{Mail: "kept@example.com"}})
	require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, newer))

	writer, release := older.acquireRedisRefreshWriter()
	require.True(t, writer)
	older.keepRejectedRefresh(writer, 2)
	release()

	require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, newer), "the later source result stays published")
	require.Empty(t, older.snapshots.Load().RedisRevision, "nothing was published by the retaining replica")
}

// With nothing newer stored, the retained set is still published, so the
// shared copy survives while the source stays unusable.
func TestV2RetainedSnapshotIsPublishedWhenNothingNewerIsStored(t *testing.T) {
	users := []define.AllowListUser{{Mail: "kept@example.com"}}
	for name, prepare := range map[string]func(t *testing.T, app *App, base time.Time){
		"missing": func(*testing.T, *App, time.Time) {},
		"older": func(t *testing.T, app *App, base time.Time) {
			other := &App{redisClient: app.redisClient, redisSnapshotMode: "v2", redisSnapshots: app.redisSnapshots, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: app.log}
			publishMigrationSource(t, other, base.Add(-time.Minute), []define.AllowListUser{{Mail: "earlier@example.com"}})
		},
		"unreadable": func(t *testing.T, app *App, _ time.Time) {
			require.NoError(t, app.redisClient.Set(t.Context(), "warden:{migration-test}:snapshot:v2", "{", 0).Err())
		},
	} {
		t.Run(name, func(t *testing.T) {
			app, _ := migrationFixture(t, "v2")
			base := time.Now().Add(-time.Minute).UTC()
			prepare(t, app, base)
			loadWithoutPublishing(t, app, base, users)

			writer, release := app.acquireRedisRefreshWriter()
			require.True(t, writer)
			app.keepRejectedRefresh(writer, 1)
			release()

			require.Equal(t, []string{"kept@example.com"}, storedSnapshotMails(t, app))
			require.NotEmpty(t, app.snapshots.Load().RedisRevision)
		})
	}
}

// A Redis failure while checking the stored publication is reported and
// nothing is written.
func TestV2RetainedSnapshotCheckFailureIsReported(t *testing.T) {
	app, mr := migrationFixture(t, "v2")
	loadWithoutPublishing(t, app, time.Now().Add(-time.Minute).UTC(), []define.AllowListUser{{Mail: "kept@example.com"}})
	app.snapshots.RecordRefreshFailure(reasonAllRecordsRejected)
	writer, release := app.acquireRedisRefreshWriter()
	require.True(t, writer)
	defer release()
	mr.SetError("LOADING Redis is loading the dataset in memory")
	defer mr.SetError("")
	require.Error(t, app.publishRedisSnapshots(app.userCache.Get()))
	require.Empty(t, app.snapshots.Load().RedisRevision)
}
