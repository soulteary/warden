package cache

import (
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"runtime"
	"testing"
)

func candidateBenchmarkUsers(count, generation int) []define.AllowListUser {
	users := make([]define.AllowListUser, count)
	for i := range users {
		users[i] = define.AllowListUser{Mail: fmt.Sprintf("Member%05d@Example.com", i), UserID: fmt.Sprintf("member-%05d", count-i), Status: "active", Name: fmt.Sprintf("generation-%d", generation), Scope: []string{"read", "write"}}
	}
	return users
}

func BenchmarkCandidateStages(b *testing.B) {
	for _, count := range []int{0, 1, 1000, 10000} {
		users := candidateBenchmarkUsers(count, 0)
		validated, err := identity.ValidateAndIndexUsers(users, identity.Options{})
		if err != nil {
			b.Fatal(err)
		}
		view := buildReadView(validated.Users, false)
		if view.Len() != count {
			b.Fatal("fixture lost rows")
		}
		b.Run(fmt.Sprintf("rows=%d", count), func(b *testing.B) {
			b.Run("CloneInput", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					result := cloneUsers(users)
					runtime.KeepAlive(result)
				}
			})
			b.Run("IdentityPublic", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					result, e := identity.ValidateAndIndexUsers(users, identity.Options{})
					if e != nil {
						b.Fatal(e)
					}
					runtime.KeepAlive(result)
				}
			})
			b.Run("IndexesAndRows", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					result := buildReadView(validated.Users, false)
					runtime.KeepAlive(result)
				}
			})
			b.Run("Prepare", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					result, e := PrepareRules(users, identity.Options{})
					if e != nil {
						b.Fatal(e)
					}
					runtime.KeepAlive(result)
				}
			})
		})
	}
}

// Measure live heap after GC with old views pinned. This is a noisy diagnostic,
// not a heap-size assertion, peak-memory measurement or process RSS estimate.
func BenchmarkReadViewRetention(b *testing.B) {
	if _, err := PrepareRules(candidateBenchmarkUsers(1000, -1), identity.Options{}); err != nil {
		b.Fatal(err)
	}
	for _, generations := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("rows=1000/generations=%d", generations), func(b *testing.B) {
			var liveBytes float64
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				b.StartTimer()
				views := make([]*ReadView, 0, generations)
				for generation := 0; generation < generations; generation++ {
					prepared, err := PrepareRules(candidateBenchmarkUsers(1000, generation), identity.Options{})
					if err != nil {
						b.Fatal(err)
					}
					views = append(views, prepared.ReadView())
				}
				b.StopTimer()
				runtime.GC()
				runtime.ReadMemStats(&after)
				liveBytes += float64(after.HeapAlloc) - float64(before.HeapAlloc)
				if views[0].Len() != 1000 || views[len(views)-1].Len() != 1000 {
					b.Fatal("retained view lost rows")
				}
				first := views[0].Get()
				last := views[len(views)-1].Get()
				if first[0].Name != "generation-0" || last[0].Name != fmt.Sprintf("generation-%d", generations-1) {
					b.Fatal("retained views mixed generations")
				}
				runtime.KeepAlive(views)
				b.StartTimer()
			}
			b.ReportMetric(float64(liveBytes)/float64(b.N), "live-B/pinned-set")
			b.ReportMetric(float64(liveBytes)/float64(b.N*generations), "live-B/generation")
		})
	}
}
