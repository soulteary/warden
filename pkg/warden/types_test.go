package warden

import "testing"

func TestAllowListUserStatus(t *testing.T) {
	user := AllowListUser{Status: "active"}
	if !user.IsActive() {
		t.Fatal("IsActive() should return true for active user")
	}
	if !user.IsValid() {
		t.Fatal("IsValid() should return true for active user")
	}

	user.Status = "inactive"
	if user.IsActive() {
		t.Fatal("IsActive() should return false for inactive user")
	}
	if user.IsValid() {
		t.Fatal("IsValid() should return false for inactive user")
	}
}
