package define

// AllowListUser 表示允许列表中的用户信息。
//
// 该结构体用于存储用户的基本信息，包括手机号和邮箱地址。
// 这些信息用于验证和授权用户访问。
type AllowListUser struct {
	Phone string `json:"phone"` // 用户手机号
	Mail  string `json:"mail"`  // 用户邮箱地址
}
