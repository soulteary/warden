// Package router 提供了 HTTP 路由处理功能。
// 包括请求日志记录、JSON 响应、健康检查等路由处理器。
package router

import (
	// 标准库
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	// 第三方库
	"github.com/rs/zerolog/hlog"

	// 项目内部包
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/metrics"
)

// bufferPool 复用 bytes.Buffer 对象
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// getBuffer 从池中获取 buffer
func getBuffer() *bytes.Buffer {
	buf, ok := bufferPool.Get().(*bytes.Buffer)
	if !ok {
		return &bytes.Buffer{}
	}
	return buf
}

// putBuffer 将 buffer 放回池中
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// parsePaginationParams 解析分页参数
// 返回 page, pageSize, hasPagination, error
// hasPagination 表示是否显式指定了分页参数
// 加强输入验证：限制参数长度、验证数值范围、防止注入攻击
func parsePaginationParams(r *http.Request) (page, pageSize int, hasPagination bool, err error) {
	page = 1
	pageSize = define.DEFAULT_PAGE_SIZE

	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("page_size")

	// 检查是否显式指定了分页参数
	hasPagination = pageStr != "" || sizeStr != ""

	// 安全验证：限制参数长度，防止过长的输入
	const maxParamLength = 20
	if len(pageStr) > maxParamLength || len(sizeStr) > maxParamLength {
		// 注意：这里没有请求上下文，使用默认语言
		return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
	}

	// 安全验证：检查参数是否包含非法字符（只允许数字）
	if pageStr != "" {
		// 验证是否为纯数字
		for _, c := range pageStr {
			if c < '0' || c > '9' {
				return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
		}
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			// 限制最大页码，防止过大的值导致性能问题
			const maxPage = 1000000
			if p > maxPage {
				return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
			page = p
		} else {
			return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
		}
	}

	if sizeStr != "" {
		// 验证是否为纯数字
		for _, c := range sizeStr {
			if c < '0' || c > '9' {
				return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
		}
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= define.MAX_PAGE_SIZE {
			pageSize = s
		} else {
			if s > define.MAX_PAGE_SIZE {
				return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
			return 0, 0, false, fmt.Errorf(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
		}
	}

	return page, pageSize, hasPagination, nil
}

// paginate 对数据进行分页
func paginate(data []define.AllowListUser, page, pageSize int) (result []define.AllowListUser, total, totalPages int) {
	total = len(data)
	if total == 0 {
		return []define.AllowListUser{}, 0, 0
	}

	totalPages = (total + pageSize - 1) / pageSize

	// 如果请求的页面超出范围，返回空数组
	if page > totalPages || page < 1 {
		return []define.AllowListUser{}, total, totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	if start >= total {
		return []define.AllowListUser{}, total, totalPages
	}

	return data[start:end], total, totalPages
}

// buildPaginatedResponse 构建分页响应结构
func buildPaginatedResponse(data []define.AllowListUser, page, pageSize, total, totalPages int) map[string]interface{} {
	return map[string]interface{}{
		"data": data,
		"pagination": map[string]int{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}

// encodeJSONResponse 编码并写入 JSON 响应
func encodeJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}) error {
	buf := getBuffer()
	defer putBuffer(buf)

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Msg(i18n.T(r, "error.json_encode_failed"))
		http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
		return err
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Msg(i18n.T(r, "error.write_response_failed"))
		return err
	}

	return nil
}

// JSON 返回用户数据的 JSON 响应处理器
//
// 该函数创建一个 HTTP 处理器，用于返回用户缓存中的 JSON 数据。
// 支持以下特性：
// - 分页支持：通过 page 和 page_size 查询参数实现分页
// - 向后兼容：未指定分页参数时返回完整数组格式
// - 性能优化：根据数据大小选择不同的编码策略（直接编码、缓冲池、流式编码）
// - 输入验证：严格验证分页参数，防止注入攻击
//
// 参数:
//   - userCache: 用户缓存实例，用于获取用户数据
//
// 返回:
//   - func(http.ResponseWriter, *http.Request): HTTP 请求处理函数
//
// 副作用:
//   - 记录缓存命中指标
//   - 记录请求日志
func JSON(userCache *cache.SafeUserCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法，只允许 GET
		if r.Method != http.MethodGet {
			hlog.FromRequest(r).Warn().
				Str("method", r.Method).
				Msg(i18n.T(r, "log.unsupported_method"))
			http.Error(w, i18n.T(r, "http.method_not_allowed"), http.StatusMethodNotAllowed)
			return
		}

		// 解析分页参数（加强输入验证）
		page, pageSize, hasPagination, err := parsePaginationParams(r)
		if err != nil {
			hlog.FromRequest(r).Warn().
				Err(err).
				Msg(i18n.T(r, "log.pagination_validation_failed"))
			http.Error(w, i18n.T(r, "http.invalid_pagination_parameters"), http.StatusBadRequest)
			return
		}

		// 获取所有数据（从内存缓存，记录为缓存命中）
		userData := userCache.Get()
		metrics.CacheHits.Inc() // 内存缓存命中

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// 如果没有显式指定分页参数，保持向后兼容，直接返回数组
		if !hasPagination {
			// 对于小数据，直接编码；对于中等数据，使用 bufferPool；对于大数据，使用流式编码
			switch {
			case len(userData) < define.SMALL_DATA_THRESHOLD:
				if err := json.NewEncoder(w).Encode(userData); err != nil {
					hlog.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.json_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
			case len(userData) < define.LARGE_DATA_THRESHOLD:
				// 中等数据：使用 bufferPool 优化
				buf := getBuffer()
				defer putBuffer(buf)
				if err := json.NewEncoder(buf).Encode(userData); err != nil {
					hlog.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.json_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
				if _, err := w.Write(buf.Bytes()); err != nil {
					hlog.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.write_response_failed"))
					return
				}
			default:
				// 大数据：使用流式 JSON 编码，减少内存占用
				encoder := json.NewEncoder(w)
				if err := encoder.Encode(userData); err != nil {
					hlog.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.stream_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
			}
			hlog.FromRequest(r).Info().Msg(i18n.T(r, "log.request_data_api"))
			return
		}

		// 如果指定了分页参数，返回分页格式
		paginatedData, total, totalPages := paginate(userData, page, pageSize)
		response := buildPaginatedResponse(paginatedData, page, pageSize, total, totalPages)

		if err := encodeJSONResponse(w, r, response); err != nil {
			return
		}

		hlog.FromRequest(r).Info().
			Int("page", page).
			Int("page_size", pageSize).
			Int("total", total).
			Msg(i18n.T(r, "log.request_data_api"))
	}
}
