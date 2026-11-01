package cache

import (
	"fmt"
	"sync"
	"testing"

	"github.com/soulteary/warden/internal/define"
)

// benchmarkUsers supplies distinct valid identities so benchmarks retain every row.
func benchmarkUsers(b *testing.B, count int) ([]define.AllowListUser, *SafeUserCache) {
	b.Helper()
	users := make([]define.AllowListUser, count)
	for i := range users {
		users[i] = define.AllowListUser{
			Phone: fmt.Sprintf("138%08d", i), Mail: fmt.Sprintf("member%d@example.com", i),
			UserID: fmt.Sprintf("member-%d", i), Status: "active", Scope: []string{"read", "write"},
		}
	}
	c := NewSafeUserCache()
	c.Set(users)
	if c.Len() != count {
		b.Fatalf("fixture retained %d of %d rows", c.Len(), count)
	}
	if user, ok := c.GetByPhone(users[0].Phone); !ok || user.UserID != users[0].UserID {
		b.Fatal("fixture lookup did not find the expected user")
	}
	b.ReportAllocs()
	return users, c
}

// BenchmarkSafeUserCache_Get tests performance of Get method
func BenchmarkSafeUserCache_Get(b *testing.B) {
	_, cache := benchmarkUsers(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Get()
	}
}

// BenchmarkSafeUserCache_Set tests performance of Set method
func BenchmarkSafeUserCache_Set(b *testing.B) {
	users, cache := benchmarkUsers(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(users)
	}
}

// BenchmarkSafeUserCache_GetByPhone tests performance of GetByPhone method (O(1) lookup)
func BenchmarkSafeUserCache_GetByPhone(b *testing.B) {
	users, cache := benchmarkUsers(b, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetByPhone(users[500].Phone)
	}
}

// BenchmarkSafeUserCache_ConcurrentRead tests concurrent read performance
func BenchmarkSafeUserCache_ConcurrentRead(b *testing.B) {
	_, cache := benchmarkUsers(b, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cache.Get()
		}
	})
}

// BenchmarkSafeUserCache_ConcurrentWrite tests concurrent write performance
func BenchmarkSafeUserCache_ConcurrentWrite(b *testing.B) {
	users, cache := benchmarkUsers(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set(users)
		}
	})
}

// BenchmarkSafeUserCache_ConcurrentReadWrite tests concurrent read-write performance
func BenchmarkSafeUserCache_ConcurrentReadWrite(b *testing.B) {
	users, cache := benchmarkUsers(b, 100)

	var wg sync.WaitGroup
	b.ResetTimer()

	// Start read goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			_ = cache.Get()
		}
	}()

	// Start write goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			cache.Set(users)
		}
	}()

	wg.Wait()
}

func BenchmarkSafeUserCache_GetByPhoneMiss(b *testing.B) {
	_, c := benchmarkUsers(b, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.GetByPhone("13999999999")
	}
}
