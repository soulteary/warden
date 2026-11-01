package cache

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
)

func BenchmarkRefreshHash(b *testing.B) {
	for _, count := range []int{0, 1, 10, 1000, 10000} {
		users := make([]define.AllowListUser, count)
		for i := range users {
			users[i] = define.AllowListUser{Mail: fmt.Sprintf("Member%05d@Example.com", i), UserID: fmt.Sprintf("member-%05d", i), Status: "active", Scope: []string{"read", "write"}, Name: "Member"}
		}
		// #nosec G404 -- fixed-seed fixtures are intentionally reproducible, not security-sensitive.
		random := rand.New(rand.NewPCG(71, 93))
		random.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
		c := NewSafeUserCache()
		c.Set(users)
		if c.Len() != count {
			b.Fatalf("fixture retained %d of %d users", c.Len(), count)
		}
		b.Run(fmt.Sprintf("Hash/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				HashUserList(users)
			}
		})
		b.Run(fmt.Sprintf("Prepare/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := PrepareRules(users, identity.Options{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
