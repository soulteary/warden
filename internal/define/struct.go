package define

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AllowListUser 表示允许列表中的用户信息。
//
// 该结构体用于存储用户的基本信息，包括手机号、邮箱地址、用户ID、状态等。
// 这些信息用于验证和授权用户访问。
//
//nolint:govet // fieldalignment: 字段顺序受 JSON 序列化标签影响，优化可能破坏 API 兼容性
type AllowListUser struct {
	Phone  string   `json:"phone"`   // 用户手机号
	Mail   string   `json:"mail"`    // 用户邮箱地址
	UserID string   `json:"user_id"` // 用户唯一标识符（可选，如果未提供则自动生成）
	Status string   `json:"status"`  // 用户状态（如 "active", "inactive", "suspended"）
	Scope  []string `json:"scope"`   // 用户权限范围（可选）
	Role   string   `json:"role"`    // 用户角色（可选）
}

// Normalize 规范化用户数据，设置默认值并生成 user_id（如果未提供）
//
// 该函数会：
// - 如果 user_id 为空，基于 phone 或 mail 生成
// - 如果 status 为空，设置为 "active"
// - 如果 scope 为 nil，设置为空数组
// - 如果 role 为空，设置为空字符串
func (u *AllowListUser) Normalize() {
	// 生成 user_id（如果未提供）
	if u.UserID == "" {
		identifier := strings.TrimSpace(u.Phone)
		if identifier == "" {
			identifier = strings.TrimSpace(strings.ToLower(u.Mail))
		}
		if identifier != "" {
			h := sha256.Sum256([]byte(identifier))
			u.UserID = hex.EncodeToString(h[:])[:16] // 取前 16 个字符
		}
	}

	// 设置默认 status
	if u.Status == "" {
		u.Status = "active"
	}

	// 设置默认 scope（如果为 nil）
	if u.Scope == nil {
		u.Scope = []string{}
	}

	// role 可以为空字符串，不需要设置默认值
}
