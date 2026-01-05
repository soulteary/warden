// Package parser 提供了数据解析功能。
// 支持从本地文件和远程 API 解析用户数据，并提供多种数据合并策略。
package parser

import (
	// 标准库
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
)

// httpClient 全局 HTTP 客户端，使用连接池复用连接
var httpClient = &http.Client{
	Timeout: define.DEFAULT_TIMEOUT * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        define.DEFAULT_MAX_IDLE_CONNS,
		MaxIdleConnsPerHost: define.DEFAULT_MAX_IDLE_CONNS_PER_HOST,
		IdleConnTimeout:     define.DEFAULT_IDLE_CONN_TIMEOUT,
		DisableKeepAlives:   false, // 明确设置，启用连接复用
	},
}

// InitHTTPClient 初始化 HTTP 客户端（使用配置）
func InitHTTPClient(timeout int, maxIdleConns int, insecureTLS bool) {
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: define.DEFAULT_MAX_IDLE_CONNS_PER_HOST,
		IdleConnTimeout:     define.DEFAULT_IDLE_CONN_TIMEOUT,
		DisableKeepAlives:   false,
	}

	// 配置 TLS
	if insecureTLS {
		// #nosec G402 -- 仅用于开发环境，允许跳过 TLS 验证
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // 仅用于开发环境
		}
	}

	httpClient = &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}
}

// doRequestWithRetry 执行 HTTP 请求，带重试机制
//
// 该函数实现了指数退避的重试策略，支持以下特性：
// - 上下文取消：在每次重试前检查上下文是否已取消
// - 自动重试：网络错误和 5xx 服务器错误会自动重试
// - 递增延迟：每次重试的延迟时间会递增（retryDelay * attempt）
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - req: HTTP 请求对象
//   - maxRetries: 最大重试次数（不包括首次请求）
//   - retryDelay: 基础重试延迟时间，实际延迟会按重试次数递增
//
// 返回:
//   - *http.Response: 成功时返回响应对象，调用者需要负责关闭响应体
//   - error: 失败时返回错误，包含重试次数和最后一次错误信息
//
// 副作用:
//   - 会记录调试和警告日志
//   - 对于 5xx 错误，会关闭响应体后重试
func doRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int, retryDelay time.Duration) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("请求被取消: %w", ctx.Err())
		default:
		}

		if attempt > 0 {
			// 等待后重试，但检查上下文
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("请求被取消: %w", ctx.Err())
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
			log.Debug().
				Int("attempt", attempt).
				Str("url", req.URL.String()).
				Msg("重试 HTTP 请求")
		}

		// 将上下文添加到请求中
		reqWithCtx := req.WithContext(ctx)
		res, err := httpClient.Do(reqWithCtx)
		if err == nil {
			// 检查状态码，5xx 错误也重试
			if res.StatusCode >= 500 && res.StatusCode < 600 && attempt < maxRetries {
				_ = res.Body.Close()
				lastErr = fmt.Errorf("服务器错误: HTTP %d", res.StatusCode)
				continue
			}
			return res, nil
		}

		lastErr = err
		// 网络错误才重试，其他错误（如超时）也重试
		if attempt < maxRetries {
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Str("url", req.URL.String()).
				Msg("HTTP 请求失败，将重试")
		}
	}

	return nil, fmt.Errorf("请求失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// buildRemoteRequest 构建远程请求
func buildRemoteRequest(ctx context.Context, url string, authorizationHeader string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_REQ_INIT_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_REQ_INIT_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_REQ_INIT_FAILED, err)
	}

	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Cache-Control": {"max-age=0"},
	}
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	return req, nil
}

// parseRemoteResponse 解析远程响应
func parseRemoteResponse(res *http.Response, url string) ([]define.AllowListUser, error) {
	defer func() {
		_ = res.Body.Close()
	}()

	// 检查 HTTP 状态码
	if res.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", res.StatusCode).
			Str("url", url).
			Msgf("%s: HTTP status %d", define.ERR_GET_CONFIG_FAILED, res.StatusCode)
		return nil, fmt.Errorf("%s: HTTP status %d", define.ERR_GET_CONFIG_FAILED, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_READ_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_READ_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_READ_CONFIG_FAILED, err)
	}

	var data []define.AllowListUser
	if err := json.Unmarshal(body, &data); err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_PARSE_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_PARSE_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_PARSE_CONFIG_FAILED, err)
	}

	return data, nil
}

// FromRemoteConfig 从远程配置获取用户列表（支持 context）
//
// 该函数从远程 URL 获取 JSON 格式的用户配置数据，支持以下特性：
// - 上下文控制：支持超时和取消操作
// - 自动重试：使用 doRequestWithRetry 实现自动重试机制
// - 认证支持：可选的 Authorization 请求头
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - url: 远程配置的 URL 地址
//   - authorizationHeader: 可选的 Authorization 请求头值，为空时不添加
//
// 返回:
//   - []define.AllowListUser: 成功时返回解析后的用户列表
//   - error: 失败时返回错误，可能的原因包括：请求初始化失败、网络错误、HTTP 状态码错误、JSON 解析失败
//
// 副作用:
//   - 会记录错误和警告日志
//   - 会设置请求头（Content-Type、Cache-Control、Authorization）
func FromRemoteConfig(ctx context.Context, url string, authorizationHeader string) ([]define.AllowListUser, error) {
	req, err := buildRemoteRequest(ctx, url, authorizationHeader)
	if err != nil {
		return nil, err
	}

	res, err := doRequestWithRetry(ctx, req, define.HTTP_RETRY_MAX_RETRIES, define.HTTP_RETRY_DELAY)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_GET_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_GET_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_GET_CONFIG_FAILED, err)
	}

	return parseRemoteResponse(res, url)
}
