// Package logger 提供了日志记录功能。
// 基于 zerolog 实现结构化日志，支持日志级别控制和敏感信息脱敏。
package logger

import (
	// 标准库
	"os"
	"strings"

	// 第三方库
	"github.com/rs/zerolog"
)

var globalLevel zerolog.Level = zerolog.InfoLevel

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// 从环境变量读取日志级别
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
		if err == nil {
			globalLevel = level
		}
	}

	zerolog.SetGlobalLevel(globalLevel)
}

// GetLogger 获取日志实例
func GetLogger() zerolog.Logger {
	logger := zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger().
		Level(globalLevel)

	return logger
}

// SetLevel 设置日志级别（用于运行时调整）
func SetLevel(level zerolog.Level) {
	globalLevel = level
	zerolog.SetGlobalLevel(level)
}

// SanitizeString 脱敏敏感信息
// 对可能包含敏感信息的字符串进行部分脱敏处理
func SanitizeString(s string) string {
	if s == "" {
		return s
	}

	// 如果字符串较短，只显示首尾
	if len(s) <= 4 {
		return "***"
	}

	// 显示前2个字符和后2个字符，中间用*替代
	prefix := s[:2]
	suffix := s[len(s)-2:]
	masked := strings.Repeat("*", len(s)-4)
	return prefix + masked + suffix
}

// SanitizeHeader 脱敏 HTTP 头信息
func SanitizeHeader(header string) string {
	// 对 Authorization 等敏感头进行脱敏
	lowerHeader := strings.ToLower(header)
	if strings.Contains(lowerHeader, "authorization") ||
		strings.Contains(lowerHeader, "token") ||
		strings.Contains(lowerHeader, "api-key") {
		return SanitizeString(header)
	}
	return header
}
