// Package warden provides a client SDK for interacting with Warden API.
package warden

// AllowListUser represents a user in the allow list.
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

// PaginatedResponse represents a paginated response from the Warden API.
type PaginatedResponse struct {
	Data       []AllowListUser `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
}

// PaginationInfo contains pagination metadata.
type PaginationInfo struct {
	Page       int `json:"page"`        // Current page number (starts from 1)
	PageSize   int `json:"page_size"`   // Page size
	Total      int `json:"total"`       // Total number of records
	TotalPages int `json:"total_pages"` // Total number of pages
}
