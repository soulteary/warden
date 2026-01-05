// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	// 第三方库
	"github.com/rs/zerolog/hlog"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
	"soulteary.com/soulteary/warden/internal/metrics"
)

// RateLimiter 实现简单的内存速率限制
type RateLimiter struct {
	mu           sync.RWMutex        // 24 bytes
	window       time.Duration       // 8 bytes
	rate         int                 // 8 bytes
	maxVisitors  int                 // 8 bytes
	maxWhitelist int                 // 8 bytes
	wg           sync.WaitGroup      // 12 bytes (padding to 16)
	visitors     map[string]*visitor // 8 bytes pointer
	whitelist    map[string]bool     // 8 bytes pointer
	cleanup      *time.Ticker        // 8 bytes pointer
	stopCh       chan struct{}       // 8 bytes pointer
}

type visitor struct {
	lastSeen time.Time // 24 bytes
	count    int       // 8 bytes
}

// NewRateLimiter 创建新的速率限制器
// rate: 允许的请求数
// window: 时间窗口（例如 1 * time.Minute）
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:     make(map[string]*visitor),
		rate:         rate,
		window:       window,
		cleanup:      time.NewTicker(define.RATE_LIMIT_CLEANUP_INTERVAL), // 定期清理过期记录
		stopCh:       make(chan struct{}),
		whitelist:    make(map[string]bool),
		maxVisitors:  define.MAX_VISITORS_MAP_SIZE,
		maxWhitelist: define.MAX_WHITELIST_SIZE,
	}

	// 启动清理协程
	rl.wg.Add(1)
	go rl.cleanupVisitors()

	return rl
}

// cleanupVisitors 定期清理过期的访问记录
func (rl *RateLimiter) cleanupVisitors() {
	defer rl.wg.Done()
	for {
		select {
		case <-rl.cleanup.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			// 如果仍然超过限制，清理最旧的记录
			if len(rl.visitors) > rl.maxVisitors {
				rl.cleanupOldestVisitors()
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// cleanupOldestVisitors 清理最旧的访问记录（当超过最大限制时）
func (rl *RateLimiter) cleanupOldestVisitors() {
	// 找到最旧的记录
	type visitorWithTime struct {
		lastSeen time.Time // 24 bytes
		ip       string    // 16 bytes
	}
	visitors := make([]visitorWithTime, 0, len(rl.visitors))
	for ip, v := range rl.visitors {
		visitors = append(visitors, visitorWithTime{ip: ip, lastSeen: v.lastSeen})
	}

	// 按时间排序，删除最旧的
	sort.Slice(visitors, func(i, j int) bool {
		return visitors[i].lastSeen.Before(visitors[j].lastSeen)
	})

	// 删除最旧的记录，直到低于限制
	toRemove := len(rl.visitors) - rl.maxVisitors
	for i := 0; i < toRemove && i < len(visitors); i++ {
		delete(rl.visitors, visitors[i].ip)
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.RLock()
	// 检查白名单
	if rl.whitelist[ip] {
		rl.mu.RUnlock()
		return true
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		// 检查是否超过最大限制
		if len(rl.visitors) >= rl.maxVisitors {
			// 清理最旧的记录
			rl.cleanupOldestVisitors()
		}
		rl.visitors[ip] = &visitor{
			count:    1,
			lastSeen: now,
		}
		return true
	}

	// 如果超过时间窗口，重置计数
	if now.Sub(v.lastSeen) > rl.window {
		v.count = 1
		v.lastSeen = now
		return true
	}

	// 检查是否超过限制
	if v.count >= rl.rate {
		// 记录限流指标
		metrics.RateLimitHits.WithLabelValues(ip).Inc()
		return false
	}

	v.count++
	v.lastSeen = now
	return true
}

// AddToWhitelist 将 IP 添加到白名单
func (rl *RateLimiter) AddToWhitelist(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 检查是否已存在
	if rl.whitelist[ip] {
		return true
	}

	// 检查是否超过最大限制
	if len(rl.whitelist) >= rl.maxWhitelist {
		return false // 白名单已满
	}

	rl.whitelist[ip] = true
	return true
}

// RemoveFromWhitelist 从白名单中移除 IP
func (rl *RateLimiter) RemoveFromWhitelist(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.whitelist, ip)
}

// IsWhitelisted 检查 IP 是否在白名单中
func (rl *RateLimiter) IsWhitelisted(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.whitelist[ip]
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	rl.cleanup.Stop()
	close(rl.stopCh)
	rl.wg.Wait() // 等待 goroutine 退出
}

// RateLimitMiddlewareWithLimiter 创建速率限制中间件（使用指定的 RateLimiter 实例）
// 推荐使用此方法，避免全局变量
func RateLimitMiddlewareWithLimiter(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取客户端 IP（安全方式）
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				hlog.FromRequest(r).Warn().
					Str("ip", ip).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Str("user_agent", r.UserAgent()).
					Str("referer", r.Referer()).
					Msg("Rate limit exceeded")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP 安全地获取客户端 IP 地址
// 改进的 IP 获取逻辑，增加了更严格的验证
func getClientIP(r *http.Request) string {
	// 优先使用 X-Real-IP（通常由反向代理设置，更可靠）
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		ip := strings.TrimSpace(realIP)
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			// 验证是否为私有 IP（如果是，可能表示配置问题）
			if !isPrivateIP(parsedIP) || isTrustedProxy(r) {
				return ip
			}
		}
	}

	// 其次使用 X-Forwarded-For，但需要验证
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// 取第一个 IP（可能是代理链）
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			// 验证 IP 格式
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				// 只有在信任代理的情况下才使用 X-Forwarded-For
				if isTrustedProxy(r) {
					return ip
				}
			}
		}
	}

	// 回退到 RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	// 验证 RemoteAddr 中的 IP
	if parsedIP := net.ParseIP(host); parsedIP != nil {
		return host
	}

	return r.RemoteAddr
}

// isPrivateIP 检查 IP 是否为私有 IP
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			ip4[0] == 127
	}
	return false
}

// isTrustedProxy 检查请求是否来自信任的代理
// 在生产环境中，应该根据实际部署情况配置信任的代理 IP
func isTrustedProxy(r *http.Request) bool {
	// 这里可以根据实际需求配置信任的代理 IP 列表
	// 例如：检查 RemoteAddr 是否在信任列表中
	// 当前实现：如果 RemoteAddr 是私有 IP，则认为可能来自内部代理
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	// 在生产环境中，应该配置实际的信任代理列表
	// 这里简化处理：如果是私有 IP，可能来自内部代理
	return isPrivateIP(ip)
}
