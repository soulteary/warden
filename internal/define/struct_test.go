package define

import (
	"testing"
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
