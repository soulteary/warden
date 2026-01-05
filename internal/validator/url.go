// Package validator 提供了配置验证功能。
// 包括 URL 验证、路径验证等安全验证功能。
package validator

import (
	// 标准库
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// ValidateRemoteURL 验证远程配置 URL，防止 SSRF 攻击
//
// 该函数对远程配置 URL 进行严格验证，包括：
// - 只允许 http:// 和 https:// 协议
// - 禁止访问私有 IP 地址（10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8）
// - 禁止访问 localhost
// - 验证 URL 格式的有效性
//
// 参数:
//   - urlStr: 要验证的 URL 字符串
//
// 返回:
//   - error: 如果 URL 无效或存在安全风险，返回错误；否则返回 nil
func ValidateRemoteURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL 不能为空")
	}

	// 解析 URL
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("无效的 URL 格式: %w", err)
	}

	// 只允许 http 和 https 协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("只允许 http 和 https 协议，当前协议: %s", u.Scheme)
	}

	// 验证 host
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL 必须包含有效的 host")
	}

	// 禁止 localhost 和 127.0.0.1
	hostLower := strings.ToLower(host)
	if hostLower == "localhost" || hostLower == "127.0.0.1" || hostLower == "::1" {
		return fmt.Errorf("不允许访问 localhost")
	}

	// 解析 IP 地址
	ip := net.ParseIP(host)
	if ip != nil {
		// 禁止私有 IP 地址
		if isPrivateIP(ip) {
			return fmt.Errorf("不允许访问私有 IP 地址: %s", ip.String())
		}
		// 禁止回环地址
		if ip.IsLoopback() {
			return fmt.Errorf("不允许访问回环地址: %s", ip.String())
		}
	}

	return nil
}

// isPrivateIP 检查 IP 是否为私有 IP
//
// 私有 IP 地址范围：
// - 10.0.0.0/8 (10.0.0.0 到 10.255.255.255)
// - 172.16.0.0/12 (172.16.0.0 到 172.31.255.255)
// - 192.168.0.0/16 (192.168.0.0 到 192.168.255.255)
// - 127.0.0.0/8 (127.0.0.0 到 127.255.255.255) - 回环地址
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			ip4[0] == 127
	}
	// IPv6 私有地址检查
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

// ValidateConfigPath 验证配置文件路径，防止路径遍历攻击
//
// 该函数对配置文件路径进行验证，包括：
// - 检查路径是否包含路径遍历字符（..）
// - 验证路径是否为绝对路径或相对路径
// - 可选：限制配置文件只能从特定目录读取
//
// 参数:
//   - path: 要验证的文件路径
//   - allowedDirs: 允许的目录列表（可选，如果为空则不限制目录）
//
// 返回:
//   - string: 规范化后的绝对路径
//   - error: 如果路径无效或存在安全风险，返回错误；否则返回 nil
func ValidateConfigPath(path string, allowedDirs []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("配置文件路径不能为空")
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("无法解析配置文件路径: %w", err)
	}

	// 检查是否包含路径遍历
	if strings.Contains(absPath, "..") {
		return "", fmt.Errorf("配置文件路径不能包含路径遍历字符 (..)")
	}

	// 如果指定了允许的目录，检查路径是否在允许的目录下
	if len(allowedDirs) > 0 {
		allowed := false
		for _, allowedDir := range allowedDirs {
			allowedAbsDir, err := filepath.Abs(allowedDir)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, allowedAbsDir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("配置文件必须在允许的目录下: %v", allowedDirs)
		}
	}

	return absPath, nil
}
