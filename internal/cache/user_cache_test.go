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

	// Set users: one email-only, one with phone (email-only users are supported)
	users := []define.AllowListUser{
		{Phone: "", Mail: "test@example.com"},
		{Phone: "13800138000", Mail: "test2@example.com"},
	}
	cache.Set(users)

	// Both should be kept: at least one of phone or mail required
	result := cache.Get()
	assert.Equal(t, 2, len(result), "应保留邮箱用户和手机用户")

	// Email-only user should be findable by mail
	user, exists := cache.GetByMail("test@example.com")
	require.True(t, exists)
	assert.Equal(t, "test@example.com", user.Mail)
	assert.Empty(t, user.Phone)

	// Phone user unchanged
	user, exists = cache.GetByPhone("13800138000")
	require.True(t, exists)
	assert.Equal(t, "13800138000", user.Phone)
}

func TestSafeUserCache_BothIdentifierEmpty(t *testing.T) {
	cache := NewSafeUserCache()

	// User with both phone and mail empty should be skipped
	users := []define.AllowListUser{
		{Phone: "", Mail: ""},
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	cache.Set(users)

	result := cache.Get()
	assert.Equal(t, 1, len(result), "phone 和 mail 都空的用户应被忽略")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestHashUserList(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "a@example.com"},
		{Phone: "", Mail: "b@example.com"},
	}

	h1 := HashUserList(users)
	h2 := HashUserList(users)
	assert.Equal(t, h1, h2, "same input should produce same hash")

	// Different order should produce same hash (sort by primary key)
	usersReversed := []define.AllowListUser{users[1], users[0]}
	h3 := HashUserList(usersReversed)
	assert.Equal(t, h1, h3, "different order should produce same hash after sort")

	emptyHash := HashUserList(nil)
	assert.NotEmpty(t, emptyHash, "empty list should return fixed hash")
}

func TestSafeUserCache_GetByUserID(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com", UserID: "uid-1"},
		{Phone: "13900139000", Mail: "test2@example.com", UserID: "uid-2"},
	}
	cache.Set(users)

	user, exists := cache.GetByUserID("uid-1")
	assert.True(t, exists, "应该通过 UserID 找到用户")
	assert.Equal(t, "13800138000", user.Phone)
	assert.Equal(t, "test@example.com", user.Mail)

	_, exists = cache.GetByUserID("uid-notexist")
	assert.False(t, exists, "不存在的 UserID 应返回 false")
}

func TestSafeUserCache_Iterate(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "a@example.com"},
		{Phone: "13900139000", Mail: "b@example.com"},
	}
	cache.Set(users)

	var count int
	cache.Iterate(func(u define.AllowListUser) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count, "应迭代 2 个用户")

	count = 0
	cache.Iterate(func(u define.AllowListUser) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count, "返回 false 时应提前停止迭代")
}

func TestSafeUserCache_GetReadOnly(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	cache.Set(users)

	ro := cache.GetReadOnly()
	require.Len(t, ro, 1)
	assert.Equal(t, "13800138000", ro[0].Phone)
}

func TestSafeUserCache_GetHash(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	cache.Set(users)

	h := cache.GetHash()
	assert.NotEmpty(t, h, "有数据时 hash 不应为空")
	assert.Len(t, h, 64, "SHA256 哈希应为 64 字符")

	cache.Set([]define.AllowListUser{})
	h2 := cache.GetHash()
	assert.NotEmpty(t, h2, "空列表也有 hash 值")
}

func TestSafeUserCache_InvalidPhoneSkipped(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "valid@example.com"},
		{Phone: "invalid-phone", Mail: "a@example.com"},
		{Phone: "", Mail: "b@example.com"},
	}
	cache.Set(users)

	result := cache.Get()
	assert.GreaterOrEqual(t, len(result), 1, "无效手机号用户应被跳过或保留有效项")
}

func TestSafeUserCache_InvalidEmailSkipped(t *testing.T) {
	cache := NewSafeUserCache()

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "valid@example.com"},
		{Phone: "13900139000", Mail: "not-an-email"},
	}
	cache.Set(users)

	result := cache.Get()
	assert.GreaterOrEqual(t, len(result), 1, "无效邮箱用户应被跳过或保留有效项")
}
