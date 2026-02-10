// Package router provides HTTP routing functionality.
// response.go: response field whitelist mapping for AllowListUser.
package router

import (
	"github.com/soulteary/warden/internal/define"
)

// UserToMap converts AllowListUser to a map for JSON response.
// If fields is nil or empty, all known fields are included.
// Otherwise only keys in fields are included (whitelist).
func UserToMap(u *define.AllowListUser, fields []string) map[string]interface{} {
	m := map[string]interface{}{
		"phone":   u.Phone,
		"mail":    u.Mail,
		"user_id": u.UserID,
		"status":  u.Status,
		"scope":   u.Scope,
		"role":    u.Role,
		"name":    u.Name,
	}
	if u.DingtalkUserID != "" {
		m["dingtalk_userid"] = u.DingtalkUserID
	}
	if len(fields) == 0 {
		return m
	}
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		set[f] = true
	}
	out := make(map[string]interface{}, len(fields))
	for k, v := range m {
		if set[k] {
			out[k] = v
		}
	}
	return out
}

// UsersToMaps converts a slice of AllowListUser to slice of maps with optional field whitelist.
func UsersToMaps(users []define.AllowListUser, fields []string) []map[string]interface{} {
	if len(users) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, len(users))
	for i := range users {
		out[i] = UserToMap(&users[i], fields)
	}
	return out
}
