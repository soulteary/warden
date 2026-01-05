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
	DefaultMode     = "DEFAULT" // 1: 2: 3: 4:

	// RateLimitCleanupInterval 速率限制器清理间隔
	RateLimitCleanupInterval = 1 * time.Minute

	// DefaultPageSize 默认每页大小
	DefaultPageSize = 100
	// MaxPageSize 最大每页大小
	MaxPageSize = 1000

	// MaxHeaderBytes 最大请求头大小（1MB）
	MaxHeaderBytes = 1 << 20
	// MaxRequestBodySize 最大请求体大小（10KB）
	MaxRequestBodySize = 10 * 1024
	// ShutdownTimeout 优雅关闭超时时间
	ShutdownTimeout = 5 * time.Second
	// HTTPRetryMaxRetries HTTP 请求最大重试次数
	HTTPRetryMaxRetries = 3
	// HTTPRetryDelay HTTP 请求重试延迟
	HTTPRetryDelay = 1 * time.Second
	// IdleTimeout HTTP 连接空闲超时时间
	IdleTimeout = 120 * time.Second

	// RedisConnectionTimeout Redis 连接超时时间
	RedisConnectionTimeout = 5 * time.Second

	// DefaultRateLimit 默认速率限制：每分钟请求数
	DefaultRateLimit = 60
	// DefaultRateLimitWindow 默认速率限制时间窗口
	DefaultRateLimitWindow = 1 * time.Minute
	// MaxVisitorsMapSize 最大访问者 map 大小，防止内存泄漏
	MaxVisitorsMapSize = 10000
	// MaxWhitelistSize 最大白名单大小
	MaxWhitelistSize = 1000

	// DefaultMaxIdleConns 默认最大空闲连接数
	DefaultMaxIdleConns = 100
	// DefaultMaxIdleConnsPerHost 默认每个主机的最大空闲连接数
	DefaultMaxIdleConnsPerHost = 10
	// DefaultIdleConnTimeout 默认空闲连接超时时间
	DefaultIdleConnTimeout = 90 * time.Second
	// DefaultLoadDataTimeout 默认加载数据超时时间
	DefaultLoadDataTimeout = 30 * time.Second

	// DefaultSlicePoolCapacity 默认切片池容量
	DefaultSlicePoolCapacity = 100
	// SmallDataThreshold 小数据阈值，小于此值的数据直接处理，不使用缓冲池
	SmallDataThreshold = 100
	// LargeDataThreshold 大数据阈值，大于此值的数据使用流式 JSON 编码
	LargeDataThreshold = 10000

	// RedisRetryMaxRetries Redis 操作最大重试次数
	RedisRetryMaxRetries = 3
	// RedisRetryDelay Redis 操作重试延迟
	RedisRetryDelay = 1 * time.Second
)

const (
	// WarnRuleNotFound 没有找到规则文件
	WarnRuleNotFound = "没有找到规则文件"
	// WarnReadRuleErr 读取规则文件遇到错误
	WarnReadRuleErr = "读取规则文件遇到错误"
	// WarnParseRuleErr 解析规则文件遇到错误
	WarnParseRuleErr = "解析规则文件遇到错误"
	// ErrorCanNotOpenRule 读取规则文件出错
	ErrorCanNotOpenRule = "读取规则文件出错"

	// ErrReqInitFailed 网络请求组件初始化失败
	ErrReqInitFailed = "网络请求组件初始化失败"
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
