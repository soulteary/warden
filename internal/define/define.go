// Package define 定义了应用中的常量和数据结构。
// 包括默认配置值、超时时间、限制参数等常量定义。
package define

import (
	// 标准库
	"time"
)

const (
	DEFAULT_PORT          = 8081
	DEFAULT_REDIS         = "localhost:6379"
	DEFAULT_REMOTE_CONFIG = "http://localhost:8080/config.json"
	DEFAULT_REMOTE_KEY    = ""
	DEFAULT_TASK_INTERVAL = 5 // 5s
	DEFAULT_TIMEOUT       = 5
	DEFAULT_LOCK_TIME     = 15
	DEFAULT_MODE          = "DEFAULT" // 1: 2: 3: 4:

	// 速率限制相关常量
	RateLimitCleanupInterval = 1 * time.Minute // 速率限制器清理间隔

	// 分页相关常量
	DefaultPageSize = 100  // 默认每页大小
	MaxPageSize     = 1000 // 最大每页大小

	// HTTP 服务器相关常量
	MaxHeaderBytes      = 1 << 20           // 1MB，最大请求头大小
	MaxRequestBodySize  = 10 * 1024         // 10KB，最大请求体大小
	ShutdownTimeout     = 5 * time.Second   // 优雅关闭超时时间
	HTTPRetryMaxRetries = 3                 // HTTP 请求最大重试次数
	HTTPRetryDelay      = 1 * time.Second   // HTTP 请求重试延迟
	IdleTimeout         = 120 * time.Second // HTTP 连接空闲超时时间

	// Redis 相关常量
	RedisConnectionTimeout = 5 * time.Second // Redis 连接超时时间

	// 速率限制相关常量
	DefaultRateLimit       = 60              // 默认速率限制：每分钟请求数
	DefaultRateLimitWindow = 1 * time.Minute // 默认速率限制时间窗口
	MaxVisitorsMapSize     = 10000           // 最大访问者 map 大小，防止内存泄漏
	MaxWhitelistSize       = 1000            // 最大白名单大小

	// HTTP 客户端相关常量
	DefaultMaxIdleConns        = 100              // 默认最大空闲连接数
	DefaultMaxIdleConnsPerHost = 10               // 默认每个主机的最大空闲连接数
	DefaultIdleConnTimeout     = 90 * time.Second // 默认空闲连接超时时间
	DefaultLoadDataTimeout     = 30 * time.Second // 默认加载数据超时时间

	// 缓存相关常量
	DefaultSlicePoolCapacity = 100   // 默认切片池容量
	SmallDataThreshold       = 100   // 小数据阈值，小于此值的数据直接处理，不使用缓冲池
	LargeDataThreshold       = 10000 // 大数据阈值，大于此值的数据使用流式 JSON 编码

	// Redis 重试相关常量
	RedisRetryMaxRetries = 3               // Redis 操作最大重试次数
	RedisRetryDelay      = 1 * time.Second // Redis 操作重试延迟
)

const (
	WARN_RULE_NOT_FOUND     = "没有找到规则文件"
	WARN_READ_RULE_ERR      = "读取规则文件遇到错误"
	WARN_PARSE_RULE_ERR     = "解析规则文件遇到错误"
	ERROR_CAN_NOT_OPEN_RULE = "读取规则文件出错"

	ERR_REQ_INIT_FAILED     = "网络请求组件初始化失败"
	ERR_GET_CONFIG_FAILED   = "获取远程配置失败"
	ERR_READ_CONFIG_FAILED  = "读取远程配置失败"
	ERR_PARSE_CONFIG_FAILED = "解析远程配置失败"

	WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL = "获取远程规则出错，仅使用本地规则"

	INFO_REQ_REMOTE_API = "请求数据接口 🎩"
)
