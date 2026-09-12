package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	health "github.com/soulteary/health-kit/v2"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Rule sets used by the policy tests.
//
// goodRuleSet is ordinary valid data. rejectedRuleSet is the case the policy exists for: a
// load that succeeds and returns records, none of which survive per-record format
// validation — what an upstream format change looks like from inside warden. Identity
// validation passes on it (distinct identifiers, no missing user_id), so it reaches the
// cache and is emptied there rather than being rejected as a conflict.
const (
	goodRuleSet = `[
		{"phone":"13800138000","mail":"a@example.com","status":"active"},
		{"phone":"13900139000","mail":"c@example.com","status":"active"}
	]`
	rejectedRuleSet = `[
		{"phone":"13-800-138-000","mail":"a(at)example.com","status":"active"},
		{"phone":"13/900/139/000","mail":"c(at)example.com","status":"active"}
	]`
	rejectedConflictingRuleSet = `[
		{"phone":"13-800-138-000","mail":"a(at)example.com","status":"active"},
		{"phone":"13-800-138-000","mail":"c(at)example.com","status":"active"}
	]`
	emptyRuleSet = `[]`
)

// policyTestApp builds an app that has already loaded goodRuleSet (so there IS a last known
// good rule set to lose), writable data file, elected as the shared-cache writer, capturing
// every payload handed to Redis.
func policyTestApp(t *testing.T, policy string) (app *App, dataFile string, published *[][]define.AllowListUser) {
	t.Helper()

	dataFile = filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(goodRuleSet), 0o600))

	app = NewApp(&cmd.Config{
		Port:               "8081",
		RedisEnabled:       false,
		Mode:               "ONLY_LOCAL",
		APIKey:             "test-key",
		TaskInterval:       60,
		DataFile:           dataFile,
		EmptyRulesetPolicy: policy,
	})
	require.NotNil(t, app)

	captured := make([][]define.AllowListUser, 0, 4)
	app.redisUserCache = &cache.RedisUserCache{}
	app.redisRefreshLocker = &stubRefreshLocker{locked: true}
	app.publishToRedis = func(users []define.AllowListUser) error {
		captured = append(captured, append([]define.AllowListUser(nil), users...))
		return nil
	}
	return app, dataFile, &captured
}

// writeRuleSet replaces the data file so the next refresh sees a changed rule set.
func writeRuleSet(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// TestApp_backgroundTask_ConsistencyFirst_AppliesEmptySet pins the default (and historical)
// behavior: when every loaded record is rejected, the empty effective set is applied locally
// and published, so the shared cache keeps describing exactly what this replica serves.
func TestApp_backgroundTask_ConsistencyFirst_AppliesEmptySet(t *testing.T) {
	app, dataFile, published := policyTestApp(t, "consistency-first")
	require.False(t, app.emptyRulesetPolicy.KeepsLastKnownGood())
	require.Len(t, app.userCache.Get(), 2, "初始数据应已加载")

	writeRuleSet(t, dataFile, rejectedRuleSet)
	app.backgroundTask(dataFile, "")

	assert.Empty(t, app.userCache.Get(), "一致性优先：生效集合必须变为空")
	require.Len(t, *published, 1, "一致性优先必须向共享缓存发布一次")
	assert.Empty(t, (*published)[0], "一致性优先必须发布空集，使副本之间保持一致")

	failures, reason := app.snapshots.RefreshFailure()
	assert.Zero(t, failures, "一致性优先把这次刷新视为成功")
	assert.Empty(t, reason)

	snap := app.snapshots.Load()
	require.NotNil(t, snap)
	assert.Equal(t, cache.HashUserList(mustParse(t, rejectedRuleSet)), snap.Version,
		"快照必须推进到新加载的版本")
}

// TestApp_backgroundTask_AvailabilityFirst_KeepsLastKnownGood is the new behavior: the same
// load is treated as a failed refresh, the previously served rule set stays in memory, and
// it is republished so the shared copy does not quietly expire while the upstream is broken.
func TestApp_backgroundTask_AvailabilityFirst_KeepsLastKnownGood(t *testing.T) {
	app, dataFile, published := policyTestApp(t, "availability-first")
	require.True(t, app.emptyRulesetPolicy.KeepsLastKnownGood())

	before := app.userCache.Get()
	require.Len(t, before, 2, "初始数据应已加载")
	snapBefore := app.snapshots.Load()
	require.NotNil(t, snapBefore)

	writeRuleSet(t, dataFile, rejectedRuleSet)
	app.backgroundTask(dataFile, "")

	assert.Equal(t, before, app.userCache.Get(),
		"可用性优先：内存中必须保留上一次有效的规则集")

	require.Len(t, *published, 1,
		"必须刷新共享缓存，否则上一次有效的规则集会随 TTL 过期")
	assert.Equal(t, before, (*published)[0],
		"发布到共享缓存的必须是保留下来的有效集合，而不是空集")

	failures, reason := app.snapshots.RefreshFailure()
	assert.EqualValues(t, 1, failures, "必须记为一次刷新失败")
	assert.Equal(t, reasonAllRecordsRejected, reason)

	assert.Equal(t, snapBefore.Version, app.snapshots.Load().Version,
		"快照版本必须停留在有效数据上，使下个周期继续重试")
	assert.True(t, app.checkDataChanged(cache.HashUserList(mustParse(t, rejectedRuleSet))),
		"损坏的数据不得被记录为已应用，否则下个周期会被当作未变化而跳过")
}

// TestApp_backgroundTask_AvailabilityFirst_RetriesAndRecovers walks the full incident: the
// upstream stays broken for another cycle (failures keep climbing, data keeps being served)
// and then heals, at which point the refresh must apply normally again.
func TestApp_backgroundTask_AvailabilityFirst_RetriesAndRecovers(t *testing.T) {
	app, dataFile, published := policyTestApp(t, "availability-first")
	before := app.userCache.Get()
	require.Len(t, before, 2)

	writeRuleSet(t, dataFile, rejectedRuleSet)
	app.backgroundTask(dataFile, "")
	app.backgroundTask(dataFile, "")

	failures, reason := app.snapshots.RefreshFailure()
	assert.EqualValues(t, 2, failures, "每个周期都必须重试并累计失败次数")
	assert.Equal(t, reasonAllRecordsRejected, reason)
	assert.Equal(t, before, app.userCache.Get(), "期间必须持续提供上一次有效的规则集")
	assert.Len(t, *published, 2, "每个周期都要续期共享缓存")

	// Upstream recovers with a different valid set.
	const healed = `[{"phone":"13700137000","mail":"d@example.com","status":"active"}]`
	writeRuleSet(t, dataFile, healed)
	app.backgroundTask(dataFile, "")

	require.Len(t, app.userCache.Get(), 1, "上游恢复后必须应用新数据")
	assert.Equal(t, "13700137000", app.userCache.Get()[0].Phone)
	failures, _ = app.snapshots.RefreshFailure()
	assert.Zero(t, failures, "一次成功刷新必须清零失败计数")
	require.Len(t, *published, 3)
	assert.Equal(t, app.userCache.Get(), (*published)[2])
}

// TestApp_backgroundTask_AvailabilityFirst_IdentityFailureTakesPrecedence covers a set
// that fails both validation layers: every record has an invalid format, and the records
// also share an identity. The collection-wide identity error is the root cause and must
// not be hidden behind the availability policy's all_records_rejected classification.
func TestApp_backgroundTask_AvailabilityFirst_IdentityFailureTakesPrecedence(t *testing.T) {
	app, dataFile, published := policyTestApp(t, "availability-first")
	before := app.userCache.Get()
	require.Len(t, before, 2)

	writeRuleSet(t, dataFile, rejectedConflictingRuleSet)
	app.backgroundTask(dataFile, "")

	assert.Equal(t, before, app.userCache.Get(),
		"身份冲突必须保留上一次有效集合，且不能先按全量格式拒绝分类")
	assert.Empty(t, *published, "身份冲突不是可续期的全量格式拒绝，不应发布共享缓存")

	failures, reason := app.snapshots.RefreshFailure()
	assert.EqualValues(t, 1, failures)
	assert.Equal(t, reasonIdentityConflict, reason,
		"同时存在身份冲突与格式错误时，身份完整性错误必须优先")
}

// TestApp_backgroundTask_GenuinelyEmptyLoadAlwaysPropagates is the safety property that
// keeps availability-first from becoming a security hole: a load that really returns zero
// records is not ambiguous, so revoking every user must still work under BOTH policies.
func TestApp_backgroundTask_GenuinelyEmptyLoadAlwaysPropagates(t *testing.T) {
	for _, policy := range []string{"consistency-first", "availability-first"} {
		t.Run(policy, func(t *testing.T) {
			app, dataFile, published := policyTestApp(t, policy)
			require.Len(t, app.userCache.Get(), 2)

			writeRuleSet(t, dataFile, emptyRuleSet)
			app.backgroundTask(dataFile, "")

			assert.Empty(t, app.userCache.Get(),
				"合法的全员撤权必须在两种策略下都生效")
			require.Len(t, *published, 1)
			assert.Empty(t, (*published)[0], "空集必须传播到共享缓存")

			failures, _ := app.snapshots.RefreshFailure()
			assert.Zero(t, failures, "真正的空数据源不是刷新失败")
		})
	}
}

// TestNewApp_EmptyRulesetPolicyDefaults covers the values NewApp can receive: unset means
// the historical behavior, and an unrecognized value (which ValidateConfig rejects at
// startup) must still leave a usable policy behind for direct NewApp callers.
func TestNewApp_EmptyRulesetPolicyDefaults(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		expect string
		keeps  bool
	}{
		{"未设置时使用一致性优先", "", "consistency-first", false},
		{"显式一致性优先", "consistency-first", "consistency-first", false},
		{"显式可用性优先", "availability-first", "availability-first", true},
		{"简写同样生效", "availability", "availability-first", true},
		{"非法值回退到一致性优先", "no-such-policy", "consistency-first", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := NewApp(&cmd.Config{
				Port:               "8081",
				Mode:               "ONLY_LOCAL",
				TaskInterval:       60,
				DataFile:           filepath.Join(t.TempDir(), "missing.json"),
				EmptyRulesetPolicy: tc.value,
			})
			require.NotNil(t, app)
			assert.Equal(t, tc.expect, app.emptyRulesetPolicy.String())
			assert.Equal(t, tc.keeps, app.emptyRulesetPolicy.KeepsLastKnownGood())
		})
	}
}

// mustParse decodes a rule-set literal the way the loader would, so a test can compute the
// version hash of the raw loaded set independently of the code under test.
func mustParse(t *testing.T, raw string) []define.AllowListUser {
	t.Helper()
	var users []define.AllowListUser
	require.NoError(t, json.Unmarshal([]byte(raw), &users))
	return users
}

// onlyLocalRestartApp simulates a replica restarting in ONLY_LOCAL mode: the in-memory cache
// starts empty as it does in a fresh process, the local file holds localContent, and shared
// is what a healthy replica last published to the shared cache (sharedErr makes that read
// fail instead).
//
// ONLY_LOCAL returns before loadInitialData's Redis read, which makes it the one startup path
// where an all-rejected local file can reach applyUsers and publish the resulting empty set
// over the shared last known good data.
//
// The shared cache is deliberately the zero value: loadInitialData seeds it by calling
// redisUserCache.Set directly rather than through the publishToRedis seam, and that call
// panics on a zero-value cache. A write is therefore observable — tests that must prove
// nothing was published wrap the call in require.NotPanics, and tests that must not write at
// all elect no writer. If cache-kit ever makes a zero-value Set return an error instead of
// panicking, those NotPanics assertions stop detecting the write and need a real fake.
func onlyLocalRestartApp(t *testing.T, policy, localContent string, shared []define.AllowListUser, sharedErr error, electWriter bool) (app *App, dataFile string) {
	t.Helper()

	dataFile = filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(goodRuleSet), 0o600))

	app = NewApp(&cmd.Config{
		Port:               "8081",
		RedisEnabled:       false,
		Mode:               "ONLY_LOCAL",
		APIKey:             "test-key",
		TaskInterval:       60,
		DataFile:           dataFile,
		EmptyRulesetPolicy: policy,
	})
	require.NotNil(t, app)

	app.redisUserCache = &cache.RedisUserCache{}
	app.redisRefreshLocker = &stubRefreshLocker{locked: electWriter}
	app.loadFromRedis = func() ([]define.AllowListUser, error) { return shared, sharedErr }

	// A restarted process holds nothing in memory; only the shared cache survives.
	app.userCache.Set(nil)
	app.snapshots = newSnapshotStore()
	require.Empty(t, app.userCache.Get(), "重启的副本内存中不应有数据")
	writeRuleSet(t, dataFile, localContent)
	return app, dataFile
}

// redisFirstRestartApp simulates a fresh non-ONLY_LOCAL replica while keeping Redis calls
// controllable. NewApp first establishes a working loader from a valid local file; the
// helper then clears process-local state to the same empty state as a restart and installs
// a zero-value shared cache whose accidental Set would panic. loadFromRedis supplies the
// scripted bootstrap/retry behavior without requiring a live Redis server.
//
// Two inputs are fixed rather than parameters, because varying either would leave the
// helper testing something else entirely: the policy is availability-first, since the
// bootstrap retry this helper scripts only exists under it (consistency-first applies and
// publishes whatever it loaded, so there is no second Redis read to script and no shared set
// to protect), and the local file holds rejectedRuleSet, since the all-rejected load is the
// ambiguity that sends startup down that retry path in the first place. Only mode and the
// scripted Redis behavior differ between callers.
func redisFirstRestartApp(t *testing.T, mode string, loadFromRedis func() ([]define.AllowListUser, error)) (app *App, dataFile string) {
	t.Helper()

	dataFile = filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(goodRuleSet), 0o600))

	app = NewApp(&cmd.Config{
		Port:               "8081",
		RedisEnabled:       false,
		Mode:               mode,
		APIKey:             "test-key",
		TaskInterval:       60,
		DataFile:           dataFile,
		EmptyRulesetPolicy: "availability-first",
	})
	require.NotNil(t, app)

	app.redisUserCache = &cache.RedisUserCache{}
	app.redisRefreshLocker = &stubRefreshLocker{locked: true}
	app.loadFromRedis = loadFromRedis
	app.userCache.Set(nil)
	app.snapshots = newSnapshotStore()
	require.Empty(t, app.userCache.Get(), "重启的副本内存中不应有数据")

	writeRuleSet(t, dataFile, rejectedRuleSet)
	return app, dataFile
}

// TestApp_loadInitialData_AvailabilityFirst_ONLY_LOCAL_KeepsSharedRuleSet pins the restart
// guarantee availability-first makes.
//
// ONLY_LOCAL returns before loadInitialData's Redis read, so an all-rejected local file used
// to be applied directly and the resulting empty cache written straight back to the shared
// cache. That both started this replica empty and destroyed the last known good set every
// other replica — and every later restart — bootstraps from, defeating the policy in exactly
// the scenario it exists to cover.
func TestApp_loadInitialData_AvailabilityFirst_ONLY_LOCAL_KeepsSharedRuleSet(t *testing.T) {
	shared := mustParse(t, goodRuleSet)
	app, dataFile := onlyLocalRestartApp(t, "availability-first", rejectedRuleSet, shared, nil, true)
	require.True(t, app.emptyRulesetPolicy.KeepsLastKnownGood())

	var loadErr error
	require.NotPanics(t, func() { loadErr = app.loadInitialData(dataFile, "") },
		"可用性优先绝不能把空集写回共享缓存：那会抹掉其他副本赖以启动的最后一份有效数据")
	require.NoError(t, loadErr)

	assert.Len(t, app.userCache.Get(), 2, "可用性优先：应从共享缓存引导出上一次有效的规则集，而不是启动为空")
}

// TestApp_loadInitialData_AvailabilityFirst_ONLY_LOCAL_PreservesUnreadableSharedSet pins the
// preserve half on its own: when the shared cache cannot be read, the empty result still must
// not be published. A failed read says nothing about whether the good data is still there.
func TestApp_loadInitialData_AvailabilityFirst_ONLY_LOCAL_PreservesUnreadableSharedSet(t *testing.T) {
	app, dataFile := onlyLocalRestartApp(t, "availability-first", rejectedRuleSet, nil, assert.AnError, true)
	require.True(t, app.emptyRulesetPolicy.KeepsLastKnownGood())

	var loadErr error
	require.NotPanics(t, func() { loadErr = app.loadInitialData(dataFile, "") },
		"读不到共享缓存时更不能写空集：那份数据可能仍在，只是这次读取失败")
	require.NoError(t, loadErr)

	assert.Empty(t, app.userCache.Get(), "没有可引导的数据，本副本为空——但共享缓存未被破坏")
}

// TestApp_loadInitialData_AvailabilityFirst_RetriesRedisAfterBootstrapError pins the
// non-ONLY_LOCAL restart race: the first Redis read fails, the fallback source returns a
// non-empty set that format validation would erase, and Redis recovers after writer
// election. Startup must retry and bootstrap from the shared good set rather than overwrite
// it with the empty fallback result.
func TestApp_loadInitialData_AvailabilityFirst_RetriesRedisAfterBootstrapError(t *testing.T) {
	shared := mustParse(t, goodRuleSet)
	reads := 0
	app, dataFile := redisFirstRestartApp(t, "DEFAULT", func() ([]define.AllowListUser, error) {
		reads++
		if reads == 1 {
			return nil, assert.AnError
		}
		return shared, nil
	})

	var loadErr error
	require.NotPanics(t, func() { loadErr = app.loadInitialData(dataFile, "") },
		"Redis 引导读取暂时失败后，不能把全量拒绝产生的空集写回共享缓存")
	require.NoError(t, loadErr)

	assert.Equal(t, 2, reads, "获得写入者锁后应重试一次共享缓存读取")
	assert.Len(t, app.userCache.Get(), 2, "Redis 恢复后应从共享的最后有效集合完成启动")
	snap := app.snapshots.Load()
	require.NotNil(t, snap)
	assert.Equal(t, loader.SourceRedis, snap.Source, "共享缓存引导必须留下明确的快照来源")
	assert.False(t, snap.LoadedAt.IsZero(), "共享缓存引导必须建立可计算新鲜度的时间基线")
	assert.Equal(t, "redis_bootstrap", snap.DegradedReason)
}

// TestApp_loadInitialData_RedisBootstrapProvidesStrictSnapshotFreshness verifies that a
// strict remote replica serving shared last-known-good data is degraded but usable. Redis
// adoption must establish source/time provenance so snapshot_freshness does not report
// snapshot_unknown immediately after a successful restart.
func TestApp_loadInitialData_RedisBootstrapProvidesStrictSnapshotFreshness(t *testing.T) {
	shared := mustParse(t, goodRuleSet)
	app, dataFile := redisFirstRestartApp(t, "ONLY_REMOTE", func() ([]define.AllowListUser, error) {
		return shared, nil
	})

	require.NoError(t, app.loadInitialData(dataFile, ""))
	require.True(t, app.hasKnownGoodSnapshot())
	assert.Equal(t, loader.SourceRedis, app.snapshots.Load().Source)

	aggregator := setupHealthChecker(nil, app.userCache, app.snapshots, time.Minute,
		"ONLY_REMOTE", "development", false, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusHealthy, result.Checks["snapshot_freshness"].Status,
		"Redis 引导出的有效集合应从启动时开始计算新鲜度，而不是立即报告来源未知")
	assert.NotEqual(t, health.StatusUnhealthy, result.Status,
		"严格模式成功从 Redis 恢复后不应立即返回 503")
}

// TestApp_loadInitialData_AvailabilityFirst_PreservesRedisAfterRepeatedReadErrors covers
// the same restart race when Redis remains unreadable. The replica has no data to serve,
// but a failed read is not evidence that the shared set is gone, so startup must leave it
// untouched instead of publishing an empty replacement.
func TestApp_loadInitialData_AvailabilityFirst_PreservesRedisAfterRepeatedReadErrors(t *testing.T) {
	reads := 0
	app, dataFile := redisFirstRestartApp(t, "DEFAULT", func() ([]define.AllowListUser, error) {
		reads++
		return nil, assert.AnError
	})

	var loadErr error
	require.NotPanics(t, func() { loadErr = app.loadInitialData(dataFile, "") },
		"共享缓存持续不可读时仍不得写入空集，因为最后有效数据可能仍然存在")
	require.NoError(t, loadErr)

	assert.Equal(t, 2, reads, "启动读取与锁内重试都应发生")
	assert.Empty(t, app.userCache.Get(), "无可引导数据时本副本保持为空，但共享集合不得被覆盖")
	assert.False(t, app.hasKnownGoodSnapshot(), "两次读取失败后不得伪造已知有效快照")

	published := make([][]define.AllowListUser, 0, 1)
	app.publishToRedis = func(users []define.AllowListUser) error {
		published = append(published, append([]define.AllowListUser(nil), users...))
		return nil
	}
	app.backgroundTask(dataFile, "")

	assert.Equal(t, 3, reads, "首次后台刷新应再次尝试恢复共享的最后有效集合")
	assert.Empty(t, published,
		"仍无已知有效快照时绝不能续期进程的零值空缓存，否则会覆盖 Redis 中可能已恢复的数据")
}

// TestApp_backgroundTask_AvailabilityFirst_RecoversRedisAfterStartupMisses covers the
// other half of the startup-to-refresh transition: both startup reads and the first
// scheduled retry fail, Redis then recovers while the source remains all-invalid. The next
// refresh must adopt the shared set before renewal without resetting the source-failure run.
func TestApp_backgroundTask_AvailabilityFirst_RecoversRedisAfterStartupMisses(t *testing.T) {
	shared := mustParse(t, goodRuleSet)
	reads := 0
	app, dataFile := redisFirstRestartApp(t, "DEFAULT", func() ([]define.AllowListUser, error) {
		reads++
		if reads <= 3 {
			return nil, assert.AnError
		}
		return shared, nil
	})

	require.NoError(t, app.loadInitialData(dataFile, ""))
	require.Empty(t, app.userCache.Get())
	require.False(t, app.hasKnownGoodSnapshot())

	published := make([][]define.AllowListUser, 0, 1)
	app.publishToRedis = func(users []define.AllowListUser) error {
		published = append(published, append([]define.AllowListUser(nil), users...))
		return nil
	}
	app.backgroundTask(dataFile, "")
	require.Empty(t, app.userCache.Get(), "首次后台重读仍失败时不得伪造有效数据")
	require.Empty(t, published, "没有已知有效集合时必须跳过共享缓存写入")
	firstFailures, _ := app.snapshots.RefreshFailure()
	require.EqualValues(t, 1, firstFailures)

	app.backgroundTask(dataFile, "")

	assert.Equal(t, 4, reads, "每次后台刷新都应重读尚未成功引导的 Redis")
	assert.Len(t, app.userCache.Get(), 2, "Redis 恢复后应采用共享的最后有效集合")
	require.Len(t, published, 1, "恢复出的已知有效集合可以安全续期")
	assert.Equal(t, app.userCache.Get(), published[0])
	assert.True(t, app.hasKnownGoodSnapshot())
	assert.Equal(t, loader.SourceRedis, app.snapshots.Load().Source)
	failures, reason := app.snapshots.RefreshFailure()
	assert.EqualValues(t, 2, failures, "Redis 恢复只补回基线，不能清零仍在连续失败的源刷新")
	assert.Equal(t, reasonAllRecordsRejected, reason)
}

// TestApp_loadInitialData_ConsistencyFirst_ONLY_LOCAL_AppliesEmptySet pins that the default
// keeps its historical startup behavior: the empty effective set is applied rather than
// kept, and the shared cache is never consulted for a last known good set to fall back on.
func TestApp_loadInitialData_ConsistencyFirst_ONLY_LOCAL_AppliesEmptySet(t *testing.T) {
	shared := mustParse(t, goodRuleSet)
	app, dataFile := onlyLocalRestartApp(t, "consistency-first", rejectedRuleSet, shared, nil, false)
	require.False(t, app.emptyRulesetPolicy.KeepsLastKnownGood())

	require.NoError(t, app.loadInitialData(dataFile, ""))

	assert.Empty(t, app.userCache.Get(), "一致性优先：生效集合必须为空，且不得回退到共享缓存里的旧数据")
}

// TestApp_loadInitialData_ONLY_LOCAL_GenuinelyEmptyIsPolicyIndependent pins that a local file
// genuinely holding zero records never reaches the ambiguity guard — len(localUsers) > 0 gates
// that branch — so neither policy bootstraps stale data from the shared cache in that case.
// Without this, moving the guard above the length check would silently make "revoke everyone"
// impossible under availability-first.
func TestApp_loadInitialData_ONLY_LOCAL_GenuinelyEmptyIsPolicyIndependent(t *testing.T) {
	for _, policy := range []string{"consistency-first", "availability-first"} {
		t.Run(policy, func(t *testing.T) {
			shared := mustParse(t, goodRuleSet)
			app, dataFile := onlyLocalRestartApp(t, policy, emptyRuleSet, shared, nil, false)

			require.NoError(t, app.loadInitialData(dataFile, ""))

			assert.Empty(t, app.userCache.Get(), "真正的空数据源不属于歧义场景，不会被当作「全部被拒」而去引导旧数据")
		})
	}
}

// TestApp_loadInitialData_ONLY_LOCAL_GenuinelyEmptyPublishesImmediately pins the startup
// revocation guarantee end to end. A successful [] file is valid data, not a load failure:
// it must replace process memory, establish an empty-but-known-good snapshot, and clear the
// shared cache immediately rather than waiting for the first scheduled refresh.
func TestApp_loadInitialData_ONLY_LOCAL_GenuinelyEmptyPublishesImmediately(t *testing.T) {
	for _, policy := range []string{"consistency-first", "availability-first"} {
		t.Run(policy, func(t *testing.T) {
			shared := mustParse(t, goodRuleSet)
			app, dataFile := onlyLocalRestartApp(t, policy, emptyRuleSet, shared, nil, true)
			published := make([][]define.AllowListUser, 0, 1)
			app.publishToRedis = func(users []define.AllowListUser) error {
				published = append(published, append([]define.AllowListUser(nil), users...))
				return nil
			}

			require.NoError(t, app.loadInitialData(dataFile, ""))

			assert.Empty(t, app.userCache.Get())
			require.Len(t, published, 1, "启动时的合法全员撤权必须立即传播到共享缓存")
			assert.Empty(t, published[0])
			assert.True(t, app.hasKnownGoodSnapshot(), "合法空集与从未加载数据必须可区分")
			assert.Equal(t, loader.SourceLocal, app.snapshots.Load().Source)
		})
	}
}
