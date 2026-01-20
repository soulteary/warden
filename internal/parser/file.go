// Package parser 提供了数据解析功能。
// 支持从本地文件和远程 API 解析用户数据，并提供多种数据合并策略。
package parser

import (
	// 标准库
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	// 项目内部包
	"github.com/soulteary/warden/internal/define"
)

// FromFile 从本地文件读取用户规则列表。
//
// 该函数从指定的 JSON 文件中读取用户数据，支持以下特性：
// - 自动处理文件不存在的情况（返回空列表）
// - 自动处理文件读取错误（记录警告日志）
// - 自动处理 JSON 解析错误（记录警告日志）
//
// 参数:
//   - rulesFile: 规则文件的路径，应为有效的 JSON 文件路径
//
// 返回:
//   - []define.AllowListUser: 解析后的用户列表，如果文件不存在或解析失败则返回空列表
//
// 副作用:
//   - 会记录警告和错误日志
//   - 如果文件存在但读取失败，会记录错误日志
//   - 如果 JSON 解析失败，会记录警告日志
func FromFile(rulesFile string) (rules []define.AllowListUser) {
	if _, err := os.Stat(rulesFile); errors.Is(err, os.ErrNotExist) {
		log.Warn().
			Str("err", define.WARN_RULE_NOT_FOUND).
			Msgf(define.WARN_RULE_NOT_FOUND)
		return rules
	}

	// #nosec G304 -- rulesFile 来自配置文件，已通过验证
	file, err := os.Open(rulesFile)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERROR_CAN_NOT_OPEN_RULE, err)).
			Str("file", rulesFile).
			Msg(define.ERROR_CAN_NOT_OPEN_RULE)
		return rules
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Error().
				Err(closeErr).
				Str("file", rulesFile).
				Msg("关闭文件失败")
		}
	}()

	// 限制文件读取大小，防止内存耗尽攻击
	raw, err := io.ReadAll(io.LimitReader(file, define.MAX_JSON_SIZE))
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WARN_READ_RULE_ERR, err)).
			Str("file", rulesFile).
			Msg(define.WARN_READ_RULE_ERR)
		return rules
	}

	err = json.Unmarshal(raw, &rules)
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WARN_PARSE_RULE_ERR, err)).
			Str("file", rulesFile).
			Msg(define.WARN_PARSE_RULE_ERR)
		return rules
	}

	// 规范化所有用户数据（设置默认值，生成 user_id）
	for i := range rules {
		rules[i].Normalize()
	}

	return rules
}
