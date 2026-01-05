// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net"
	"net/http"
	"os"
	"strings"

	// 第三方库
	"github.com/rs/zerolog/hlog"
)

// IPWhitelistMiddleware 创建 IP 白名单中间件
//
// 该中间件只允许白名单中的 IP 地址访问受保护的端点。
// 白名单通过环境变量 IP_WHITELIST 配置，支持逗号分隔的多个 IP 或 CIDR 网段。
//
// 参数:
//   - whitelist: IP 白名单（逗号分隔的 IP 地址或 CIDR 网段）
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func IPWhitelistMiddleware(whitelist string) func(http.Handler) http.Handler {
	// 如果未配置白名单，允许所有 IP（向后兼容）
	if whitelist == "" {
		whitelist = os.Getenv("IP_WHITELIST")
	}
	if whitelist == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// 解析白名单
	allowedIPs, allowedNetworks := parseIPWhitelist(whitelist)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			// 检查 IP 是否在白名单中
			if !isIPAllowed(clientIP, allowedIPs, allowedNetworks) {
				hlog.FromRequest(r).Warn().
					Str("ip", clientIP).
					Str("path", r.URL.Path).
					Msg("IP 不在白名单中，访问被拒绝")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseIPWhitelist 解析 IP 白名单
func parseIPWhitelist(whitelist string) (map[string]bool, []*net.IPNet) {
	allowedIPs := make(map[string]bool)
	var allowedNetworks []*net.IPNet

	ips := strings.Split(whitelist, ",")
	for _, ipStr := range ips {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// 尝试解析为 CIDR 网段
		if _, network, err := net.ParseCIDR(ipStr); err == nil {
			allowedNetworks = append(allowedNetworks, network)
			continue
		}

		// 尝试解析为单个 IP
		if ip := net.ParseIP(ipStr); ip != nil {
			allowedIPs[ip.String()] = true
		}
	}

	return allowedIPs, allowedNetworks
}

// isIPAllowed 检查 IP 是否在白名单中
func isIPAllowed(ipStr string, allowedIPs map[string]bool, allowedNetworks []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 检查是否在单个 IP 白名单中
	if allowedIPs[ip.String()] {
		return true
	}

	// 检查是否在任何 CIDR 网段中
	for _, network := range allowedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
