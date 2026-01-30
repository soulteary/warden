// Package define defines constants and data structures in the application.
// Includes default configuration values, timeout durations, limit parameters and other constant definitions.
//
//nolint:revive // Constants use ALL_CAPS naming which conforms to project standards (see CODE_STYLE.md)
package define

import (
	// Standard library
	"strings"
	"time"
)

// DEFAULT_PORT default port number
const DEFAULT_PORT = 8081

// DEFAULT_REDIS default Redis address
const DEFAULT_REDIS = "localhost:6379"

// DEFAULT_REMOTE_CONFIG default remote configuration address
const DEFAULT_REMOTE_CONFIG = "http://localhost:8080/data.json"

// DEFAULT_REMOTE_KEY default remote configuration key
const DEFAULT_REMOTE_KEY = ""

// DEFAULT_DATA_FILE default local user data file path
const DEFAULT_DATA_FILE = "./data.json"

// HTTP path constants. Used for route registration, rate-limit skip paths, and access-log skip paths.
const (
	PATH_HEALTH      = "/health"
	PATH_HEALTHCHECK = "/healthcheck"
	PATH_METRICS     = "/metrics"
)

// SkipPathsHealthAndMetrics is the path list to skip for rate limiting and access logging.
// Reduces log noise and keeps health/metrics probes from consuming rate limit.
var SkipPathsHealthAndMetrics = []string{PATH_HEALTH, PATH_HEALTHCHECK, PATH_METRICS}

// ParseTrustedProxyIPs parses comma-separated string (e.g. TRUSTED_PROXY_IPS env), trims each element, drops empty.
// Used by main and ip_whitelist so env parsing is explicit and consistent.
func ParseTrustedProxyIPs(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

const (
	// DEFAULT_TASK_INTERVAL default task interval (seconds)
	DEFAULT_TASK_INTERVAL = 5 // 5s
	// DEFAULT_TIMEOUT default timeout (seconds)
	DEFAULT_TIMEOUT = 5
	// DEFAULT_LOCK_TIME default lock expiration time (seconds)
	DEFAULT_LOCK_TIME = 15
	// DEFAULT_MODE default mode
	DEFAULT_MODE = "DEFAULT" // 1: 2: 3: 4:

	// RATE_LIMIT_CLEANUP_INTERVAL rate limiter cleanup interval
	RATE_LIMIT_CLEANUP_INTERVAL = 1 * time.Minute

	// DEFAULT_PAGE_SIZE default page size
	DEFAULT_PAGE_SIZE = 100
	// MAX_PAGE_SIZE maximum page size
	MAX_PAGE_SIZE = 1000
	// MAX_IDENTIFIER_LENGTH maximum length for user query params (phone/mail/user_id), prevents DoS and log/cache bloat
	MAX_IDENTIFIER_LENGTH = 512

	// MAX_HEADER_BYTES maximum request header size (1MB)
	MAX_HEADER_BYTES = 1 << 20
	// MAX_REQUEST_BODY_SIZE maximum request body size (10KB)
	MAX_REQUEST_BODY_SIZE = 10 * 1024
	// MAX_JSON_SIZE maximum JSON response body size (10MB), prevents memory exhaustion attacks
	MAX_JSON_SIZE = 10 * 1024 * 1024
	// SHUTDOWN_TIMEOUT graceful shutdown timeout
	SHUTDOWN_TIMEOUT = 5 * time.Second
	// HTTP_RETRY_MAX_RETRIES HTTP request maximum retry count
	HTTP_RETRY_MAX_RETRIES = 3
	// HTTP_RETRY_DELAY HTTP request retry delay
	HTTP_RETRY_DELAY = 1 * time.Second
	// IDLE_TIMEOUT HTTP connection idle timeout
	IDLE_TIMEOUT = 120 * time.Second

	// REDIS_CONNECTION_TIMEOUT Redis connection timeout
	REDIS_CONNECTION_TIMEOUT = 5 * time.Second

	// DEFAULT_RATE_LIMIT default rate limit: requests per minute
	DEFAULT_RATE_LIMIT = 60
	// DEFAULT_RATE_LIMIT_WINDOW default rate limit time window
	DEFAULT_RATE_LIMIT_WINDOW = 1 * time.Minute
	// MAX_VISITORS_MAP_SIZE maximum visitors map size, prevents memory leaks
	MAX_VISITORS_MAP_SIZE = 10000
	// MAX_WHITELIST_SIZE maximum whitelist size
	MAX_WHITELIST_SIZE = 1000

	// DEFAULT_MAX_IDLE_CONNS default maximum idle connections
	DEFAULT_MAX_IDLE_CONNS = 100
	// DEFAULT_MAX_IDLE_CONNS_PER_HOST default maximum idle connections per host
	DEFAULT_MAX_IDLE_CONNS_PER_HOST = 10
	// DEFAULT_IDLE_CONN_TIMEOUT default idle connection timeout
	DEFAULT_IDLE_CONN_TIMEOUT = 90 * time.Second
	// DEFAULT_LOAD_DATA_TIMEOUT default data loading timeout
	DEFAULT_LOAD_DATA_TIMEOUT = 30 * time.Second

	// DEFAULT_SLICE_POOL_CAPACITY default slice pool capacity
	DEFAULT_SLICE_POOL_CAPACITY = 100
	// SMALL_DATA_THRESHOLD small data threshold, data smaller than this is processed directly without using buffer pool
	SMALL_DATA_THRESHOLD = 100
	// LARGE_DATA_THRESHOLD large data threshold, data larger than this uses streaming JSON encoding
	LARGE_DATA_THRESHOLD = 10000

	// REDIS_RETRY_MAX_RETRIES Redis operation maximum retry count
	REDIS_RETRY_MAX_RETRIES = 3
	// REDIS_RETRY_DELAY Redis operation retry delay
	REDIS_RETRY_DELAY = 1 * time.Second
)

const (
	// WARN_RULE_NOT_FOUND rules file not found
	// Note: This constant is deprecated, please use i18n.T(r, "log.data_file_not_found") instead
	WARN_RULE_NOT_FOUND = "没有找到规则文件"
	// WARN_READ_RULE_ERR error reading rules file
	// Note: This constant is deprecated, please use i18n.T(r, "error.data_load_failed") instead
	WARN_READ_RULE_ERR = "读取规则文件遇到错误"
	// WARN_PARSE_RULE_ERR error parsing rules file
	// Note: This constant is deprecated, please use i18n.T(r, "error.data_parse_failed") instead
	WARN_PARSE_RULE_ERR = "解析规则文件遇到错误"
	// ERROR_CAN_NOT_OPEN_RULE error opening rules file
	// Note: This constant is deprecated, please use i18n.T(r, "error.data_load_failed") instead
	ERROR_CAN_NOT_OPEN_RULE = "读取规则文件出错"

	// ERR_REQ_INIT_FAILED network request component initialization failed
	// Note: This constant is deprecated, please use i18n.T(r, "error.http_request_failed") instead
	ERR_REQ_INIT_FAILED = "网络请求组件初始化失败"
	// ERR_GET_CONFIG_FAILED failed to get remote configuration
	// Note: This constant is deprecated, please use i18n.T(r, "error.config_load_failed") instead
	ERR_GET_CONFIG_FAILED = "获取远程配置失败"
	// ERR_READ_CONFIG_FAILED failed to read remote configuration
	// Note: This constant is deprecated, please use i18n.T(r, "error.config_load_failed") instead
	ERR_READ_CONFIG_FAILED = "读取远程配置失败"
	// ERR_PARSE_CONFIG_FAILED failed to parse remote configuration
	// Note: This constant is deprecated, please use i18n.T(r, "error.config_parse_failed") instead
	ERR_PARSE_CONFIG_FAILED = "解析远程配置失败"

	// WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL error getting remote rules, only using local rules
	// Note: This constant is deprecated, please use i18n.T(r, "log.all_sources_failed") instead
	WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL = "获取远程规则出错，仅使用本地规则"

	// INFO_REQ_REMOTE_API request data API
	// Note: This constant is deprecated, please use i18n.T(r, "log.request_data_api") instead
	INFO_REQ_REMOTE_API = "请求数据接口 🎩"
)
