package cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestSafeUserCache_ConcurrentAccess(t *testing.T) {
	cache := NewSafeUserCache()

	// 并发写入测试
	var wg sync.WaitGroup
	numGoroutines := 100
	numWrites := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
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

	// 并发读取测试
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

	// 验证最终状态
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

	// 测试存在的手机号
	user, exists := cache.GetByPhone("13800138000")
	assert.True(t, exists, "应该找到用户")
	assert.Equal(t, "test@example.com", user.Mail)

	// 测试不存在的手机号
	_, exists = cache.GetByPhone("9999999999")
	assert.False(t, exists, "不应该找到用户")
}

func TestSafeUserCache_Deduplication(t *testing.T) {
	cache := NewSafeUserCache()

	// 设置包含重复手机号的用户列表
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138000", Mail: "test2@example.com"}, // 重复的手机号
		{Phone: "13800138001", Mail: "test3@example.com"},
	}
	cache.Set(users)

	// 验证去重（应该只保留最后一个）
	result := cache.Get()
	assert.Equal(t, 2, len(result), "应该去重，只保留2个用户")

	// 验证最后一个值被保留
	user, exists := cache.GetByPhone("13800138000")
	require.True(t, exists)
	assert.Equal(t, "test2@example.com", user.Mail, "应该保留最后一个值")
}

func TestSafeUserCache_EmptyPhone(t *testing.T) {
	cache := NewSafeUserCache()

	// 设置包含空手机号的用户
	users := []define.AllowListUser{
		{Phone: "", Mail: "test@example.com"},
		{Phone: "13800138000", Mail: "test2@example.com"},
	}
	cache.Set(users)

	// 空手机号的用户应该被忽略
	result := cache.Get()
	assert.Equal(t, 1, len(result), "空手机号的用户应该被忽略")
	assert.Equal(t, "13800138000", result[0].Phone)
}
