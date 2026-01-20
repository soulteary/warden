package define

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AllowListUser represents user information in the allow list.
//
// This struct is used to store user's basic information, including phone number, email address, user ID, status, etc.
// This information is used to verify and authorize user access.
//
//nolint:govet // fieldalignment: field order is affected by JSON serialization tags, optimization may break API compatibility
type AllowListUser struct {
	Phone  string   `json:"phone"`   // User phone number
	Mail   string   `json:"mail"`    // User email address
	UserID string   `json:"user_id"` // User unique identifier (optional, auto-generated if not provided)
	Status string   `json:"status"`  // User status (e.g., "active", "inactive", "suspended")
	Scope  []string `json:"scope"`   // User permission scope (optional)
	Role   string   `json:"role"`    // User role (optional)
}

// Normalize normalizes user data, sets default values and generates user_id (if not provided)
//
// This function will:
// - If user_id is empty, generate based on phone or mail
// - If status is empty, set to "active"
// - If scope is nil, set to empty array
// - If role is empty, set to empty string
func (u *AllowListUser) Normalize() {
	// Generate user_id (if not provided)
	if u.UserID == "" {
		identifier := strings.TrimSpace(u.Phone)
		if identifier == "" {
			identifier = strings.TrimSpace(strings.ToLower(u.Mail))
		}
		if identifier != "" {
			h := sha256.Sum256([]byte(identifier))
			u.UserID = hex.EncodeToString(h[:])[:16] // Take first 16 characters
		}
	}

	// Set default status
	if u.Status == "" {
		u.Status = "active"
	}

	// Set default scope (if nil)
	if u.Scope == nil {
		u.Scope = []string{}
	}

	// role can be empty string, no need to set default value
}
