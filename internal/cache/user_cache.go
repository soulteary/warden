// Package cache 提供了用户数据的缓存功能。
// 支持内存缓存和 Redis 缓存两种实现，以及基于 Redis 的分布式锁。
package cache

import (
	// 标准库
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	// 项目内部包
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/validator"
)

var log = logger.GetLogger()

// slicePool 复用切片，减少内存分配
// slicePool 已移除，因为需要返回独立副本，无法直接使用池

// SafeUserCache 提供线程安全的用户缓存
// 使用 map 结构提高查找效率，同时保持 API 兼容性
// 维护顺序列表以保持插入顺序
// 缓存哈希值以优化数据变化检测
// 维护多个索引以支持通过 phone、mail、user_id 快速查询
type SafeUserCache struct { //nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
	mu       sync.RWMutex                    // 24 bytes
	order    []string                        // 24 bytes (8 pointer + 8 len + 8 cap)
	hash     string                          // 16 bytes (8 pointer + 8 len)
	users    map[string]define.AllowListUser // 8 bytes pointer (以 phone 为 key)
	byMail   map[string]string               // 8 bytes pointer (mail -> phone 映射)
	byUserID map[string]string               // 8 bytes pointer (user_id -> phone 映射)
}

// NewSafeUserCache 创建新的线程安全用户缓存
func NewSafeUserCache() *SafeUserCache {
	return &SafeUserCache{
		users:    make(map[string]define.AllowListUser),
		order:    make([]string, 0),
		byMail:   make(map[string]string),
		byUserID: make(map[string]string),
	}
}

// Get 获取用户列表的副本（线程安全）
// 为了保持 API 兼容性，返回切片格式
// 返回顺序与 Set 时的顺序一致
// 使用 sync.Pool 优化内存分配
func (c *SafeUserCache) Get() []define.AllowListUser {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 从池中获取切片（如果可用）
	// 注意：由于我们需要返回一个独立的副本，不能直接使用池中的切片
	// 但我们可以利用池来减少初始分配
	expectedLen := len(c.order)

	// 对于小数据，直接分配；对于大数据，考虑使用池
	// 但由于需要返回独立副本，这里仍然需要复制
	// 优化：预分配足够的容量
	result := make([]define.AllowListUser, 0, expectedLen)
	for _, phone := range c.order {
		if user, exists := c.users[phone]; exists {
			result = append(result, user)
		}
	}
	return result
}

// Set 设置用户列表（线程安全）
// 接受切片格式，内部转换为 map 存储
// 保持输入顺序
func (c *SafeUserCache) Set(users []define.AllowListUser) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清空现有数据
	c.users = make(map[string]define.AllowListUser, len(users))
	c.order = make([]string, 0, len(users))
	c.byMail = make(map[string]string, len(users))
	c.byUserID = make(map[string]string, len(users))

	// 使用 map 去重并存储（以 phone 为 key）
	// 同时维护顺序列表和索引
	validCount := 0
	invalidCount := 0
	duplicateCount := 0

	for _, user := range users {
		if user.Phone == "" {
			continue
		}
		// 规范化用户数据（设置默认值，生成 user_id）
		user.Normalize()

		// 验证用户数据
		if err := validator.ValidateUser(user.Phone, user.Mail); err != nil {
			invalidCount++
			// 安全地获取验证错误字段
			field := "unknown"
			if ve, ok := err.(*validator.ValidationError); ok {
				field = ve.Field
			}
			log.Warn().
				Err(err).
				Str("phone", user.Phone).
				Str("mail", user.Mail).
				Str("field", field).
				Msg("跳过无效用户数据")
			continue
		}
		// 如果 phone 已存在，更新数据但保持第一次出现的位置
		if _, exists := c.users[user.Phone]; !exists {
			c.order = append(c.order, user.Phone)
			validCount++
		} else {
			duplicateCount++
			log.Debug().
				Str("phone", user.Phone).
				Msg("更新已存在的用户数据")
		}
		c.users[user.Phone] = user

		// 建立 mail 索引（如果存在）
		if user.Mail != "" {
			mailKey := strings.ToLower(strings.TrimSpace(user.Mail))
			c.byMail[mailKey] = user.Phone
		}

		// 建立 user_id 索引（如果存在）
		if user.UserID != "" {
			c.byUserID[user.UserID] = user.Phone
		}
	}

	// 记录验证统计信息
	if invalidCount > 0 || duplicateCount > 0 {
		log.Info().
			Int("total", len(users)).
			Int("valid", validCount).
			Int("invalid", invalidCount).
			Int("duplicates", duplicateCount).
			Msg("数据验证完成")
	}

	// 计算并缓存哈希值（优化数据变化检测）
	c.hash = calculateHashInternal(c.users, c.order)
}

// calculateHashInternal 计算用户数据的哈希值（内部方法）
// 使用 map 和 order 列表，避免创建临时切片
func calculateHashInternal(users map[string]define.AllowListUser, order []string) string {
	if len(users) == 0 {
		h := sha256.New()
		h.Write([]byte("empty"))
		return hex.EncodeToString(h.Sum(nil))
	}

	// 创建排序后的用户列表用于哈希计算
	sortedUsers := make([]define.AllowListUser, 0, len(order))
	for _, phone := range order {
		if user, exists := users[phone]; exists {
			sortedUsers = append(sortedUsers, user)
		}
	}

	// 对用户列表进行排序，确保相同数据产生相同哈希
	sort.Slice(sortedUsers, func(i, j int) bool {
		if sortedUsers[i].Phone != sortedUsers[j].Phone {
			return sortedUsers[i].Phone < sortedUsers[j].Phone
		}
		return sortedUsers[i].Mail < sortedUsers[j].Mail
	})

	// 计算哈希（包含所有字段以确保数据变化检测准确）
	h := sha256.New()
	for _, user := range sortedUsers {
		scopeStr := strings.Join(user.Scope, ",")
		h.Write([]byte(user.Phone + ":" + user.Mail + ":" + user.UserID + ":" + user.Status + ":" + scopeStr + ":" + user.Role + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Len 获取用户数量（线程安全）
func (c *SafeUserCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.users)
}

// GetByPhone 根据手机号获取用户（线程安全，O(1) 查找）
func (c *SafeUserCache) GetByPhone(phone string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	user, exists := c.users[phone]
	return user, exists
}

// GetByMail 根据邮箱获取用户（线程安全，O(1) 查找）
func (c *SafeUserCache) GetByMail(mail string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	mailKey := strings.ToLower(strings.TrimSpace(mail))
	phone, exists := c.byMail[mailKey]
	if !exists {
		return define.AllowListUser{}, false
	}
	user, exists := c.users[phone]
	return user, exists
}

// GetByUserID 根据用户ID获取用户（线程安全，O(1) 查找）
func (c *SafeUserCache) GetByUserID(userID string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	phone, exists := c.byUserID[userID]
	if !exists {
		return define.AllowListUser{}, false
	}
	user, exists := c.users[phone]
	return user, exists
}

// Iterate 迭代所有用户，避免复制整个切片（线程安全）
// 回调函数按插入顺序接收用户数据
// 如果回调函数返回 false，迭代将停止
func (c *SafeUserCache) Iterate(fn func(user define.AllowListUser) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, phone := range c.order {
		if user, exists := c.users[phone]; exists {
			if !fn(user) {
				return
			}
		}
	}
}

// GetReadOnly 获取只读视图（实际返回副本，但语义上表示只读）
// 对于只读场景，建议使用 Iterate 方法以避免复制
func (c *SafeUserCache) GetReadOnly() []define.AllowListUser {
	return c.Get()
}

// GetHash 获取缓存的哈希值（线程安全）
// 如果哈希值未计算，返回空字符串
// 使用缓存的哈希值可以避免重复计算，提高性能
func (c *SafeUserCache) GetHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hash
}
