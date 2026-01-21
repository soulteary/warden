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
	// Test Locker struct creation
	locker := &Locker{
		Cache: &redis.Client{},
	}

	assert.NotNil(t, locker, "Locker应该可以创建")
	assert.NotNil(t, locker.Cache, "Cache字段应该存在")
}

func TestLocker_MethodsExist(t *testing.T) {
	// Test method existence (without actually calling, as it requires real Redis connection)
	locker := &Locker{
		Cache: nil, // Don't set Cache, only test struct
	}

	// Verify struct can be created
	assert.NotNil(t, locker)

	// Note: Cannot actually call Lock/Unlock methods, as Cache being nil will cause panic
	// Actual integration tests are in TestLocker_Integration
}

// Integration test using real Redis (optional, requires Redis running)
func TestLocker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("跳过测试：无法连接到Redis: %v", err)
	}

	locker := &Locker{
		Cache: client,
	}

	key := "test-lock-key-integration"

	// Test Lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "第一次锁定应该成功")

	// Second attempt to lock should fail
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// Test Unlock
	err = locker.Unlock(key)
	require.NoError(t, err)

	// After unlocking, should be able to lock again
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")

	// Cleanup
	if err := locker.Unlock(key); err != nil {
		t.Logf("清理锁失败: %v", err)
	}
}

// TestLocker_LocalFallback tests fallback to local lock when Redis fails
func TestLocker_LocalFallback(t *testing.T) {
	// Use nil Cache to directly test local lock fallback (faster)
	// In actual scenarios, when Redis connection fails, Locker will fallback to local lock
	locker := &Locker{
		Cache: nil, // nil Cache will directly use local lock
	}

	key := "test-lock-key-fallback"

	// Test Lock (should use local lock)
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "应该成功获取本地锁")

	// Second attempt to lock should fail (local lock)
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// Test Unlock (should use local lock)
	err = locker.Unlock(key)
	require.NoError(t, err)

	// After unlocking, should be able to lock again
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")
}

// TestLocker_UnlockWithoutLockValue tests unlocking when lockValue is not stored
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

	// Set a lock directly in Redis (not through Lock method)
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Set(ctx2, key, "some-value", 10*time.Second)

	// Try to unlock (no stored lockValue, should directly delete)
	err = locker.Unlock(key)
	// Should not return error (backward compatibility)
	assert.NoError(t, err)
}

// TestLocker_UnlockWithWrongValue tests unlocking when lock value doesn't match
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

	// Acquire lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	require.True(t, success)

	// Modify lock value in Redis (simulate lock held by another process)
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Set(ctx2, key, "wrong-value", 10*time.Second)

	// Try to unlock (lock value doesn't match, should return error)
	err = locker.Unlock(key)
	assert.Error(t, err, "锁值不匹配应该返回错误")
}

// TestLocalLocker_Basic tests basic functionality of local lock
func TestLocalLocker_Basic(t *testing.T) {
	locker := NewLocalLocker()
	require.NotNil(t, locker)

	key := "test-local-lock"

	// Test Lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success, "第一次锁定应该成功")

	// Second attempt to lock should fail
	success2, err := locker.Lock(key)
	require.NoError(t, err)
	assert.False(t, success2, "第二次锁定应该失败")

	// Test Unlock
	err = locker.Unlock(key)
	require.NoError(t, err)

	// After unlocking, should be able to lock again
	success3, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success3, "解锁后应该可以再次锁定")
}

// TestLocalLocker_Concurrent tests concurrency safety of local lock
func TestLocalLocker_Concurrent(t *testing.T) {
	locker := NewLocalLocker()
	key := "test-concurrent-lock"

	// Concurrently acquire lock
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

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Only one should succeed
	assert.Equal(t, 1, successCount, "并发时只有一个应该成功获取锁")

	// Cleanup
	if err := locker.Unlock(key); err != nil {
		t.Logf("清理锁时出错: %v", err)
	}
}

// TestLocalLocker_MultipleKeys tests locks for multiple keys
func TestLocalLocker_MultipleKeys(t *testing.T) {
	locker := NewLocalLocker()

	key1 := "test-key-1"
	key2 := "test-key-2"

	// Acquire two different locks
	success1, err := locker.Lock(key1)
	require.NoError(t, err)
	assert.True(t, success1)

	success2, err := locker.Lock(key2)
	require.NoError(t, err)
	assert.True(t, success2, "不同的键应该可以同时锁定")

	// Unlock
	err = locker.Unlock(key1)
	require.NoError(t, err)

	err = locker.Unlock(key2)
	require.NoError(t, err)
}

// TestLocalLocker_UnlockNonExistent tests unlocking non-existent lock
func TestLocalLocker_UnlockNonExistent(t *testing.T) {
	locker := NewLocalLocker()

	// Unlocking a non-existent lock should not panic
	err := locker.Unlock("non-existent-key")
	assert.NoError(t, err, "解锁不存在的锁应该不会返回错误")
}

// TestGenerateLockValue tests lock value generation
// Note: This test is removed as generateLockValue is now handled internally by redis-kit's HybridLocker

// TestLocker_NilCache tests case when Cache is nil
func TestLocker_NilCache(t *testing.T) {
	locker := &Locker{
		Cache: nil,
	}

	key := "test-nil-cache"

	// Should use local lock
	success, err := locker.Lock(key)
	require.NoError(t, err)
	assert.True(t, success)

	// Unlock
	err = locker.Unlock(key)
	require.NoError(t, err)
}
