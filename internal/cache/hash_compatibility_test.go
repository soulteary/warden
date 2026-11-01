package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
)

// Frozen from main 80e84e9: retain its normalization, comparator, unstable sort,
// field order and length framing. Do not replace this with production helpers.
func legacyHashUserList(users []define.AllowListUser) string {
	if len(users) == 0 {
		sum := sha256.Sum256([]byte("empty"))
		return hex.EncodeToString(sum[:])
	}
	sorted := make([]define.AllowListUser, len(users))
	copy(sorted, users)
	for i := range sorted {
		sorted[i].Normalize()
	}
	key := func(user define.AllowListUser) string { //nolint:gocritic // Preserve the previous comparator's value semantics.
		value := strings.TrimSpace(user.Phone)
		if value == "" {
			value = strings.ToLower(strings.TrimSpace(user.Mail))
		}
		return value
	}
	sort.Slice(sorted, func(i, j int) bool { return key(sorted[i]) < key(sorted[j]) })
	var encoded strings.Builder
	field := func(value string) {
		encoded.WriteString(strconv.Itoa(len(value)))
		encoded.WriteByte(':')
		encoded.WriteString(value)
	}
	for i := range sorted {
		field(sorted[i].Phone)
		field(sorted[i].Mail)
		field(sorted[i].UserID)
		field(sorted[i].Status)
		field(strconv.Itoa(len(sorted[i].Scope)))
		for _, scope := range sorted[i].Scope {
			field(scope)
		}
		field(sorted[i].Role)
		field(sorted[i].Name)
		field(sorted[i].DingtalkUserID)
	}
	sum := sha256.Sum256([]byte(encoded.String()))
	return hex.EncodeToString(sum[:])
}

func copyHashInput(users []define.AllowListUser) []define.AllowListUser {
	if users == nil {
		return nil
	}
	out := make([]define.AllowListUser, len(users))
	copy(out, users)
	for i := range out {
		if users[i].Scope != nil {
			out[i].Scope = append([]string{}, users[i].Scope...)
		}
	}
	return out
}

func assertLegacyHash(t *testing.T, users []define.AllowListUser) {
	t.Helper()
	before := copyHashInput(users)
	want := legacyHashUserList(users)
	require.Equal(t, want, HashUserList(users))
	require.Equal(t, before, users, "hashing must not normalize or reorder caller-owned data")
}

func TestHashUserListLegacyDifferential(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	for _, strategy := range []define.UserIDStrategy{define.UserIDStrategyLegacy, define.UserIDStrategySHA256_128} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%t", strategy, explicit), func(t *testing.T) {
				define.SetUserIDStrategy(strategy)
				define.SetRequireExplicitUserID(explicit)
				fixtures := [][]define.AllowListUser{nil, {}, {{}}, {{Phone: " 13800138000 ", Mail: " MIXED@Example.COM ", Scope: []string{"", "a:b", "a,b", "你好\x00🙂"}, Name: "0:1:名字", DingtalkUserID: "\xff"}}, {{Mail: "x@example.com", Scope: nil}}, {{Mail: "x@example.com", Scope: []string{}}}}
				fixtures = append(fixtures, []define.AllowListUser{{Mail: "long@example.com", UserID: "long", Name: strings.Repeat("你好:\x00", 20000), Scope: []string{strings.Repeat("scope", 1000), ""}}})
				for _, users := range fixtures {
					assertLegacyHash(t, users)
				}
				// #nosec G404 -- fixed-seed fixtures are intentionally reproducible, not security-sensitive.
				random := rand.New(rand.NewPCG(17, 43))
				fields := []string{"", " ", "13800138000", " 13800138000 ", "MiXeD@example.com", "mixed@example.com", "invalid", "a:b,c\n", "你好🙂", "\x00\xff"}
				for n := 0; n < 400; n++ {
					users := make([]define.AllowListUser, random.IntN(100))
					for i := range users {
						pick := func() string { return fields[random.IntN(len(fields))] }
						users[i] = define.AllowListUser{Phone: pick(), Mail: pick(), UserID: pick(), Status: pick(), Role: pick(), Name: pick(), DingtalkUserID: pick()}
						if i%3 != 0 {
							users[i].Scope = make([]string, random.IntN(5))
							for j := range users[i].Scope {
								users[i].Scope[j] = pick()
							}
						}
					}
					assertLegacyHash(t, users)
				}
			})
		}
	}
}

func TestPreparedRulesVersionsMatchLegacyHash(t *testing.T) {
	for _, users := range [][]define.AllowListUser{nil, {{Mail: "b@example.com", UserID: "b", Scope: []string{"write", "read"}}, {Mail: "a@example.com", UserID: "a"}}, {{Mail: "valid@example.com", UserID: "valid"}, {Mail: "invalid", UserID: "invalid"}}, {{UserID: "unusable"}}} {
		prepared, err := PrepareRules(users, identity.Options{})
		require.NoError(t, err)
		require.Equal(t, legacyHashUserList(users), prepared.InputVersion())
		require.Equal(t, legacyHashUserList(prepared.ReadView().Get()), prepared.ReadView().GetHash())
	}
}

func FuzzHashUserListLegacy(f *testing.F) {
	for _, seed := range []string{`[]`, `[{}]`, `[{"mail":" X@Example.com ","scope":["a:b",""]}]`, `[{"phone":"1","user_id":"a"},{"phone":"1","user_id":"b"}]`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 65536 {
			return
		}
		var users []define.AllowListUser
		if json.Unmarshal(raw, &users) != nil || len(users) > 128 {
			return
		}
		assertLegacyHash(t, users)
	})
}

func TestHashUserListHistoricalDigests(t *testing.T) {
	tied := make([]define.AllowListUser, 25)
	for i := range tied {
		tied[i] = define.AllowListUser{Phone: "same", UserID: fmt.Sprint(i), Name: fmt.Sprint(25 - i)}
	}
	vectors := map[string]struct {
		want  string
		users []define.AllowListUser
	}{
		"empty":           {want: "2e1cfa82b035c26cbbbdae632cea070514eb8b773f616aaeaf668e2f0be8f10d"},
		"unicode":         {want: "a89fa0ed438f86040ada368d9e884a9b9dca86078a47887ab2b81402577bf570", users: []define.AllowListUser{{Mail: " MIXED@Example.COM ", UserID: "fixed", Status: "active", Scope: []string{"", "a:b", "a,b", "你好\x00🙂"}, Role: " admin ", Name: "0:1:名字", DingtalkUserID: "\xff"}}},
		"scope framing A": {want: "8e40b99ca7d88597612c67bf2b92fc7fbe16eac224846e47c41057d6d44d1043", users: []define.AllowListUser{{Phone: "13800138000", UserID: "fixed", Scope: []string{"a,b", "c"}}}},
		"scope framing B": {want: "24fd824add0c63aa786008b2baffaa38cee075a64bd27e973c5db612aba3ca99", users: []define.AllowListUser{{Phone: "13800138000", UserID: "fixed", Scope: []string{"a", "b,c"}}}},
		"equal keys":      {want: "91e189b785c4cb096b00b0b0248b15a00bd5ac0b2858a52f8b31aded036e8a61", users: tied},
	}
	for name, vector := range vectors {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, vector.want, legacyHashUserList(vector.users))
			require.Equal(t, vector.want, HashUserList(vector.users))
		})
	}
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	users := []define.AllowListUser{{Mail: " Mixed@Example.COM "}}
	for strategy, want := range map[define.UserIDStrategy]string{define.UserIDStrategyLegacy: "bf218262e223015bae60b4f02be82ebdd312520a283ca7bbf43b4fd4bb405a75", define.UserIDStrategySHA256_128: "9910f6f59288423a631596e3f3110c94f5d4a364285252996648f5854e47e9dd"} {
		define.SetUserIDStrategy(strategy)
		define.SetRequireExplicitUserID(false)
		require.Equal(t, want, HashUserList(users))
	}
	define.SetRequireExplicitUserID(true)
	require.Equal(t, "c1ccc0ee25a2b6b8f6a4a5ebe3bf2a61dbb95d6a60b9ceea6573ab5e7a48c53c", HashUserList(users))
}
