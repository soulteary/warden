// Package warden provides a client SDK for interacting with Warden API.
package warden

// AllowListUser represents a user in the allow list.
type AllowListUser struct {
	Phone  string   `json:"phone"`   // 用户手机号
	Mail   string   `json:"mail"`    // 用户邮箱地址
	UserID string   `json:"user_id"` // 用户唯一标识符（可选，如果未提供则自动生成）
	Status string   `json:"status"`  // 用户状态（如 "active", "inactive", "suspended"）
	Scope  []string `json:"scope"`   // 用户权限范围（可选）
	Role   string   `json:"role"`    // 用户角色（可选）
}

// PaginatedResponse represents a paginated response from the Warden API.
type PaginatedResponse struct {
	Data       []AllowListUser `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
}

// PaginationInfo contains pagination metadata.
type PaginationInfo struct {
	Page       int `json:"page"`        // 当前页码（从 1 开始）
	PageSize   int `json:"page_size"`   // 每页大小
	Total      int `json:"total"`       // 总记录数
	TotalPages int `json:"total_pages"` // 总页数
}
