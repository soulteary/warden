// Package router 提供了 HTTP 路由处理功能。
// 包括请求日志记录、JSON 响应、健康检查等路由处理器。
package router

import (
	// 标准库
	"encoding/json"
	"net/http"
	"strings"

	// 第三方库
	"github.com/rs/zerolog"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/logger"
)

// LogLevelHandler 处理日志级别调整的 HTTP 端点
// 支持 GET（查询当前级别）和 POST（设置新级别）
func LogLevelHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// 获取当前日志级别
			currentLevel := zerolog.GlobalLevel()
			response := map[string]interface{}{
				"level": currentLevel.String(),
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log := logger.GetLogger()
				log.Error().Err(err).Msg("编码日志级别响应失败")
			}

		case http.MethodPost:
			// 设置新的日志级别
			var request struct {
				Level string `json:"level"`
			}

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error": "无效的请求体",
				}); err != nil {
					log := logger.GetLogger()
					log.Error().Err(err).Msg("编码错误响应失败")
				}
				return
			}

			level, err := zerolog.ParseLevel(strings.ToLower(request.Level))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error": "无效的日志级别，支持: trace, debug, info, warn, error, fatal, panic",
				}); err != nil {
					log := logger.GetLogger()
					log.Error().Err(err).Msg("编码错误响应失败")
				}
				return
			}

			logger.SetLevel(level)
			response := map[string]interface{}{
				"message": "日志级别已更新",
				"level":   level.String(),
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log := logger.GetLogger()
				log.Error().Err(err).Msg("编码日志级别响应失败")
			}

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			if err := json.NewEncoder(w).Encode(map[string]string{
				"error": "不支持的方法，请使用 GET 或 POST",
			}); err != nil {
				log := logger.GetLogger()
				log.Error().Err(err).Msg("编码错误响应失败")
			}
		}
	}
}
