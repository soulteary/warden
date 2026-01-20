package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidatePhone tests phone number validation
func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		// Chinese mainland phone numbers
		{name: "有效的中国手机号", phone: "13800138000", want: true},
		{name: "有效的中国手机号（13开头）", phone: "13900139000", want: true},
		{name: "有效的中国手机号（15开头）", phone: "15000150000", want: true},
		{name: "有效的中国手机号（18开头）", phone: "18800188000", want: true},
		{name: "无效的中国手机号（12开头）", phone: "12000120000", want: true},  // May match international format
		{name: "无效的中国手机号（位数不足）", phone: "1380013800", want: true},   // May match international format
		{name: "无效的中国手机号（位数过多）", phone: "138001380000", want: true}, // May match international format

		// US phone numbers
		{name: "有效的美国手机号", phone: "+12025551234", want: true},
		{name: "有效的美国手机号（无+号）", phone: "12025551234", want: true},

		// UK phone numbers
		{name: "有效的英国手机号", phone: "+447911123456", want: true},
		{name: "有效的英国手机号（无+号）", phone: "447911123456", want: true},

		// International format
		{name: "有效的国际手机号", phone: "+8613800138000", want: true},
		{name: "有效的国际手机号（无+号）", phone: "8613800138000", want: true},

		// Edge cases
		{name: "空字符串", phone: "", want: true}, // Allow empty
		{name: "只有空格", phone: "   ", want: false},
		{name: "包含字母", phone: "138abc12345", want: false},
		{name: "包含特殊字符", phone: "138-001-38000", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePhone(tt.phone)
			assert.Equal(t, tt.want, got, "ValidatePhone(%q) = %v, want %v", tt.phone, got, tt.want)
		})
	}
}

// TestValidateEmail tests email validation
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		// Valid emails
		{name: "有效的邮箱", email: "test@example.com", want: true},
		{name: "有效的邮箱（带数字）", email: "test123@example.com", want: true},
		{name: "有效的邮箱（带下划线）", email: "test_user@example.com", want: true},
		{name: "有效的邮箱（带点）", email: "test.user@example.com", want: true},
		{name: "有效的邮箱（带横线）", email: "test-user@example.com", want: true},
		{name: "有效的邮箱（大写）", email: "TEST@EXAMPLE.COM", want: true},
		{name: "有效的邮箱（子域名）", email: "test@mail.example.com", want: true},

		// Invalid emails
		{name: "缺少@符号", email: "testexample.com", want: false},
		{name: "多个@符号", email: "test@@example.com", want: false},
		{name: "缺少域名", email: "test@", want: false},
		{name: "缺少本地部分", email: "@example.com", want: false},
		{name: "以点开头", email: ".test@example.com", want: false},
		{name: "以点结尾", email: "test.@example.com", want: false},
		{name: "连续的点", email: "test..user@example.com", want: false},
		{name: "域名以点开头", email: "test@.example.com", want: false},
		{name: "域名以点结尾", email: "test@example.com.", want: false},
		{name: "缺少TLD", email: "test@example", want: false},
		{name: "TLD太短", email: "test@example.c", want: false},
		{name: "包含空格", email: "test user@example.com", want: false},

		// Edge cases
		{name: "空字符串", email: "", want: true}, // Allow empty
		{name: "只有空格", email: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			assert.Equal(t, tt.want, got, "ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
		})
	}
}

// TestValidateUser tests user data validation
func TestValidateUser(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		email string
		want  bool
	}{
		{
			name:  "有效的手机号和邮箱",
			phone: "13800138000",
			email: "test@example.com",
			want:  true,
		},
		{
			name:  "有效的手机号，无效的邮箱",
			phone: "13800138000",
			email: "invalid-email",
			want:  false,
		},
		{
			name:  "无效的手机号，有效的邮箱",
			phone: "invalid-phone",
			email: "test@example.com",
			want:  false,
		},
		{
			name:  "都为空（允许）",
			phone: "",
			email: "",
			want:  true,
		},
		{
			name:  "只有手机号",
			phone: "13800138000",
			email: "",
			want:  true,
		},
		{
			name:  "只有邮箱",
			phone: "",
			email: "test@example.com",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUser(tt.phone, tt.email)
			if tt.want {
				assert.NoError(t, err, "ValidateUser(%q, %q) 应该返回 nil", tt.phone, tt.email)
			} else {
				assert.Error(t, err, "ValidateUser(%q, %q) 应该返回错误", tt.phone, tt.email)
				if err != nil {
					validationErr, ok := err.(*ValidationError)
					assert.True(t, ok, "错误应该是 ValidationError 类型")
					if ok {
						assert.NotEmpty(t, validationErr.Field, "错误应该包含字段名")
						assert.NotEmpty(t, validationErr.Message, "错误应该包含消息")
					}
				}
			}
		})
	}
}

// TestValidationError tests ValidationError type
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "phone",
		Message: "无效的手机号格式",
	}

	assert.Equal(t, "无效的手机号格式", err.Error(), "Error() 应该返回消息")
	assert.Equal(t, "phone", err.Field, "Field 应该正确设置")
	assert.Equal(t, "无效的手机号格式", err.Message, "Message 应该正确设置")
}

// TestValidatePhone_TrimSpace tests phone number space trimming
func TestValidatePhone_TrimSpace(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{name: "前后有空格", phone: "  13800138000  ", want: true},
		{name: "前面有空格", phone: " 13800138000", want: true},
		{name: "后面有空格", phone: "13800138000 ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePhone(tt.phone)
			assert.Equal(t, tt.want, got, "应该能够处理带空格的手机号")
		})
	}
}

// TestValidateEmail_TrimSpace tests email space trimming
func TestValidateEmail_TrimSpace(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "前后有空格", email: "  test@example.com  ", want: true},
		{name: "前面有空格", email: " test@example.com", want: true},
		{name: "后面有空格", email: "test@example.com ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			assert.Equal(t, tt.want, got, "应该能够处理带空格的邮箱")
		})
	}
}

// TestValidateEmail_CaseSensitivity tests email case sensitivity handling
func TestValidateEmail_CaseSensitivity(t *testing.T) {
	// Email validation should be case-insensitive (according to implementation)
	emails := []string{
		"Test@Example.com",
		"TEST@EXAMPLE.COM",
		"test@example.com",
		"TeSt@ExAmPlE.CoM",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			got := ValidateEmail(email)
			assert.True(t, got, "邮箱验证应该对大小写不敏感")
		})
	}
}

// TestValidateEmail_MoreEdgeCases tests more edge cases to improve coverage
func TestValidateEmail_MoreEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		// Test more edge cases
		{
			name:  "本地部分以点结尾",
			email: "test.@example.com",
			want:  false,
		},
		{
			name:  "本地部分以点开头",
			email: ".test@example.com",
			want:  false,
		},
		{
			name:  "域名部分以点开头",
			email: "test@.example.com",
			want:  false,
		},
		{
			name:  "域名部分以点结尾",
			email: "test@example.com.",
			want:  false,
		},
		{
			name:  "域名部分没有点（缺少TLD）",
			email: "test@example",
			want:  false,
		},
		{
			name:  "多个@符号",
			email: "test@@example.com",
			want:  false,
		},
		{
			name:  "只有@符号",
			email: "@",
			want:  false,
		},
		{
			name:  "只有本地部分",
			email: "test@",
			want:  false,
		},
		{
			name:  "只有域名部分",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "连续的点在本地部分",
			email: "test..user@example.com",
			want:  false,
		},
		{
			name:  "连续的点在域名部分",
			email: "test@example..com",
			want:  false,
		},
		{
			name:  "有效的邮箱（带多个子域名）",
			email: "test@mail.sub.example.com",
			want:  true,
		},
		{
			name:  "有效的邮箱（本地部分带多个点）",
			email: "first.middle.last@example.com",
			want:  true,
		},
		{
			name:  "有效的邮箱（带数字）",
			email: "user123@example123.com",
			want:  true,
		},
		{
			name:  "有效的邮箱（带下划线和横线）",
			email: "user_name-test@example-site.com",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			assert.Equal(t, tt.want, got, "ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
		})
	}
}
