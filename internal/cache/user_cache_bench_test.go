package cache

import (
	"sync"
	"testing"

	"soulteary.com/soulteary/warden/internal/define"
)

// BenchmarkSafeUserCache_Get 测试 Get 方法的性能
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

// BenchmarkSafeUserCache_Set 测试 Set 方法的性能
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

// BenchmarkSafeUserCache_GetByPhone 测试 GetByPhone 方法的性能（O(1) 查找）
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

// BenchmarkSafeUserCache_ConcurrentRead 测试并发读取性能
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

// BenchmarkSafeUserCache_ConcurrentWrite 测试并发写入性能
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

// BenchmarkSafeUserCache_ConcurrentReadWrite 测试并发读写性能
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

	// 启动读取协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			_ = cache.Get()
		}
	}()

	// 启动写入协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			cache.Set(users)
		}
	}()

	wg.Wait()
}
