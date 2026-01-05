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
	"soulteary.com/soulteary/warden/internal/define"
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
			Str("err", define.WarnRuleNotFound).
			Msgf(define.WarnRuleNotFound)
		return rules
	}

	// #nosec G304 -- rulesFile 来自配置文件，已通过验证
	file, err := os.Open(rulesFile)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ErrorCanNotOpenRule, err)).
			Str("file", rulesFile).
			Msg(define.ErrorCanNotOpenRule)
		return rules
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error().
				Err(err).
				Str("file", rulesFile).
				Msg("关闭文件失败")
		}
	}()

	raw, err := io.ReadAll(file)
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WarnReadRuleErr, err)).
			Str("file", rulesFile).
			Msg(define.WarnReadRuleErr)
		return rules
	}

	err = json.Unmarshal(raw, &rules)
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WarnParseRuleErr, err)).
			Str("file", rulesFile).
			Msg(define.WarnParseRuleErr)
		return rules
	}
	return rules
}
