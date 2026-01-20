// Package middleware 提供了 HTTP 中间件功能。
// 包括国际化语言检测中间件。
package middleware

import (
	// 标准库
	"net/http"
	"strings"

	// 项目内部包
	"github.com/soulteary/warden/internal/i18n"
)

// I18nMiddleware 创建国际化语言检测中间件
//
// 该中间件从 HTTP 请求中检测用户语言，优先级如下：
// 1. 查询参数 ?lang=xx
// 2. Accept-Language 请求头
// 3. 默认语言（英语）
//
// 检测到的语言会被存储到请求上下文中，供后续处理使用。
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func I18nMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 检测语言
			lang := detectLanguage(r)

			// 将语言设置到请求上下文中
			r = i18n.SetLanguageInContext(r, lang)

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// detectLanguage 从请求中检测语言
// 优先级：查询参数 > Accept-Language > 默认
func detectLanguage(r *http.Request) i18n.Language {
	// 1. 检查查询参数
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return i18n.NormalizeLanguage(lang)
	}

	// 2. 检查 Accept-Language 头
	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		return parseAcceptLanguage(acceptLang)
	}

	// 3. 默认英语
	return i18n.LangEN
}

// parseAcceptLanguage 解析 Accept-Language 请求头
// 支持格式：en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7
func parseAcceptLanguage(acceptLang string) i18n.Language {
	// 移除空格
	acceptLang = strings.ReplaceAll(acceptLang, " ", "")

	// 按逗号分割语言标签
	langs := strings.Split(acceptLang, ",")

	// 遍历语言标签，找到第一个支持的语言
	for _, langTag := range langs {
		// 移除质量值（q=0.9）
		langTag = strings.Split(langTag, ";")[0]
		langTag = strings.TrimSpace(langTag)

		// 规范化语言代码
		normalized := i18n.NormalizeLanguage(langTag)
		if normalized != i18n.LangEN || langTag == "en" || strings.HasPrefix(langTag, "en-") {
			// 如果规范化后不是默认值，或者确实是英语，返回该语言
			return normalized
		}
	}

	// 默认返回英语
	return i18n.LangEN
}

// GetLanguage 从请求中获取语言（辅助函数）
func GetLanguage(r *http.Request) i18n.Language {
	return i18n.GetLanguageFromContext(r)
}
