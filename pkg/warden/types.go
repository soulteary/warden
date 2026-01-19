// Package warden provides a client SDK for interacting with Warden API.
package warden

// AllowListUser represents a user in the allow list.
type AllowListUser struct {
	Phone string `json:"phone"` // 用户手机号
	Mail  string `json:"mail"`  // 用户邮箱地址
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
