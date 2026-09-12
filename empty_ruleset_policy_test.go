package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
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
