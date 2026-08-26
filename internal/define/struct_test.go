package define

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowListUser_IsActive(t *testing.T) {
	tests := []struct {
		name string
		user AllowListUser
		want bool
	}{
		{
			name: "active user",
			user: AllowListUser{Status: "active"},
			want: true,
		},
		{
			name: "inactive user",
			user: AllowListUser{Status: "inactive"},
			want: false,
		},
		{
			name: "suspended user",
			user: AllowListUser{Status: "suspended"},
			want: false,
		},
		{
			name: "empty status",
			user: AllowListUser{Status: ""},
			want: false,
		},
		{
			name: "unknown status",
			user: AllowListUser{Status: "unknown"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsActive(); got != tt.want {
				t.Errorf("AllowListUser.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllowListUser_IsValid(t *testing.T) {
	tests := []struct {
		name string
		user AllowListUser
		want bool
	}{
		{
			name: "active user",
			user: AllowListUser{Status: "active"},
			want: true,
		},
		{
			name: "inactive user",
			user: AllowListUser{Status: "inactive"},
			want: false,
		},
		{
			name: "suspended user",
			user: AllowListUser{Status: "suspended"},
			want: false,
		},
		{
			name: "empty status",
			user: AllowListUser{Status: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsValid(); got != tt.want {
				t.Errorf("AllowListUser.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAllowListUser_Normalize tests Normalize method
func TestAllowListUser_Normalize(t *testing.T) {
	//nolint:govet // fieldalignment: test cases prioritize readability
	tests := []struct {
		name     string
		user     AllowListUser
		validate func(t *testing.T, user AllowListUser)
	}{
		{
			name: "已有user_id，不生成",
			user: AllowListUser{
				UserID: "existing-user-id",
				Phone:  "13800138000",
				Mail:   "test@example.com",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.Equal(t, "existing-user-id", user.UserID, "已有user_id不应该被覆盖")
			},
		},
		{
			name: "从phone生成user_id",
			user: AllowListUser{
				Phone: "13800138000",
				Mail:  "test@example.com",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.NotEmpty(t, user.UserID, "应该生成user_id")
				assert.Len(t, user.UserID, 16, "user_id应该是16个字符")
			},
		},
		{
			name: "从mail生成user_id（phone为空）",
			user: AllowListUser{
				Phone: "",
				Mail:  "test@example.com",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.NotEmpty(t, user.UserID, "应该从mail生成user_id")
				assert.Len(t, user.UserID, 16, "user_id应该是16个字符")
			},
		},
		{
			name: "空status设置默认值",
			user: AllowListUser{
				Phone:  "13800138000",
				Status: "",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.Equal(t, "active", user.Status, "空status应该设置为active")
			},
		},
		{
			name: "nil scope设置默认值",
			user: AllowListUser{
				Phone: "13800138000",
				Scope: nil,
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.NotNil(t, user.Scope, "nil scope应该设置为空数组")
				assert.Len(t, user.Scope, 0, "scope应该是空数组")
			},
		},
		{
			name: "所有字段都已有值",
			user: AllowListUser{
				UserID: "user-123",
				Phone:  "13800138000",
				Mail:   "test@example.com",
				Status: "inactive",
				Scope:  []string{"read", "write"},
				Role:   "admin",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.Equal(t, "user-123", user.UserID, "user_id不应该改变")
				assert.Equal(t, "inactive", user.Status, "status不应该改变")
				assert.Equal(t, []string{"read", "write"}, user.Scope, "scope不应该改变")
				assert.Equal(t, "admin", user.Role, "role不应该改变")
			},
		},
		{
			name: "phone和mail都为空，不生成user_id",
			user: AllowListUser{
				Phone: "",
				Mail:  "",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.Empty(t, user.UserID, "phone和mail都为空时不应该生成user_id")
			},
		},
		{
			name: "phone有空格，生成user_id",
			user: AllowListUser{
				Phone: "  13800138000  ",
				Mail:  "",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.NotEmpty(t, user.UserID, "应该从去除空格的phone生成user_id")
			},
		},
		{
			name: "mail有空格和大写，生成user_id",
			user: AllowListUser{
				Phone: "",
				Mail:  "  Test@Example.COM  ",
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.NotEmpty(t, user.UserID, "应该从mail生成user_id")
			},
		},
		{
			name: "已有scope，不改变",
			user: AllowListUser{
				Phone: "13800138000",
				Scope: []string{"read"},
			},
			validate: func(t *testing.T, user AllowListUser) {
				assert.Equal(t, []string{"read"}, user.Scope, "已有scope不应该改变")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.user.Normalize()
			tt.validate(t, tt.user)
		})
	}
}

// TestAllowListUser_Normalize_Consistency tests that Normalize produces consistent results
func TestAllowListUser_Normalize_Consistency(t *testing.T) {
	user1 := AllowListUser{
		Phone: "13800138000",
		Mail:  "test@example.com",
	}
	user2 := AllowListUser{
		Phone: "13800138000",
		Mail:  "test@example.com",
	}

	user1.Normalize()
	user2.Normalize()

	assert.Equal(t, user1.UserID, user2.UserID, "相同输入应该生成相同的user_id")
}

// TestAllowListUser_Normalize_DifferentPhone tests that different phones generate different user_ids
func TestAllowListUser_Normalize_DifferentPhone(t *testing.T) {
	user1 := AllowListUser{Phone: "13800138000"}
	user2 := AllowListUser{Phone: "13900139000"}

	user1.Normalize()
	user2.Normalize()

	assert.NotEqual(t, user1.UserID, user2.UserID, "不同phone应该生成不同的user_id")
}

// TestAllowListUser_Normalize_DifferentMail tests that different mails generate different user_ids
func TestAllowListUser_Normalize_DifferentMail(t *testing.T) {
	user1 := AllowListUser{Mail: "test1@example.com"}
	user2 := AllowListUser{Mail: "test2@example.com"}

	user1.Normalize()
	user2.Normalize()

	assert.NotEqual(t, user1.UserID, user2.UserID, "不同mail应该生成不同的user_id")
}
