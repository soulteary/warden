package router

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/soulteary/warden/internal/define"
)

func TestUserToMap_AllFields(t *testing.T) {
	u := &define.AllowListUser{
		Phone:          "13800138000",
		Mail:           "a@b.com",
		UserID:         "user1",
		Status:         "active",
		Scope:          []string{"read"},
		Role:           "user",
		Name:           "Test",
		DingtalkUserID: "dt123",
	}
	// nil or empty fields => all fields
	m := UserToMap(u, nil)
	assert.Len(t, m, 8)
	assert.Equal(t, "13800138000", m["phone"])
	assert.Equal(t, "a@b.com", m["mail"])
	assert.Equal(t, "user1", m["user_id"])
	assert.Equal(t, "active", m["status"])
	assert.Equal(t, []string{"read"}, m["scope"])
	assert.Equal(t, "user", m["role"])
	assert.Equal(t, "Test", m["name"])
	assert.Equal(t, "dt123", m["dingtalk_userid"])

	m2 := UserToMap(u, []string{})
	assert.Len(t, m2, 8)
}

func TestUserToMap_Whitelist(t *testing.T) {
	u := &define.AllowListUser{
		Phone: "13800138000", Mail: "a@b.com", UserID: "user1",
	}
	m := UserToMap(u, []string{"phone", "mail"})
	assert.Len(t, m, 2)
	assert.Equal(t, "13800138000", m["phone"])
	assert.Equal(t, "a@b.com", m["mail"])
	_, ok := m["user_id"]
	assert.False(t, ok)
}

func TestUserToMap_NoDingtalk(t *testing.T) {
	u := &define.AllowListUser{Phone: "13800138000", Mail: "a@b.com"}
	m := UserToMap(u, nil)
	assert.NotContains(t, m, "dingtalk_userid")
}

func TestUsersToMaps_Empty(t *testing.T) {
	out := UsersToMaps(nil, nil)
	assert.Nil(t, out)
	out2 := UsersToMaps([]define.AllowListUser{}, []string{"phone"})
	assert.Nil(t, out2)
}

func TestUsersToMaps_WithFields(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "a@example.com", UserID: "u1"},
		{Phone: "13900139000", Mail: "b@example.com", UserID: "u2"},
	}
	out := UsersToMaps(users, []string{"phone", "user_id"})
	assert.Len(t, out, 2)
	assert.Len(t, out[0], 2)
	assert.Equal(t, "13800138000", out[0]["phone"])
	assert.Equal(t, "u1", out[0]["user_id"])
	assert.Len(t, out[1], 2)
	assert.Equal(t, "13900139000", out[1]["phone"])
	assert.Equal(t, "u2", out[1]["user_id"])
}
