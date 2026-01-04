// Package validator 提供了数据验证功能。
// 支持手机号和邮箱格式验证，支持多个国家/地区的手机号格式。
package validator

import (
	// 标准库
	"regexp"
	"strings"
)

var (
	// phoneRegexCN 匹配中国大陆手机号
	phoneRegexCN = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// phoneRegexUS 匹配美国手机号
	phoneRegexUS = regexp.MustCompile(`^\+?1[2-9]\d{2}[2-9]\d{6}$`)
	// phoneRegexUK 匹配英国手机号
	phoneRegexUK = regexp.MustCompile(`^\+?44[1-9]\d{8,9}$`)
	// phoneRegexInternational 匹配国际手机号（通用格式，7-15位数字，可能包含+号）
	phoneRegexInternational = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
	// emailRegex 更严格的邮箱验证（符合 RFC 5322 标准）
	// 改进：不允许连续的点，不允许以点开头或结尾，域名部分更严格
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?@[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`)
)

// ValidatePhone 验证手机号格式（支持多个国家/地区）
func ValidatePhone(phone string) bool {
	if phone == "" {
		return true // 允许为空
	}
	phone = strings.TrimSpace(phone)

	// 尝试匹配各种格式
	if phoneRegexCN.MatchString(phone) {
		return true // 中国大陆
	}
	if phoneRegexUS.MatchString(phone) {
		return true // 美国
	}
	if phoneRegexUK.MatchString(phone) {
		return true // 英国
	}
	// 通用国际格式（作为后备）
	return phoneRegexInternational.MatchString(phone)
}

// ValidateEmail 验证邮箱格式（更严格的验证）
func ValidateEmail(email string) bool {
	if email == "" {
		return true // 允许为空
	}
	email = strings.TrimSpace(email)

	// 基本格式检查
	if !emailRegex.MatchString(email) {
		return false
	}

	// 额外检查：不允许连续的点
	if strings.Contains(email, "..") {
		return false
	}

	// 检查 @ 符号前后的部分
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	localPart := parts[0]
	domainPart := parts[1]

	// 本地部分不能以点开头或结尾
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}

	// 域名部分不能以点开头或结尾
	if strings.HasPrefix(domainPart, ".") || strings.HasSuffix(domainPart, ".") {
		return false
	}

	// 域名部分必须包含至少一个点（用于TLD）
	if !strings.Contains(domainPart, ".") {
		return false
	}

	return true
}

// ValidateUser 验证用户数据
func ValidateUser(phone, email string) error {
	if !ValidatePhone(phone) {
		return &ValidationError{Field: "phone", Message: "无效的手机号格式"}
	}
	if !ValidateEmail(email) {
		return &ValidationError{Field: "email", Message: "无效的邮箱格式"}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
