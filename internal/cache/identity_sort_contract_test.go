package cache

import (
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"
)

// Frozen identity ordering from main bf8863e.
func legacyStableSort(users []define.AllowListUser) []define.AllowListUser {
	out := make([]define.AllowListUser, len(users))
	copy(out, users)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := out[i].UserID, out[j].UserID
		if ki != kj {
			return ki < kj
		}
		ci, cj := legacyCanonicalKey(&out[i]), legacyCanonicalKey(&out[j])
		if ci != cj {
			return ci < cj
		}
		return strings.ToLower(out[i].Mail) < strings.ToLower(out[j].Mail)
	})
	return out
}

func legacyCanonicalKey(u *define.AllowListUser) string {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.ToLower(strings.TrimSpace(u.Mail))
	}
	return k
}

func TestPreparedRulesLegacyIdentityOrder(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	// #nosec G404 -- deterministic differential fixtures.
	rng := rand.New(rand.NewPCG(13, 37))
	for _, strategy := range []define.UserIDStrategy{define.UserIDStrategyLegacy, define.UserIDStrategySHA256_128} {
		define.SetUserIDStrategy(strategy)
		for _, explicit := range []bool{false, true} {
			define.SetRequireExplicitUserID(explicit)
			for sample := 0; sample < 100; sample++ {
				users := make([]define.AllowListUser, rng.IntN(100))
				for i := range users {
					users[i] = define.AllowListUser{Mail: fmt.Sprintf("Member%d@Example.com", i), UserID: fmt.Sprintf("id-%d", i), Status: "active", Scope: []string{"write", "read"}}
					if i%7 == 0 {
						users[i].Mail = fmt.Sprintf("invalid-%d", i)
					}
					if !explicit && i%5 == 0 {
						users[i].UserID = ""
					}
				}
				rng.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
				before := copyHashInput(users)
				normalized := copyHashInput(users)
				for i := range normalized {
					normalized[i].Normalize()
				}
				expected := buildReadView(legacyStableSort(normalized), false)
				prepared, err := PrepareRules(users, identity.Options{RequireExplicitUserID: explicit})
				require.NoError(t, err)
				require.Equal(t, expected.Get(), prepared.ReadView().Get())
				require.Equal(t, legacyHashUserList(users), prepared.InputVersion())
				require.Equal(t, legacyHashUserList(expected.Get()), prepared.ReadView().GetHash())
				require.Equal(t, before, users)
			}
		}
	}
}
