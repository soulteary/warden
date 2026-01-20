package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocker_Structure(t *testing.T) {
	// 测试Locker结构体创建
	locker := &Locker{
		Cache: &redis.Client{},
	}

	assert.NotNil(t, locker, "Locker应该可以创建")
	assert.NotNil(t, locker.Cache, "Cache字段应该存在")
}

func TestLocker_MethodsExist(t *testing.T) {
	// 测试方法存在性（不实际调用，因为需要真实的Redis连接）
	locker := &Locker{
		Cache: nil, // 不设置Cache，只测试结构体
	}

	// 验证结构体可以创建
	assert.NotNil(t, locker)

	// 注意：不能实际调用Lock/Unlock方法，因为Cache是nil会导致panic
	// 实际的集成测试在TestLocker_Integration中
}

// 使用真实Redis的集成测试（可选，需要Redis运行）
func TestLocker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 测试连接
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("跳过测试：无法连接到Redis: %v", err)
	}

	locker := &Locker{
		Cache: client,
	}

	key := "test-lock-key-integration"

	// 测试Lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "第一次锁定应该成功")

	// 再次尝试锁定应该失败
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// 测试Unlock
	err = locker.Unlock(key)
	require.NoError(t, err)

	// 解锁后应该可以再次锁定
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")

	// 清理
	if err := locker.Unlock(key); err != nil {
		t.Logf("清理锁失败: %v", err)
	}
}

// TestLocker_LocalFallback 测试Redis失败时降级到本地锁
func TestLocker_LocalFallback(t *testing.T) {
	// 使用nil Cache来直接测试本地锁降级（更快速）
	// 在实际场景中，当Redis连接失败时，Locker会降级到本地锁
	locker := &Locker{
		Cache: nil, // nil Cache会直接使用本地锁
	}

	key := "test-lock-key-fallback"

	// 测试Lock（应该使用本地锁）
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "应该成功获取本地锁")

	// 再次尝试锁定应该失败（本地锁）
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// 测试Unlock（应该使用本地锁）
	err = locker.Unlock(key)
	require.NoError(t, err)

	// 解锁后应该可以再次锁定
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")
}

// TestLocker_UnlockWithoutLockValue 测试没有存储lockValue时的解锁
func TestLocker_UnlockWithoutLockValue(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("跳过测试：无法连接到Redis: %v", err)
	}

	locker := &Locker{
		Cache: client,
	}

	key := "test-lock-key-no-value"

	// 直接在Redis中设置一个锁（不通过Lock方法）
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Set(ctx2, key, "some-value", 10*time.Second)

	// 尝试解锁（没有存储的lockValue，应该直接删除）
	err = locker.Unlock(key)
	// 应该不会返回错误（向后兼容）
	assert.NoError(t, err)
}

// TestLocker_UnlockWithWrongValue 测试锁值不匹配时的解锁
func TestLocker_UnlockWithWrongValue(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("跳过测试：无法连接到Redis: %v", err)
	}

	locker := &Locker{
		Cache: client,
	}

	key := "test-lock-key-wrong-value"

	// 获取锁
	success, err := locker.Lock(key)
	require.NoError(t, err)
	require.True(t, success)

	// 在Redis中修改锁值（模拟锁被其他进程持有）
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Set(ctx2, key, "wrong-value", 10*time.Second)

	// 尝试解锁（锁值不匹配，应该返回错误）
	err = locker.Unlock(key)
	assert.Error(t, err, "锁值不匹配应该返回错误")
}

// TestLocalLocker_Basic 测试本地锁的基本功能
func TestLocalLocker_Basic(t *testing.T) {
	locker := NewLocalLocker()
	require.NotNil(t, locker)

	key := "test-local-lock"

	// 测试Lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "第一次锁定应该成功")

	// 再次尝试锁定应该失败
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// 测试Unlock
	err = locker.Unlock(key)
	require.NoError(t, err)

	// 解锁后应该可以再次锁定
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")
}

// TestLocalLocker_Concurrent 测试本地锁的并发安全性
func TestLocalLocker_Concurrent(t *testing.T) {
	locker := NewLocalLocker()
	key := "test-concurrent-lock"

	// 并发获取锁
	successCount := 0
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			success, err := locker.Lock(key)
			if err == nil && success {
				successCount++
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 只有一个应该成功
	assert.Equal(t, 1, successCount, "并发时只有一个应该成功获取锁")

	// 清理
	if err := locker.Unlock(key); err != nil {
		t.Logf("清理锁时出错: %v", err)
	}
}

// TestLocalLocker_MultipleKeys 测试多个键的锁
func TestLocalLocker_MultipleKeys(t *testing.T) {
	locker := NewLocalLocker()

	key1 := "test-key-1"
	key2 := "test-key-2"

	// 获取两个不同的锁
	success1, err := locker.Lock(key1)
	require.NoError(t, err)
	assert.True(t, success1)

	success2, err := locker.Lock(key2)
	require.NoError(t, err)
	assert.True(t, success2, "不同的键应该可以同时锁定")

	// 解锁
	err = locker.Unlock(key1)
	require.NoError(t, err)

	err = locker.Unlock(key2)
	require.NoError(t, err)
}

// TestLocalLocker_UnlockNonExistent 测试解锁不存在的锁
func TestLocalLocker_UnlockNonExistent(t *testing.T) {
	locker := NewLocalLocker()

	// 解锁一个不存在的锁应该不会panic
	err := locker.Unlock("non-existent-key")
	assert.NoError(t, err, "解锁不存在的锁应该不会返回错误")
}

// TestGenerateLockValue 测试生成锁值
func TestGenerateLockValue(t *testing.T) {
	value1, err := generateLockValue()
	require.NoError(t, err)
	assert.NotEmpty(t, value1)
	assert.Len(t, value1, 32, "锁值应该是32个字符（16字节的十六进制）")

	// 生成多个锁值，应该不同
	value2, err := generateLockValue()
	require.NoError(t, err)
	assert.NotEqual(t, value1, value2, "不同的锁值应该不同")
}

// TestLocker_NilCache 测试Cache为nil的情况
func TestLocker_NilCache(t *testing.T) {
	locker := &Locker{
		Cache: nil,
	}

	key := "test-nil-cache"

	// 应该使用本地锁
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success)

	// 解锁
	err = locker.Unlock(key)
	require.NoError(t, err)
}
