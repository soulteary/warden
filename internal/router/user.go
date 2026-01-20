// Package router 提供了 HTTP 路由处理功能。
// 包括用户查询、JSON 响应等路由处理器。
package router

import (
	// 标准库
	"encoding/json"
	"net/http"
	"strings"

	// 第三方库
	"github.com/rs/zerolog/hlog"

	// 项目内部包
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
)

// GetUserByIdentifier 根据标识符查询单个用户
//
// 该函数创建一个 HTTP 处理器，用于根据 phone、mail 或 user_id 查询单个用户。
// 支持以下查询方式：
// - GET /user?phone=<phone> - 通过手机号查询
// - GET /user?mail=<mail> - 通过邮箱查询
// - GET /user?user_id=<user_id> - 通过用户 ID 查询
//
// 参数:
//   - userCache: 用户缓存实例，用于获取用户数据
//
// 返回:
//   - func(http.ResponseWriter, *http.Request): HTTP 请求处理函数
func GetUserByIdentifier(userCache *cache.SafeUserCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法，只允许 GET
		if r.Method != http.MethodGet {
			hlog.FromRequest(r).Warn().
				Str("method", r.Method).
				Msg(i18n.T(r, "log.unsupported_method"))
			http.Error(w, i18n.T(r, "http.method_not_allowed"), http.StatusMethodNotAllowed)
			return
		}

		// 获取查询参数
		phone := strings.TrimSpace(r.URL.Query().Get("phone"))
		mail := strings.TrimSpace(r.URL.Query().Get("mail"))
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))

		// 验证至少提供一个标识符
		if phone == "" && mail == "" && userID == "" {
			hlog.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.missing_query_params"))
			http.Error(w, i18n.T(r, "error.missing_identifier"), http.StatusBadRequest)
			return
		}

		// 验证只提供一个标识符
		identifierCount := 0
		if phone != "" {
			identifierCount++
		}
		if mail != "" {
			identifierCount++
		}
		if userID != "" {
			identifierCount++
		}
		if identifierCount > 1 {
			hlog.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.multiple_query_params"))
			http.Error(w, i18n.T(r, "error.multiple_identifiers"), http.StatusBadRequest)
			return
		}

		var user define.AllowListUser
		var found bool

		// 根据标识符类型查询用户
		switch {
		case phone != "":
			user, found = userCache.GetByPhone(phone)
		case mail != "":
			user, found = userCache.GetByMail(mail)
		case userID != "":
			user, found = userCache.GetByUserID(userID)
		}

		// 如果用户不存在，返回 404
		if !found {
			hlog.FromRequest(r).Info().
				Str("phone", phone).
				Str("mail", mail).
				Str("user_id", userID).
				Msg(i18n.T(r, "log.user_not_found"))
			http.Error(w, i18n.T(r, "http.user_not_found"), http.StatusNotFound)
			return
		}

		// 返回用户信息
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(user); err != nil {
			hlog.FromRequest(r).Error().
				Err(err).
				Msg(i18n.T(r, "error.json_encode_failed"))
			http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
			return
		}

		hlog.FromRequest(r).Info().
			Str("user_id", user.UserID).
			Str("phone", user.Phone).
			Str("mail", user.Mail).
			Msg(i18n.T(r, "log.user_query_success"))
	}
}
