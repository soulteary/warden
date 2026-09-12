package cache

import (
	"fmt"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// acceptableCountPool holds one record per behaviour AcceptableCount has to reproduce:
// plain valid records, an email-only record, records rejected by each validation rule,
// and records that collide with an earlier entry on the phone key and on the mail key.
var acceptableCountPool = []define.AllowListUser{
	{Phone: "13800138000", Mail: "alice@example.com", UserID: "u1"},
	{Phone: "13900139000", Mail: "bob@example.com", UserID: "u2"},
	{Phone: "", Mail: "carol@example.com", UserID: "u3"},                // email-only: kept
	{Phone: "", Mail: "", UserID: "u4"},                                 // no identifier: dropped
	{Phone: "not-a-phone-xyz", Mail: "dave@example.com", UserID: "u5"},  // invalid phone: dropped
	{Phone: "13700137000", Mail: "not-an-email", UserID: "u6"},          // invalid mail: dropped
	{Phone: "13800138000", Mail: "eve@example.com", UserID: "u7"},       // duplicate phone key
	{Phone: "  13900139000  ", Mail: "frank@example.com", UserID: "u8"}, // duplicate after trim
	{Phone: "", Mail: "CAROL@Example.COM", UserID: "u9"},                // duplicate mail key after lowering
}

// cacheLen reports the effective size the shared cache reaches for users, which is the
// number AcceptableCount must predict without touching any cache.
func cacheLen(users []define.AllowListUser) int {
	c := NewSafeUserCache()
	c.Set(users)
	return c.Len()
}

// TestAcceptableCount_MatchesCacheLen pins AcceptableCount against the real cache for the
// individually interesting cases. AcceptableCount reimplements the filter chain rather
// than writing to a cache (a write would expose an empty rule set to concurrent readers),
// so every rule it mirrors needs a case here.
func TestAcceptableCount_MatchesCacheLen(t *testing.T) {
	cases := []struct {
		name  string
		users []define.AllowListUser
		want  int
	}{
		{"空输入", nil, 0},
		{"空切片", []define.AllowListUser{}, 0},
		{"全部有效", acceptableCountPool[:2], 2},
		{"仅邮箱用户保留", acceptableCountPool[2:3], 1},
		{"双标识为空被丢弃", acceptableCountPool[3:4], 0},
		{"手机号非法被丢弃", acceptableCountPool[4:5], 0},
		{"邮箱非法被丢弃", acceptableCountPool[5:6], 0},
		{"手机号重复折叠为一条", []define.AllowListUser{acceptableCountPool[0], acceptableCountPool[6]}, 1},
		{"去空格后重复折叠为一条", []define.AllowListUser{acceptableCountPool[1], acceptableCountPool[7]}, 1},
		{"邮箱大小写不同仍为重复", []define.AllowListUser{acceptableCountPool[2], acceptableCountPool[8]}, 1},
		{"全部记录被拒时为 0", acceptableCountPool[3:6], 0},
		{"完整样本集", acceptableCountPool, 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AcceptableCount(tc.users)
			assert.Equal(t, tc.want, got, "AcceptableCount 结果不符合预期")
			assert.Equal(t, cacheLen(tc.users), got,
				"AcceptableCount 必须与缓存实际生效条数一致")
		})
	}
}

// TestAcceptableCount_MatchesCacheLenAcrossSubsets is the drift guard: it walks every
// subset of the sample pool and requires AcceptableCount to agree with the cache on all
// of them. A future change to validateUser, normalizeUser, primaryKeyForUser or the
// cache-kit Set pipeline that AcceptableCount does not follow fails here.
//
// One limit is worth knowing: today define.AllowListUser.Normalize only fills defaults
// (user_id, status, scope) and never touches phone or mail, so the normalize step is
// count-neutral and no record in the pool can detect its removal. If Normalize ever starts
// rewriting the identifier fields, add a record whose acceptance or primary key depends on
// that rewrite, otherwise this guard silently stops covering the step.
func TestAcceptableCount_MatchesCacheLenAcrossSubsets(t *testing.T) {
	n := len(acceptableCountPool)
	require.LessOrEqual(t, n, 16, "子集枚举规模需保持可控")

	for mask := 0; mask < 1<<n; mask++ {
		subset := make([]define.AllowListUser, 0, n)
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				subset = append(subset, acceptableCountPool[i])
			}
		}
		if got, want := AcceptableCount(subset), cacheLen(subset); got != want {
			t.Fatalf("子集 %b 不一致: AcceptableCount=%d, 缓存生效条数=%d, 记录=%v",
				mask, got, want, subset)
		}
	}
}

// TestAcceptableCount_IsSilent asserts AcceptableCount emits no log lines. Set validates
// the same records again and logs each rejection; if the probe logged too, every rejected
// record would produce a duplicate warning (and a second masked-identifier line) on every
// refresh cycle.
func TestAcceptableCount_IsSilent(t *testing.T) {
	out := captureLog(t, func() {
		assert.Equal(t, 0, AcceptableCount(acceptableCountPool[3:6]),
			"全部为非法记录，生效条数应为 0")
	})
	assert.Empty(t, out, "AcceptableCount 不得产生任何日志输出")
}

// TestAcceptableCount_DoesNotMutateInput guards the caller's slice: the probe runs on the
// rule set that is about to be applied, so normalizing in place would silently change what
// gets written afterwards.
func TestAcceptableCount_DoesNotMutateInput(t *testing.T) {
	users := make([]define.AllowListUser, len(acceptableCountPool))
	copy(users, acceptableCountPool)
	before := fmt.Sprintf("%+v", users)

	AcceptableCount(users)

	assert.Equal(t, before, fmt.Sprintf("%+v", users), "输入切片不得被修改")
}
