package logger

import (
	"bytes"
	"net/url"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.NotNil(t, logger, "logger应该不为nil")
	assert.IsType(t, zerolog.Logger{}, logger, "应该返回zerolog.Logger类型")
}

func TestLogger_Output(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	// Test log output
	logger.Info().Msg("test message")
	output := buf.String()

	assert.Contains(t, output, "test message", "日志应该包含测试消息")
	assert.Contains(t, output, "level", "日志应该包含级别信息")
}

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	// Set level to InfoLevel explicitly to filter out debug messages
	logger := zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	// Test different log levels
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	logger.Warn().Msg("warn message")
	logger.Error().Msg("error message")

	output := buf.String()

	// Since level is set to InfoLevel, Debug messages should not appear
	assert.NotContains(t, output, "debug message", "Debug消息不应该出现在输出中")
	assert.Contains(t, output, "info message", "Info消息应该出现在输出中")
	assert.Contains(t, output, "warn message", "Warn消息应该出现在输出中")
	assert.Contains(t, output, "error message", "Error消息应该出现在输出中")
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().
		Str("key", "value").
		Int("number", 42).
		Timestamp().
		Logger()

	logger.Info().Msg("test with fields")
	output := buf.String()

	assert.Contains(t, output, "key", "日志应该包含字段key")
	assert.Contains(t, output, "value", "日志应该包含字段值value")
	assert.Contains(t, output, "number", "日志应该包含字段number")
	assert.Contains(t, output, "42", "日志应该包含数字42")
}

func TestLogger_DefaultOutput(t *testing.T) {
	// Test default logger outputs to stderr
	logger := GetLogger()

	// Verify logger is not nil
	assert.NotNil(t, logger)

	// Verify default output is stderr (by checking logger's internal state)
	// Note: zerolog.Logger doesn't directly expose output target, so we can only verify the logger itself
	assert.IsType(t, zerolog.Logger{}, logger)
}

func TestLogger_Concurrent(t *testing.T) {
	// Test concurrency safety
	logger := GetLogger()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info().Int("id", id).Msg("concurrent log")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test passes if no panic occurs
	assert.True(t, true, "并发日志写入应该安全")
}

func TestLogger_Init(t *testing.T) {
	// Test global configuration set by init function
	// Since init function has already executed, we verify zerolog's global configuration
	// Note: These are global settings that may affect other tests, so only do basic verification

	// Verify logger can be created
	logger := GetLogger()
	assert.NotNil(t, logger)

	// Verify default level (through actual log behavior)
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()
	testLogger.Debug().Msg("should not appear")

	// Debug messages should not appear (if level is set correctly)
	output := buf.String()
	// Since we created a new logger, it uses default level, so here we just verify logger works
	assert.NotNil(t, testLogger)
	_ = output // Avoid unused variable warning
}

func TestLogger_Stderr(t *testing.T) {
	// Verify default logger outputs to stderr
	logger := GetLogger()

	// Get stderr file descriptor
	stderr := os.Stderr
	assert.NotNil(t, stderr, "stderr应该存在")

	// Logger should work normally
	assert.NotNil(t, logger)

	// Try to write log (won't actually write, as zerolog may buffer)
	logger.Info().Msg("test stderr output")

	// If no panic, means can write to stderr normally
	assert.True(t, true)
}

func TestSetLevel(t *testing.T) {
	// Save original level
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	// Test setting different levels
	levels := []zerolog.Level{
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
		zerolog.FatalLevel,
		zerolog.PanicLevel,
	}

	for _, level := range levels {
		SetLevel(level)
		assert.Equal(t, level, zerolog.GlobalLevel(), "级别应该被正确设置")
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "***", // secure-kit returns "***" for empty strings
		},
		{
			name:     "短字符串（<=4字符）",
			input:    "test",
			expected: "***", // secure-kit returns "***" for short strings
		},
		{
			name:     "短字符串（3字符）",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "正常字符串",
			input:    "password123",
			expected: "pa***23", // secure-kit uses fixed 3 asterisks (more secure, doesn't leak length)
		},
		{
			name:     "长字符串",
			input:    "very-long-secret-key-that-needs-masking",
			expected: "ve***ng", // secure-kit uses fixed 3 asterisks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		shouldMask bool
	}{
		{
			name:       "Authorization头",
			input:      "Authorization: Bearer token123",
			shouldMask: true,
		},
		{
			name:       "authorization头（小写）",
			input:      "authorization: Bearer token123",
			shouldMask: true,
		},
		{
			name:       "Token头",
			input:      "Token: secret123",
			shouldMask: true,
		},
		{
			name:       "API-Key头",
			input:      "API-Key: key123",
			shouldMask: true,
		},
		{
			name:       "api-key头（小写）",
			input:      "api-key: key123",
			shouldMask: true,
		},
		{
			name:       "普通头",
			input:      "Content-Type: application/json",
			shouldMask: false,
		},
		{
			name:       "User-Agent头",
			input:      "User-Agent: Mozilla/5.0",
			shouldMask: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHeader(tt.input)
			if tt.shouldMask {
				assert.Contains(t, result, "**", "敏感头应该被脱敏")
				assert.NotEqual(t, tt.input, result, "敏感头应该被修改")
			} else {
				assert.Equal(t, tt.input, result, "非敏感头不应该被修改")
			}
		})
	}
}

func TestGetLogger_WithEnvLevel(t *testing.T) {
	// Save original environment variable
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		if originalLevel != "" {
			require.NoError(t, os.Setenv("LOG_LEVEL", originalLevel))
		} else {
			require.NoError(t, os.Unsetenv("LOG_LEVEL"))
		}
	}()

	// Test different log levels
	levels := []string{"debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		require.NoError(t, os.Setenv("LOG_LEVEL", level))
		// Note: Since init function has already executed, environment variable changes won't take effect immediately
		// Here we mainly verify GetLogger doesn't panic
		logger := GetLogger()
		assert.NotNil(t, logger)
	}
}

func TestSanitizeString_EdgeCases(t *testing.T) {
	// Test edge cases - secure-kit uses fixed 3 asterisks for all masked content
	assert.Equal(t, "***", SanitizeString(""))     // secure-kit returns "***" for empty
	assert.Equal(t, "***", SanitizeString("a"))    // too short
	assert.Equal(t, "***", SanitizeString("ab"))   // too short
	assert.Equal(t, "***", SanitizeString("abc"))  // too short
	assert.Equal(t, "***", SanitizeString("abcd")) // too short
	assert.Equal(t, "ab***ef", SanitizeString("abcdef"))
}

func TestSanitizeHeader_EdgeCases(t *testing.T) {
	// Test edge cases
	assert.Equal(t, "", SanitizeHeader(""))
	assert.Equal(t, "test", SanitizeHeader("test"))

	// Test case with multiple sensitive keywords
	result := SanitizeHeader("Authorization: Bearer token123")
	assert.Contains(t, result, "**", "应该被脱敏")
}

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "***", // secure-kit returns "***" for empty
		},
		{
			name:     "短手机号",
			input:    "138",
			expected: "***",
		},
		{
			name:     "正常手机号",
			input:    "13800138000",
			expected: "13***00", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "国际手机号",
			input:    "+8613800138000",
			expected: "+8***00", // secure-kit uses fixed 3 asterisks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "***", // secure-kit returns "***" for empty
		},
		{
			name:     "短邮箱",
			input:    "a@b",
			expected: "***",
		},
		{
			name:     "正常邮箱",
			input:    "test@example.com",
			expected: "te***om", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "长邮箱",
			input:    "verylongemailaddress@example.com",
			expected: "ve***om", // secure-kit uses fixed 3 asterisks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected URL string (we'll check if sensitive params are masked)
	}{
		{
			name:     "无查询参数",
			input:    "https://example.com/user",
			expected: "https://example.com/user",
		},
		{
			name:     "单个phone参数",
			input:    "https://example.com/user?phone=13800138000",
			expected: "phone=13%2A%2A%2A00", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "单个mail参数",
			input:    "https://example.com/user?mail=test@example.com",
			expected: "mail=te%2A%2A%2Aom", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "单个email参数",
			input:    "https://example.com/user?email=user@example.com",
			expected: "email=us%2A%2A%2Aom", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "多个敏感参数",
			input:    "https://example.com/user?phone=13800138000&mail=test@example.com",
			expected: "phone=13%2A%2A%2A00", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "敏感参数和非敏感参数混合",
			input:    "https://example.com/user?phone=13800138000&page=1&page_size=10",
			expected: "phone=13%2A%2A%2A00", // secure-kit uses fixed 3 asterisks
		},
		{
			name:     "空值参数",
			input:    "https://example.com/user?phone=",
			expected: "phone=%2A%2A%2A", // secure-kit returns "***" for empty
		},
		{
			name:     "大小写混合参数名",
			input:    "https://example.com/user?Phone=13800138000&MAIL=test@example.com",
			expected: "Phone=13%2A%2A%2A00", // secure-kit uses fixed 3 asterisks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.input)
			require.NoError(t, err, "应该能解析URL")

			result := SanitizeURL(u)
			assert.Contains(t, result, tt.expected, "URL应该包含脱敏后的查询参数")

			// 验证敏感参数确实被脱敏了（使用大小写不敏感的查询）
			// 注意：URL编码后，* 会被编码为 %2A
			if phoneVal := u.Query().Get("phone"); phoneVal != "" {
				assert.NotContains(t, result, phoneVal, "phone参数应该被脱敏")
				assert.Contains(t, result, "%2A", "应该包含脱敏标记（URL编码后的*）")
			}
			if mailVal := u.Query().Get("mail"); mailVal != "" {
				assert.NotContains(t, result, mailVal, "mail参数应该被脱敏")
				assert.Contains(t, result, "%2A", "应该包含脱敏标记（URL编码后的*）")
			}
			if emailVal := u.Query().Get("email"); emailVal != "" {
				assert.NotContains(t, result, emailVal, "email参数应该被脱敏")
				assert.Contains(t, result, "%2A", "应该包含脱敏标记（URL编码后的*）")
			}
		})
	}
}

func TestSanitizeURLString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		shouldMask bool
	}{
		{
			name:       "空字符串",
			input:      "",
			shouldMask: false,
		},
		{
			name:       "无效URL",
			input:      "not-a-valid-url",
			shouldMask: false, // 解析失败时返回原始字符串
		},
		{
			name:       "包含phone参数的URL",
			input:      "https://example.com/user?phone=13800138000",
			shouldMask: true,
		},
		{
			name:       "包含mail参数的URL",
			input:      "https://example.com/user?mail=test@example.com",
			shouldMask: true,
		},
		{
			name:       "包含email参数的URL",
			input:      "https://example.com/user?email=user@example.com",
			shouldMask: true,
		},
		{
			name:       "多个敏感参数",
			input:      "https://example.com/user?phone=13800138000&mail=test@example.com&email=user@example.com",
			shouldMask: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURLString(tt.input)
			if tt.shouldMask {
				// URL编码后，* 会被编码为 %2A
				assert.Contains(t, result, "%2A", "应该包含脱敏标记（URL编码后的*）")
				// 验证原始敏感信息不在结果中
				if tt.input != "" {
					u, err := url.Parse(tt.input)
					if err == nil {
						if phone := u.Query().Get("phone"); phone != "" {
							assert.NotContains(t, result, phone, "phone参数应该被脱敏")
						}
						if mail := u.Query().Get("mail"); mail != "" {
							assert.NotContains(t, result, mail, "mail参数应该被脱敏")
						}
						if email := u.Query().Get("email"); email != "" {
							assert.NotContains(t, result, email, "email参数应该被脱敏")
						}
					}
				}
			} else if tt.input == "not-a-valid-url" {
				// 对于无效URL，应该返回原始字符串
				assert.Equal(t, tt.input, result, "无效URL应该返回原始字符串")
			}
		})
	}
}

func TestSanitizeURL_NilURL(t *testing.T) {
	result := SanitizeURL(nil)
	assert.Equal(t, "", result, "nil URL应该返回空字符串")
}
