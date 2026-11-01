package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/soulteary/warden/internal/loader"
)

func refreshBenchmarkUsers(count int, scope string, generation int) []define.AllowListUser {
	rows := make([]define.AllowListUser, count)
	for i := range rows {
		rows[i] = define.AllowListUser{UserID: fmt.Sprintf("member-%05d", count-i), Mail: fmt.Sprintf("member%d@example.com", i), Status: "active", Name: fmt.Sprintf("generation-%d", generation)}
		switch scope {
		case "empty":
			rows[i].Scope = []string{}
		case "short":
			rows[i].Scope = []string{"read", "write"}
		case "long":
			rows[i].Scope = []string{strings.Repeat("scope", 64), "read", "write"}
		}
	}
	return rows
}

func benchmarkLocalApp(b *testing.B) *App {
	b.Helper()
	rules, err := loader.NewRulesLoader(nil, loader.ModeOnlyLocal)
	if err != nil {
		b.Fatal(err)
	}
	return &App{rulesLoader: rules, appMode: "ONLY_LOCAL", userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Output: io.Discard})}
}

type refreshFixture struct {
	app        *App
	snapshot   *Snapshot
	loaded     *loader.LoadResult
	file       string
	users      []define.AllowListUser
	raw, other []byte
	count      int
}

func newRefreshFixture(b *testing.B, count int, scope string) *refreshFixture {
	b.Helper()
	f := &refreshFixture{app: benchmarkLocalApp(b), users: refreshBenchmarkUsers(count, scope, 0), count: count, file: filepath.Join(b.TempDir(), "rules.json")}
	var err error
	f.raw, err = json.Marshal(f.users)
	if err != nil {
		b.Fatal(err)
	}
	f.other, err = json.Marshal(refreshBenchmarkUsers(count, scope, 1))
	if err != nil {
		b.Fatal(err)
	}
	if err = os.WriteFile(f.file, f.raw, 0o600); err != nil {
		b.Fatal(err)
	}
	loaded := f.app.rulesLoader.LoadWithResult(context.Background(), f.file, "", "", "")
	f.loaded = &loaded
	if f.loaded.Err != nil || len(f.loaded.Users) != count {
		b.Fatalf("invalid fixture: %v", f.loaded.Err)
	}
	prepared, err := cache.PrepareRules(f.loaded.Users, identity.Options{})
	if err != nil {
		b.Fatal(err)
	}
	if prepared.ReadView().Len() != count {
		b.Fatal("fixture lost records")
	}
	f.snapshot = f.app.sourceSnapshot(f.loaded, prepared.ReadView())
	return f
}

func benchmarkLoadStage(b *testing.B, f *refreshFixture) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(f.raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := f.app.rulesLoader.LoadWithResult(context.Background(), f.file, "", "", "")
		if result.Err != nil || len(result.Users) != f.count {
			b.Fatalf("load failed: %v", result.Err)
		}
		runtime.KeepAlive(result)
	}
}
func benchmarkPrepareStage(b *testing.B, f *refreshFixture) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := cache.PrepareRules(f.loaded.Users, identity.Options{})
		if err != nil {
			b.Fatal(err)
		}
		runtime.KeepAlive(result)
	}
}
func benchmarkFreezeStage(b *testing.B, f *refreshFixture) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runtime.KeepAlive(freezeSnapshot(f.snapshot))
	}
}

// Writes are excluded; backgroundTask reads/parses the real file and exercises
// its changed/unchanged branch. With store=nil, Redis is explicitly disabled.
func benchmarkRefreshStage(b *testing.B, f *refreshFixture, changed bool, store *cache.RedisSnapshotStore) {
	b.Helper()
	app := benchmarkLocalApp(b)
	if store != nil {
		app.redisSnapshotMode = "v2"
		app.redisSnapshots = store
	}
	if err := os.WriteFile(f.file, f.raw, 0o600); err != nil {
		b.Fatal(err)
	}
	app.backgroundTask(f.file, "")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		payload := f.raw
		if changed && i%2 == 0 {
			payload = f.other
		}
		if err := os.WriteFile(f.file, payload, 0o600); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		app.backgroundTask(f.file, "")
	}
	b.StopTimer()
	payload := f.raw
	if changed && (b.N-1)%2 == 0 {
		payload = f.other
	}
	checkRefreshedFixture(b, app, f, payload, store)
	runtime.KeepAlive(app)
}

func checkRefreshedFixture(b *testing.B, app *App, f *refreshFixture, payload []byte, store *cache.RedisSnapshotStore) {
	b.Helper()
	snap := app.snapshots.Load()
	if snap.EffectiveCount != f.count || snap.EffectiveVersion != app.userCache.GetHash() {
		b.Fatal("refresh lost effective records/version")
	}
	var rows []define.AllowListUser
	if err := json.Unmarshal(payload, &rows); err != nil {
		b.Fatal(err)
	}
	if snap.Version != cache.HashUserList(rows) {
		b.Fatal("refresh used wrong input generation")
	}
	expected, err := cache.PrepareRules(rows, identity.Options{})
	if err != nil {
		b.Fatal(err)
	}
	if !reflect.DeepEqual(expected.ReadView().Get(), snap.view.Get()) {
		b.Fatal("refresh changed list order or records")
	}
	if store != nil {
		envelope, err := store.Read(context.Background())
		if err != nil || envelope == nil {
			b.Fatalf("published snapshot missing: %v", err)
		}
		if envelope.InputVersion != snap.Version || envelope.EffectiveVersion != snap.EffectiveVersion {
			b.Fatal("Redis differs from accepted generation")
		}
	}
}

func BenchmarkRefreshStages(b *testing.B) {
	for _, count := range []int{0, 1, 1000, 10000} {
		for _, scope := range []string{"nil", "empty", "short", "long"} {
			b.Run(fmt.Sprintf("rows=%d/scope=%s", count, scope), func(b *testing.B) {
				f := newRefreshFixture(b, count, scope)
				b.Run("Load", func(b *testing.B) { benchmarkLoadStage(b, f) })
				b.Run("Prepare", func(b *testing.B) { benchmarkPrepareStage(b, f) })
				b.Run("Freeze", func(b *testing.B) { benchmarkFreezeStage(b, f) })
				b.Run("RefreshUnchanged", func(b *testing.B) { benchmarkRefreshStage(b, f, false, nil) })
				if count > 0 {
					b.Run("RefreshChanged", func(b *testing.B) { benchmarkRefreshStage(b, f, true, nil) })
				}
			})
		}
	}
}

func benchmarkRedisStore(b *testing.B) *cache.RedisSnapshotStore {
	b.Helper()
	addr := os.Getenv("WARDEN_REDIS_INTEGRATION_ADDR")
	if addr == "" {
		b.Skip("set WARDEN_REDIS_INTEGRATION_ADDR for disposable Redis 8")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, ContextTimeoutEnabled: true})
	b.Cleanup(func() {
		if err := client.Close(); err != nil {
			b.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		b.Fatal(err)
	}
	if !strings.Contains(info, "redis_version:8.") {
		b.Fatal("Redis 8 required")
	}
	namespace := fmt.Sprintf("bench-%d-%d", os.Getpid(), time.Now().UnixNano())
	store, err := cache.NewRedisSnapshotStore(client, namespace, strings.Repeat("a", 64))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		cleanup, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		if cleanupErr := client.Del(cleanup, "warden:{"+namespace+"}:snapshot:v2", "warden:{"+namespace+"}:snapshot:v2:lease").Err(); cleanupErr != nil {
			b.Error(cleanupErr)
		}
	})
	return store
}

func benchmarkPublishEnvelope(b *testing.B, store *cache.RedisSnapshotStore, envelope *cache.SnapshotEnvelope) {
	b.Helper()
	lease, err := store.Acquire(context.Background(), time.Minute)
	if err != nil {
		b.Fatal(err)
	}
	if lease == nil {
		b.Fatal("unexpected busy private lease")
	}
	_, publishErr := lease.Publish(context.Background(), envelope)
	releaseErr := lease.Release(context.Background())
	if publishErr != nil {
		b.Fatal(publishErr)
	}
	if releaseErr != nil {
		b.Fatal(releaseErr)
	}
}
func benchmarkRestoreStage(b *testing.B, store *cache.RedisSnapshotStore, envelope *cache.SnapshotEnvelope) {
	b.Helper()
	app := &App{redisSnapshotMode: "v2", redisSnapshots: store, userCache: cache.NewSafeUserCache(), snapshots: newSnapshotStore(), log: loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Output: io.Discard})}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !app.restoreVersionedSnapshot(nil) {
			b.Fatal("restore failed")
		}
	}
	b.StopTimer()
	result := app.snapshots.Load()
	if result.EffectiveCount != envelope.InputCount || result.Version != envelope.InputVersion || result.EffectiveVersion != envelope.EffectiveVersion || !result.SourceSucceededAt.Equal(envelope.SourceSucceededAt) {
		b.Fatal("restore changed data/version/source time")
	}
	if !reflect.DeepEqual(envelope.Users, result.view.Get()) {
		b.Fatal("restore changed list order or records")
	}
	runtime.KeepAlive(app)
}

// Requires an explicitly selected disposable Redis 8. No FLUSHDB/FLUSHALL.
func BenchmarkRedisSnapshotStages(b *testing.B) {
	for _, count := range []int{0, 1, 1000, 10000} {
		b.Run(fmt.Sprintf("rows=%d", count), func(b *testing.B) {
			store := benchmarkRedisStore(b)
			f := newRefreshFixture(b, count, "short")
			envelope := &cache.SnapshotEnvelope{Users: f.snapshot.view.Get(), EffectiveVersion: f.snapshot.EffectiveVersion, InputVersion: f.snapshot.Version, InputCount: count, Origin: string(loader.SourceLocal), SourceSucceededAt: time.Now().Add(-time.Minute).UTC()}
			benchmarkPublishEnvelope(b, store, envelope)
			b.Run("PublishLease", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchmarkPublishEnvelope(b, store, envelope)
				}
			})
			b.Run("ReadDecodeValidate", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					result, err := store.Read(context.Background())
					if err != nil || result == nil {
						b.Fatalf("read failed: %v", err)
					}
					runtime.KeepAlive(result)
				}
			})
			b.Run("Restore", func(b *testing.B) { benchmarkRestoreStage(b, store, envelope) })
			b.Run("RefreshUnchanged", func(b *testing.B) { benchmarkRefreshStage(b, f, false, store) })
			if count > 0 {
				b.Run("RefreshChanged", func(b *testing.B) { benchmarkRefreshStage(b, f, true, store) })
			}
		})
	}
}

// Pins frozen provenance copies of one effective view. It measures post-GC
// residency, not peak memory, RSS, or independently built index generations.
func BenchmarkSnapshotFreezeRetention(b *testing.B) {
	for _, generations := range []int{1, 16} {
		b.Run(fmt.Sprintf("rows=1000/copies=%d", generations), func(b *testing.B) {
			f := newRefreshFixture(b, 1000, "short")
			var live float64
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				b.StartTimer()
				pinned := make([]*Snapshot, 0, generations)
				for n := 0; n < generations; n++ {
					pinned = append(pinned, freezeSnapshot(f.snapshot))
				}
				b.StopTimer()
				runtime.GC()
				runtime.ReadMemStats(&after)
				live += float64(after.HeapAlloc) - float64(before.HeapAlloc)
				if len(pinned[0].Users) != 1000 || len(pinned[generations-1].Users) != 1000 {
					b.Fatal("lost pinned rows")
				}
				runtime.KeepAlive(pinned)
				b.StartTimer()
			}
			b.ReportMetric(live/float64(b.N), "live-B/pinned-set")
		})
	}
}
