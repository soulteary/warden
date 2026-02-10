package define

import (
	"strings"

	secure "github.com/soulteary/secure-kit"
)

// AllowListUser represents user information in the allow list.
//
// This struct is used to store user's basic information, including phone number, email address, user ID, status, etc.
// This information is used to verify and authorize user access.
//
// At least one of Phone or Mail must be non-empty. Email-only users are supported (Phone may be empty).
//
//nolint:govet // fieldalignment: field order is affected by JSON serialization tags, optimization may break API compatibility
type AllowListUser struct {
	Phone          string   `json:"phone"`                     // User phone number (optional if Mail is set)
	Mail           string   `json:"mail"`                      // User email address (optional if Phone is set)
	UserID         string   `json:"user_id"`                   // User unique identifier (optional, auto-generated if not provided)
	Status         string   `json:"status"`                    // User status (e.g., "active", "inactive", "suspended")
	Scope          []string `json:"scope"`                     // User permission scope (optional)
	Role           string   `json:"role"`                      // User role (optional)
	Name           string   `json:"name,omitempty"`            // User display name (optional)
	DingtalkUserID string   `json:"dingtalk_userid,omitempty"` // DingTalk user ID for work notification (optional)
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
			u.UserID = secure.GetSHA256Hash(identifier)[:16] // Take first 16 characters
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

// IsActive checks if the user status is active.
//
// Returns true if the user status is "active", false otherwise.
// This method is used to verify if a user is allowed to access the system.
func (u *AllowListUser) IsActive() bool {
	return u.Status == "active"
}

// IsValid checks if the user has a valid status for authentication.
//
// Returns true if the user status is one of the valid statuses (currently only "active").
// This method can be extended in the future to support other valid statuses if needed.
func (u *AllowListUser) IsValid() bool {
	validStatuses := []string{"active"}
	for _, status := range validStatuses {
		if u.Status == status {
			return true
		}
	}
	return false
}
