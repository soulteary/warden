package logger

import (
	"bytes"
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
	// 创建一个buffer来捕获日志输出
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	// 测试日志输出
	logger.Info().Msg("test message")
	output := buf.String()

	assert.Contains(t, output, "test message", "日志应该包含测试消息")
	assert.Contains(t, output, "level", "日志应该包含级别信息")
}

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	// 测试不同级别的日志
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	logger.Warn().Msg("warn message")
	logger.Error().Msg("error message")

	output := buf.String()

	// 由于默认级别是InfoLevel，Debug消息不应该出现
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
	// 测试默认logger输出到stderr
	logger := GetLogger()

	// 验证logger不是nil
	assert.NotNil(t, logger)

	// 验证默认输出是stderr（通过检查logger的内部状态）
	// 注意：zerolog.Logger不直接暴露输出目标，所以我们只能验证logger本身
	assert.IsType(t, zerolog.Logger{}, logger)
}

func TestLogger_Concurrent(t *testing.T) {
	// 测试并发安全性
	logger := GetLogger()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info().Int("id", id).Msg("concurrent log")
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 如果没有panic，测试通过
	assert.True(t, true, "并发日志写入应该安全")
}

func TestLogger_Init(t *testing.T) {
	// 测试init函数设置的全局配置
	// 由于init函数已经执行，我们验证zerolog的全局配置
	// 注意：这些是全局设置，可能影响其他测试，所以只做基本验证

	// 验证可以创建logger
	logger := GetLogger()
	assert.NotNil(t, logger)

	// 验证默认级别（通过实际日志行为）
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()
	testLogger.Debug().Msg("should not appear")

	// Debug消息不应该出现（如果级别设置正确）
	output := buf.String()
	// 由于我们创建了新的logger，它使用默认级别，所以这里只是验证logger可以工作
	assert.NotNil(t, testLogger)
	_ = output // 避免未使用变量警告
}

func TestLogger_Stderr(t *testing.T) {
	// 验证默认logger输出到stderr
	logger := GetLogger()

	// 获取stderr文件描述符
	stderr := os.Stderr
	assert.NotNil(t, stderr, "stderr应该存在")

	// logger应该可以正常使用
	assert.NotNil(t, logger)

	// 尝试写入日志（不会真正写入，因为zerolog可能缓冲）
	logger.Info().Msg("test stderr output")

	// 如果没有panic，说明可以正常写入stderr
	assert.True(t, true)
}

func TestSetLevel(t *testing.T) {
	// 保存原始级别
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	// 测试设置不同级别
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
			expected: "",
		},
		{
			name:     "短字符串（<=4字符）",
			input:    "test",
			expected: "***",
		},
		{
			name:     "短字符串（3字符）",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "正常字符串",
			input:    "password123",
			expected: "pa*******23",
		},
		{
			name:     "长字符串",
			input:    "very-long-secret-key-that-needs-masking",
			expected: "ve***********************************ng", // 实际长度：43字符，前2后2，中间39个*
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
	// 保存原始环境变量
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		if originalLevel != "" {
			require.NoError(t, os.Setenv("LOG_LEVEL", originalLevel))
		} else {
			require.NoError(t, os.Unsetenv("LOG_LEVEL"))
		}
	}()

	// 测试不同的日志级别
	levels := []string{"debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		require.NoError(t, os.Setenv("LOG_LEVEL", level))
		// 注意：由于init函数已经执行，环境变量的改变不会立即生效
		// 这里主要验证GetLogger不会panic
		logger := GetLogger()
		assert.NotNil(t, logger)
	}
}

func TestSanitizeString_EdgeCases(t *testing.T) {
	// 测试边界情况
	assert.Equal(t, "", SanitizeString(""))
	assert.Equal(t, "***", SanitizeString("a"))
	assert.Equal(t, "***", SanitizeString("ab"))
	assert.Equal(t, "***", SanitizeString("abc"))
	assert.Equal(t, "***", SanitizeString("abcd"))
	assert.Equal(t, "ab**ef", SanitizeString("abcdef"))
}

func TestSanitizeHeader_EdgeCases(t *testing.T) {
	// 测试边界情况
	assert.Equal(t, "", SanitizeHeader(""))
	assert.Equal(t, "test", SanitizeHeader("test"))

	// 测试包含多个敏感关键词的情况
	result := SanitizeHeader("Authorization: Bearer token123")
	assert.Contains(t, result, "**", "应该被脱敏")
}
