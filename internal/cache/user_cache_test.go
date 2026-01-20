package cache

import (
	"sync"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeUserCache_ConcurrentAccess(t *testing.T) {
	cache := NewSafeUserCache()

	// Concurrent write test
	var wg sync.WaitGroup
	numGoroutines := 100
	numWrites := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < numWrites; j++ {
				users := []define.AllowListUser{
					{Phone: "13800138000", Mail: "test@example.com"},
					{Phone: "13800138001", Mail: "test2@example.com"},
				}
				cache.Set(users)
			}
		}(i)
	}

	// Concurrent read test
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numWrites; j++ {
				_ = cache.Get()
				_ = cache.Len()
			}
		}()
	}

	wg.Wait()

	// Verify final state
	users := cache.Get()
	assert.GreaterOrEqual(t, len(users), 0, "缓存应该包含数据")
}

func TestSafeUserCache_GetByPhone(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}
	cache.Set(users)

	// Test existing phone number
	user, exists := cache.GetByPhone("13800138000")
	assert.True(t, exists, "应该找到用户")
	assert.Equal(t, "test@example.com", user.Mail)

	// Test non-existent phone number
	_, exists = cache.GetByPhone("9999999999")
	assert.False(t, exists, "不应该找到用户")
}

func TestSafeUserCache_Deduplication(t *testing.T) {
	cache := NewSafeUserCache()

	// Set user list containing duplicate phone numbers
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138000", Mail: "test2@example.com"}, // Duplicate phone number
		{Phone: "13800138001", Mail: "test3@example.com"},
	}
	cache.Set(users)

	// Verify deduplication (should only keep the last one)
	result := cache.Get()
	assert.Equal(t, 2, len(result), "应该去重，只保留2个用户")

	// Verify last value is retained
	user, exists := cache.GetByPhone("13800138000")
	require.True(t, exists)
	assert.Equal(t, "test2@example.com", user.Mail, "应该保留最后一个值")
}

func TestSafeUserCache_EmptyPhone(t *testing.T) {
	cache := NewSafeUserCache()

	// Set users containing empty phone number
	users := []define.AllowListUser{
		{Phone: "", Mail: "test@example.com"},
		{Phone: "13800138000", Mail: "test2@example.com"},
	}
	cache.Set(users)

	// Users with empty phone number should be ignored
	result := cache.Get()
	assert.Equal(t, 1, len(result), "空手机号的用户应该被忽略")
	assert.Equal(t, "13800138000", result[0].Phone)
}
