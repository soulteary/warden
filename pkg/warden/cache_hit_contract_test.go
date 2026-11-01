package warden

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestCacheHitExpiryAndOwnership(t *testing.T) {
	for _, bound := range []bool{false, true} {
		for _, elapsed := range []time.Duration{time.Minute - time.Nanosecond, time.Minute, time.Minute + time.Nanosecond} {
			for _, empty := range []bool{false, true} {
				t.Run(fmt.Sprintf("bound=%t/elapsed=%s/empty=%t", bound, elapsed, empty), func(t *testing.T) {
					clock := newSourceTestClock()
					now := clock.Now()
					cache := NewCache(time.Minute)
					cache.now = clock.Now
					input := []AllowListUser{{UserID: "member", Scope: []string{"read"}}, {UserID: "nil", Scope: nil}, {UserID: "empty", Scope: []string{}}}
					if empty {
						input = nil
					}
					deadline := time.Time{}
					if bound {
						deadline = now.Add(time.Minute)
					}
					require.True(t, cache.setUntil(input, deadline))
					if !empty {
						input[0].Scope[0] = "caller changed"
						input[0].UserID = "caller changed"
					}
					clock.set(now.Add(elapsed))
					hit := elapsed < time.Minute || (!bound && elapsed == time.Minute)
					direct := cache.Get()
					users, request := cache.beginRead()
					if !hit {
						require.Nil(t, direct)
						require.Nil(t, users)
						require.Equal(t, cacheRequest{sequence: 1}, request)
						return
					}
					require.NotNil(t, direct)
					require.NotNil(t, users)
					require.Equal(t, cacheRequest{}, request)
					require.Zero(t, cache.nextRequest)
					require.Equal(t, direct, users)
					if empty {
						require.Empty(t, users)
						return
					}
					require.Equal(t, "member", users[0].UserID)
					require.Equal(t, []string{"read"}, users[0].Scope)
					require.Nil(t, users[1].Scope)
					require.NotNil(t, users[2].Scope)
					users[0].Scope[0] = "read changed"
					users[0].UserID = "read changed"
					require.Equal(t, "member", direct[0].UserID)
					require.Equal(t, "read", direct[0].Scope[0])
					require.Equal(t, direct, cache.Get())
				})
			}
		}
	}
}

func TestCacheHitCopiesSurviveReplacement(t *testing.T) {
	cache := NewCache(time.Hour)
	cache.Set([]AllowListUser{{UserID: "old", Scope: []string{"read"}}})
	direct := cache.Get()
	hit, request := cache.beginRead()
	require.Equal(t, cacheRequest{}, request)
	cache.Clear()
	cache.Set([]AllowListUser{{UserID: "new", Scope: []string{"write"}}})
	require.Equal(t, "old", direct[0].UserID)
	require.Equal(t, "read", hit[0].Scope[0])
	direct[0].Scope[0] = "changed"
	hit[0].Scope[0] = "also changed"
	require.Equal(t, []AllowListUser{{UserID: "new", Scope: []string{"write"}}}, cache.Get())
}

func TestCacheHitsConcurrentWithReplacement(t *testing.T) {
	const count = 128
	fixture := func(tag string) []AllowListUser {
		users := make([]AllowListUser, count)
		for i := range users {
			users[i] = AllowListUser{UserID: tag, Scope: []string{tag}}
		}
		return users
	}
	first, second := fixture("first"), fixture("second")
	cache := NewCache(time.Hour)
	cache.Set(first)
	start := make(chan struct{})
	errors := make(chan error, 4)
	var workers sync.WaitGroup
	for reader := 0; reader < 4; reader++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for i := 0; i < 300; i++ {
				users := cache.Get()
				if i%2 == 0 {
					users, _ = cache.beginRead()
				}
				if users == nil {
					continue
				}
				if len(users) != count {
					errors <- fmt.Errorf("got %d rows", len(users))
					return
				}
				tag := users[0].UserID
				for j := range users {
					if users[j].UserID != tag || len(users[j].Scope) != 1 || users[j].Scope[0] != tag {
						errors <- fmt.Errorf("mixed or modified generation at row %d", j)
						return
					}
				}
				users[0].UserID = "changed"
				users[0].Scope[0] = "changed"
			}
		}()
	}
	close(start)
	for i := 0; i < 300; i++ {
		cache.Clear()
		if i%2 == 0 {
			cache.Set(first)
		} else {
			cache.Set(second)
		}
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, "first", first[0].Scope[0])
	require.Equal(t, "second", second[0].Scope[0])
}

func BenchmarkCacheHit(b *testing.B) {
	for _, count := range []int{0, 1, 1000, 10000} {
		users := make([]AllowListUser, count)
		for i := range users {
			users[i] = AllowListUser{UserID: fmt.Sprintf("member-%d", i), Scope: []string{"read", "write"}}
		}
		cache := NewCache(time.Hour)
		cache.Set(users)
		if got := cache.Get(); got == nil || len(got) != count {
			b.Fatal("invalid cache fixture")
		}
		for _, method := range []struct {
			read func() []AllowListUser
			name string
		}{
			{name: "Get", read: cache.Get},
			{name: "beginRead", read: func() []AllowListUser { rows, _ := cache.beginRead(); return rows }},
		} {
			b.Run(fmt.Sprintf("rows=%d/%s", count, method.name), func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						rows := method.read()
						if rows == nil || len(rows) != count {
							b.Error("unexpected cache miss")
							return
						}
					}
				})
			})
		}
	}
}
