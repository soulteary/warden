package cache

import (
	"sync"
	"testing"

	"github.com/soulteary/warden/internal/define"
)

// BenchmarkSafeUserCache_Get tests performance of Get method
func BenchmarkSafeUserCache_Get(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}
	cache.Set(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Get()
	}
}

// BenchmarkSafeUserCache_Set tests performance of Set method
func BenchmarkSafeUserCache_Set(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(users)
	}
}

// BenchmarkSafeUserCache_GetByPhone tests performance of GetByPhone method (O(1) lookup)
func BenchmarkSafeUserCache_GetByPhone(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}
	cache.Set(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetByPhone("5")
	}
}

// BenchmarkSafeUserCache_ConcurrentRead tests concurrent read performance
func BenchmarkSafeUserCache_ConcurrentRead(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}
	cache.Set(users)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cache.Get()
		}
	})
}

// BenchmarkSafeUserCache_ConcurrentWrite tests concurrent write performance
func BenchmarkSafeUserCache_ConcurrentWrite(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set(users)
		}
	})
}

// BenchmarkSafeUserCache_ConcurrentReadWrite tests concurrent read-write performance
func BenchmarkSafeUserCache_ConcurrentReadWrite(b *testing.B) {
	cache := NewSafeUserCache()
	users := make([]define.AllowListUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = define.AllowListUser{
			Phone: string(rune('0' + (i % 10))),
			Mail:  "test@example.com",
		}
	}
	cache.Set(users)

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
