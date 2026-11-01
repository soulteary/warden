package warden

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestCacheScopeOwnsSetAndGetMemory(t *testing.T) {
	cache := NewCache(time.Minute)
	users := []AllowListUser{{UserID: "member", Scope: []string{"read"}}, {UserID: "nil", Scope: nil}, {UserID: "empty", Scope: []string{}}}
	cache.Set(users)
	users[0].Scope[0] = "changed input"
	first := cache.Get()
	require.Equal(t, []string{"read"}, first[0].Scope)
	require.Nil(t, first[1].Scope)
	require.NotNil(t, first[2].Scope)
	require.Empty(t, first[2].Scope)
	first[0].Scope[0] = "changed result"
	first[0].Scope = append(first[0].Scope, "write")
	require.Equal(t, []string{"read"}, cache.Get()[0].Scope)
	cache.Set(nil)
	require.NotNil(t, cache.Get())
	require.Empty(t, cache.Get())
	cache.Clear()
	require.Nil(t, cache.Get())
}

func TestCacheScopeConcurrentCallersAreIsolated(t *testing.T) {
	cache := NewCache(time.Minute)
	input := []AllowListUser{{UserID: "member", Scope: []string{"read"}}}
	cache.Set(input)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				got := cache.Get()
				got[0].Scope[0] = "caller change"
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			input[0].Scope[0] = "input change"
		}
	}()
	wg.Wait()
	require.Equal(t, []string{"read"}, cache.Get()[0].Scope)
}
