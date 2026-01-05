// Package define 定义了应用中的常量和数据结构。
// 包括默认配置值、超时时间、限制参数等常量定义。
package define

import (
	// 标准库
	"time"
)

// DefaultPort 默认端口号
const DefaultPort = 8081

// DefaultRedis 默认 Redis 地址
const DefaultRedis = "localhost:6379"

// DefaultRemoteConfig 默认远程配置地址
const DefaultRemoteConfig = "http://localhost:8080/config.json"

// DefaultRemoteKey 默认远程配置密钥
const DefaultRemoteKey = ""

const (
	// DefaultTaskInterval 默认任务间隔时间（秒）
	DefaultTaskInterval = 5 // 5s
	// DefaultTimeout 默认超时时间（秒）
	DefaultTimeout = 5
	// DefaultLockTime 默认锁过期时间（秒）
	DefaultLockTime = 15
	// DefaultMode 默认模式
	DefaultMode = "DEFAULT" // 1: 2: 3: 4:

	// RateLimitCleanupInterval 速率限制器清理间隔
	RateLimitCleanupInterval = 1 * time.Minute

	// DefaultPageSize 默认每页大小
	DefaultPageSize = 100
	// MaxPageSize 最大每页大小
	MaxPageSize = 1000

	// MaxHeaderBytes 最大请求头大小（1MB）
	MaxHeaderBytes = 1 << 20
	// MAX_REQUEST_BODY_SIZE 最大请求体大小（10KB）
	MAX_REQUEST_BODY_SIZE = 10 * 1024
	// SHUTDOWN_TIMEOUT 优雅关闭超时时间
	SHUTDOWN_TIMEOUT = 5 * time.Second
	// HTTP_RETRY_MAX_RETRIES HTTP 请求最大重试次数
	HTTP_RETRY_MAX_RETRIES = 3
	// HTTP_RETRY_DELAY HTTP 请求重试延迟
	HTTP_RETRY_DELAY = 1 * time.Second
	// IDLE_TIMEOUT HTTP 连接空闲超时时间
	IDLE_TIMEOUT = 120 * time.Second

	// REDIS_CONNECTION_TIMEOUT Redis 连接超时时间
	REDIS_CONNECTION_TIMEOUT = 5 * time.Second

	// DEFAULT_RATE_LIMIT 默认速率限制：每分钟请求数
	DEFAULT_RATE_LIMIT = 60
	// DEFAULT_RATE_LIMIT_WINDOW 默认速率限制时间窗口
	DEFAULT_RATE_LIMIT_WINDOW = 1 * time.Minute
	// MAX_VISITORS_MAP_SIZE 最大访问者 map 大小，防止内存泄漏
	MAX_VISITORS_MAP_SIZE = 10000
	// MAX_WHITELIST_SIZE 最大白名单大小
	MAX_WHITELIST_SIZE = 1000

	// DEFAULT_MAX_IDLE_CONNS 默认最大空闲连接数
	DEFAULT_MAX_IDLE_CONNS = 100
	// DEFAULT_MAX_IDLE_CONNS_PER_HOST 默认每个主机的最大空闲连接数
	DEFAULT_MAX_IDLE_CONNS_PER_HOST = 10
	// DEFAULT_IDLE_CONN_TIMEOUT 默认空闲连接超时时间
	DEFAULT_IDLE_CONN_TIMEOUT = 90 * time.Second
	// DEFAULT_LOAD_DATA_TIMEOUT 默认加载数据超时时间
	DEFAULT_LOAD_DATA_TIMEOUT = 30 * time.Second

	// DEFAULT_SLICE_POOL_CAPACITY 默认切片池容量
	DEFAULT_SLICE_POOL_CAPACITY = 100
	// SMALL_DATA_THRESHOLD 小数据阈值，小于此值的数据直接处理，不使用缓冲池
	SMALL_DATA_THRESHOLD = 100
	// LARGE_DATA_THRESHOLD 大数据阈值，大于此值的数据使用流式 JSON 编码
	LARGE_DATA_THRESHOLD = 10000

	// REDIS_RETRY_MAX_RETRIES Redis 操作最大重试次数
	REDIS_RETRY_MAX_RETRIES = 3
	// REDIS_RETRY_DELAY Redis 操作重试延迟
	REDIS_RETRY_DELAY = 1 * time.Second
)

const (
	// WARN_RULE_NOT_FOUND 没有找到规则文件
	WARN_RULE_NOT_FOUND = "没有找到规则文件"
	// WARN_READ_RULE_ERR 读取规则文件遇到错误
	WARN_READ_RULE_ERR = "读取规则文件遇到错误"
	// WARN_PARSE_RULE_ERR 解析规则文件遇到错误
	WARN_PARSE_RULE_ERR = "解析规则文件遇到错误"
	// ERROR_CAN_NOT_OPEN_RULE 读取规则文件出错
	ERROR_CAN_NOT_OPEN_RULE = "读取规则文件出错"

	// ERR_REQ_INIT_FAILED 网络请求组件初始化失败
	ERR_REQ_INIT_FAILED = "网络请求组件初始化失败"
	// ERR_GET_CONFIG_FAILED 获取远程配置失败
	ERR_GET_CONFIG_FAILED = "获取远程配置失败"
	// ERR_READ_CONFIG_FAILED 读取远程配置失败
	ERR_READ_CONFIG_FAILED = "读取远程配置失败"
	// ERR_PARSE_CONFIG_FAILED 解析远程配置失败
	ERR_PARSE_CONFIG_FAILED = "解析远程配置失败"

	// WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL 获取远程规则出错，仅使用本地规则
	WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL = "获取远程规则出错，仅使用本地规则"

	// INFO_REQ_REMOTE_API 请求数据接口
	INFO_REQ_REMOTE_API = "请求数据接口 🎩"
)
