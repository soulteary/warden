// Package router 提供了 HTTP 路由处理功能。
// 包括请求日志记录、JSON 响应、健康检查等路由处理器。
package router

import (
	// 标准库
	"context"
	"encoding/json"
	"net/http"
	"time"

	// 第三方库
	"github.com/redis/go-redis/v9"

	// 项目内部包
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/logger"
)

// HealthCheck 返回健康检查处理器
// 检查 Redis 连接状态和数据是否已加载
// appMode 控制响应详细程度：生产环境（"production"）隐藏详细信息，开发环境显示详细信息
func HealthCheck(redisClient *redis.Client, userCache *cache.SafeUserCache, appMode string) http.HandlerFunc {
	// 判断是否为生产环境
	isProduction := appMode == "production" || appMode == "prod"

	return func(w http.ResponseWriter, _ *http.Request) {
		status := "ok"
		code := http.StatusOK
		details := make(map[string]interface{})

		// 检查 Redis 连接
		if redisClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := redisClient.Ping(ctx).Err(); err != nil {
				status = "redis_unavailable"
				code = http.StatusServiceUnavailable
				details["redis"] = "unavailable"
				// 生产环境不返回详细错误信息，避免泄露敏感信息
				if !isProduction {
					// 仅在非生产环境返回详细错误信息
					details["redis_error"] = err.Error()
				}
			} else {
				details["redis"] = "ok"
			}
		} else {
			details["redis"] = "not_configured"
		}

		// 检查数据是否已加载
		if userCache != nil {
			userCount := userCache.Len()
			details["data_loaded"] = userCount > 0
			// 生产环境隐藏具体用户数量，只返回是否已加载
			if isProduction {
				// 生产环境：只返回布尔值，不返回具体数量
				details["data_loaded"] = userCount > 0
			} else {
				// 开发环境：返回详细信息
				details["data_loaded"] = userCount > 0
				details["user_count"] = userCount
			}
			if userCount == 0 {
				// 数据未加载不影响健康状态，但记录在 details 中
				if !isProduction {
					// 仅在非生产环境返回警告信息
					details["data_warning"] = "no data loaded yet"
				}
			}
		} else {
			details["data_loaded"] = false
			if !isProduction {
				// 仅在非生产环境返回警告信息
				details["data_warning"] = "cache not initialized"
			}
		}

		response := map[string]interface{}{
			"status":  status,
			"details": details,
		}

		// 生产环境不返回模式信息
		if !isProduction {
			response["mode"] = appMode
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log := logger.GetLogger()
			log.Error().
				Err(err).
				Msg("健康检查响应编码失败")
			// 如果已经写入了状态码，无法再修改，只能记录错误
		}
	}
}
