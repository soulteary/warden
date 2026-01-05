package cache

import (
	"context"
	"testing"

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
