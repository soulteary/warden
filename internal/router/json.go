// Package router provides HTTP routing functionality.
// Includes request logging, JSON responses, health checks and other route handlers.
package router

import (
	// Standard library
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"

	// Internal packages
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/metrics"
)

// bufferPool reuses bytes.Buffer objects
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// getBuffer gets a buffer from the pool
func getBuffer() *bytes.Buffer {
	buf, ok := bufferPool.Get().(*bytes.Buffer)
	if !ok {
		return &bytes.Buffer{}
	}
	return buf
}

// putBuffer returns a buffer to the pool
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// parsePaginationParams parses pagination parameters
// Returns page, pageSize, hasPagination, error
// hasPagination indicates whether pagination parameters are explicitly specified
// Enhanced input validation: limit parameter length, validate numeric range, prevent injection attacks
func parsePaginationParams(r *http.Request) (page, pageSize int, hasPagination bool, err error) {
	page = 1
	pageSize = define.DEFAULT_PAGE_SIZE

	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("page_size")

	// Check if pagination parameters are explicitly specified
	hasPagination = pageStr != "" || sizeStr != ""

	// Security validation: limit parameter length to prevent overly long input
	const maxParamLength = 20
	if len(pageStr) > maxParamLength || len(sizeStr) > maxParamLength {
		// Note: no request context here, use default language
		return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
	}

	// Security validation: check if parameters contain illegal characters (only digits allowed)
	if pageStr != "" {
		// Validate if it's pure digits
		for _, c := range pageStr {
			if c < '0' || c > '9' {
				return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
		}
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			// Limit maximum page number to prevent performance issues from overly large values
			const maxPage = 1000000
			if p > maxPage {
				return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
			page = p
		} else {
			return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
		}
	}

	if sizeStr != "" {
		// Validate if it's pure digits
		for _, c := range sizeStr {
			if c < '0' || c > '9' {
				return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
		}
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= define.MAX_PAGE_SIZE {
			pageSize = s
		} else {
			if s > define.MAX_PAGE_SIZE {
				return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
			}
			return 0, 0, false, errors.New(i18n.TWithLang(i18n.LangEN, "error.invalid_pagination"))
		}
	}

	return page, pageSize, hasPagination, nil
}

// paginate paginates data
func paginate(data []define.AllowListUser, page, pageSize int) (result []define.AllowListUser, total, totalPages int) {
	total = len(data)
	if total == 0 {
		return []define.AllowListUser{}, 0, 0
	}

	totalPages = (total + pageSize - 1) / pageSize

	// If the requested page is out of range, return empty array
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

// buildPaginatedResponse builds paginated response structure
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

// encodeJSONResponse encodes and writes JSON response
func encodeJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}) error {
	buf := getBuffer()
	defer putBuffer(buf)

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		logger.FromRequest(r).Error().
			Err(err).
			Msg(i18n.T(r, "error.json_encode_failed"))
		http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
		return err
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.FromRequest(r).Error().
			Err(err).
			Msg(i18n.T(r, "error.write_response_failed"))
		return err
	}

	return nil
}

// JSON returns a JSON response handler for user data
//
// This function creates an HTTP handler that returns JSON data from the user cache.
// Supports the following features:
// - Pagination: implemented via page and page_size query parameters
// - Backward compatibility: returns full array format when pagination parameters are not specified
// - Performance optimization: selects different encoding strategies based on data size (direct encoding, buffer pool, streaming encoding)
// - Input validation: strictly validates pagination parameters to prevent injection attacks
//
// Parameters:
//   - userCache: user cache instance for retrieving user data
//
// Returns:
//   - func(http.ResponseWriter, *http.Request): HTTP request handler function
//
// Side effects:
//   - Records cache hit metrics
//   - Records request logs
func JSON(userCache *cache.SafeUserCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate request method, only allow GET
		if r.Method != http.MethodGet {
			logger.FromRequest(r).Warn().
				Str("method", r.Method).
				Msg(i18n.T(r, "log.unsupported_method"))
			http.Error(w, i18n.T(r, "http.method_not_allowed"), http.StatusMethodNotAllowed)
			return
		}

		// Parse pagination parameters (enhanced input validation)
		page, pageSize, hasPagination, err := parsePaginationParams(r)
		if err != nil {
			logger.FromRequest(r).Warn().
				Err(err).
				Msg(i18n.T(r, "log.pagination_validation_failed"))
			http.Error(w, i18n.T(r, "http.invalid_pagination_parameters"), http.StatusBadRequest)
			return
		}

		// Get all data (from memory cache, record as cache hit)
		userData := userCache.Get()
		metrics.CacheHits.Inc() // Memory cache hit

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// If pagination parameters are not explicitly specified, maintain backward compatibility and return array directly
		if !hasPagination {
			// For small data, encode directly; for medium data, use bufferPool; for large data, use streaming encoding
			switch {
			case len(userData) < define.SMALL_DATA_THRESHOLD:
				if err := json.NewEncoder(w).Encode(userData); err != nil {
					logger.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.json_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
			case len(userData) < define.LARGE_DATA_THRESHOLD:
				// Medium data: use bufferPool for optimization
				buf := getBuffer()
				defer putBuffer(buf)
				if err := json.NewEncoder(buf).Encode(userData); err != nil {
					logger.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.json_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
				if _, err := w.Write(buf.Bytes()); err != nil {
					logger.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.write_response_failed"))
					return
				}
			default:
				// Large data: use streaming JSON encoding to reduce memory usage
				encoder := json.NewEncoder(w)
				if err := encoder.Encode(userData); err != nil {
					logger.FromRequest(r).Error().
						Err(err).
						Msg(i18n.T(r, "error.stream_encode_failed"))
					http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
					return
				}
			}
			logger.FromRequest(r).Info().Msg(i18n.T(r, "log.request_data_api"))
			return
		}

		// If pagination parameters are specified, return paginated format
		paginatedData, total, totalPages := paginate(userData, page, pageSize)
		response := buildPaginatedResponse(paginatedData, page, pageSize, total, totalPages)

		if err := encodeJSONResponse(w, r, response); err != nil {
			return
		}

		logger.FromRequest(r).Info().
			Int("page", page).
			Int("page_size", pageSize).
			Int("total", total).
			Msg(i18n.T(r, "log.request_data_api"))
	}
}

// WriteJSONError writes a JSON error response { "error": message } with the given status code.
func WriteJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		// Response already committed; log only
		logger.GetLoggerKit().Debug().Err(err).Str("message", message).Msg("failed to encode JSON error response")
	}
}
