// Package parser 提供了数据解析功能。
// 支持从本地文件和远程 API 解析用户数据，并提供多种数据合并策略。
package parser

import (
	// 标准库
	"context"

	// 项目内部包
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

var log = logger.GetLogger()

// mergeUsers 将 map 转换为切片，按照指定的顺序
// dict: 用户数据 map，key 为 phone
// order: 顺序列表，存储 phone
func mergeUsers(dict map[string]define.AllowListUser, order []string) []define.AllowListUser {
	result := make([]define.AllowListUser, 0, len(order))
	for _, phone := range order {
		if user, exists := dict[phone]; exists {
			result = append(result, user)
		}
	}
	return result
}

// addRulesToDict 将规则添加到字典中，维护顺序列表
// dict: 用户数据 map，key 为 phone
// order: 顺序列表，存储 phone
// rules: 要添加的规则列表
// logMessage: 是否记录日志消息（用于区分远程和本地规则）
// 返回添加的新规则数量
func addRulesToDict(dict map[string]define.AllowListUser, order *[]string, rules []define.AllowListUser, logMessage bool) {
	for _, rule := range rules {
		if _, exists := dict[rule.Phone]; !exists {
			*order = append(*order, rule.Phone)
			if logMessage {
				log.Debug().Msgf("载入远程规则 %s => %s", rule.Mail, rule.Phone)
			}
		}
		dict[rule.Phone] = rule
	}
}

// GetRules 根据模式获取规则（支持 context）
//
// 该函数是规则获取的统一入口，根据不同的模式（appMode）选择不同的数据获取和合并策略。
// 支持的模式包括：
//   - DEFAULT/REMOTE_FIRST: 远程优先，本地补充
//   - ONLY_REMOTE: 仅使用远程规则
//   - ONLY_LOCAL: 仅使用本地规则
//   - LOCAL_FIRST: 本地优先，远程补充
//   - REMOTE_FIRST_ALLOW_REMOTE_FAILED: 远程优先，允许远程失败时继续
//   - LOCAL_FIRST_ALLOW_REMOTE_FAILED: 本地优先，允许远程失败时继续
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - rulesFile: 本地规则文件路径
//   - configUrl: 远程配置 URL
//   - authorizationHeader: 远程请求的 Authorization 头
//   - appMode: 应用模式，决定数据获取策略
//
// 返回:
//   - []define.AllowListUser: 合并后的用户列表，按添加顺序排列
//
// 副作用:
//   - 会记录调试和警告日志
//   - 可能进行网络请求（根据模式）
//   - 可能读取本地文件（根据模式）
func GetRules(ctx context.Context, rulesFile, configURL, authorizationHeader, appMode string) (result []define.AllowListUser) {
	switch appMode {
	case "DEFAULT", "REMOTE_FIRST":
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, false)
	case "ONLY_REMOTE":
		return onlyRemoteRules(ctx, configURL, authorizationHeader)
	case "ONLY_LOCAL":
		return onlyLocalRules(rulesFile)
	case "LOCAL_FIRST":
		return localRulesFirstAppendNotExistsFromRemoteRules(ctx, rulesFile, configURL, authorizationHeader, false)
	case "REMOTE_FIRST_ALLOW_REMOTE_FAILED":
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, true)
	case "LOCAL_FIRST_ALLOW_REMOTE_FAILED":
		return localRulesFirstAppendNotExistsFromRemoteRules(ctx, rulesFile, configURL, authorizationHeader, true)
	default:
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, false)
	}
}

// remoteRulesFirstAppendNotExistsFromLocalRules 远程规则优先，补充本地规则中不存在的项
//
// 该函数实现了远程优先的数据合并策略：
// - 首先尝试从远程 API 获取规则
// - 如果远程获取失败且 allowSkipRemoteFailed 为 false，返回空结果
// - 如果远程获取失败且 allowSkipRemoteFailed 为 true，继续使用本地规则
// - 将本地规则中不存在于远程规则中的项补充到结果中
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - rulesFile: 本地规则文件路径
//   - configUrl: 远程配置 URL
//   - authorizationHeader: 远程请求的 Authorization 头
//   - allowSkipRemoteFailed: 是否允许远程失败时继续处理
//
// 返回:
//   - []define.AllowListUser: 合并后的用户列表，按添加顺序排列
func remoteRulesFirstAppendNotExistsFromLocalRules(ctx context.Context, rulesFile, configURL, authorizationHeader string, allowSkipRemoteFailed bool) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // 维护顺序列表

	// 优先使用远程规则进行初始化
	// 如果 configURL 为空，跳过远程请求，直接使用本地规则
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
			if !allowSkipRemoteFailed {
				return result
			}
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	// 补充远程规则中不存在的本地规则
	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	result = mergeUsers(dict, order)
	log.Debug().Msgf("更新规则完毕 📦")
	return result
}

// localRulesFirstAppendNotExistsFromRemoteRules 本地规则优先，补充远程规则中不存在的项
//
// 该函数实现了本地优先的数据合并策略：
// - 首先从本地文件加载规则
// - 然后尝试从远程 API 获取规则
// - 如果远程获取失败且 allowSkipRemoteFailed 为 false，返回仅包含本地规则的结果
// - 如果远程获取失败且 allowSkipRemoteFailed 为 true，继续使用本地规则
// - 将远程规则中不存在于本地规则中的项补充到结果中
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - rulesFile: 本地规则文件路径
//   - configUrl: 远程配置 URL
//   - authorizationHeader: 远程请求的 Authorization 头
//   - allowSkipRemoteFailed: 是否允许远程失败时继续处理
//
// 返回:
//   - []define.AllowListUser: 合并后的用户列表，按添加顺序排列
func localRulesFirstAppendNotExistsFromRemoteRules(ctx context.Context, rulesFile, configURL, authorizationHeader string, allowSkipRemoteFailed bool) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // 维护顺序列表

	// 优先加载本地数据
	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	// 补充本地规则中不存在的远程规则
	// 如果 configURL 为空，跳过远程请求
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
			if !allowSkipRemoteFailed {
				return result
			}
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	result = mergeUsers(dict, order)
	log.Debug().Msgf("更新规则完毕 📦")
	return result
}

// onlyRemoteRules 仅使用远程规则
//
// 该函数仅从远程 API 获取规则，不使用本地文件。
// 如果远程获取失败，返回空结果。
//
// 参数:
//   - ctx: 上下文，用于取消请求和超时控制
//   - configUrl: 远程配置 URL
//   - authorizationHeader: 远程请求的 Authorization 头
//
// 返回:
//   - []define.AllowListUser: 远程获取的用户列表，如果获取失败则返回空列表
func onlyRemoteRules(ctx context.Context, configURL, authorizationHeader string) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // 维护顺序列表

	// 使用远程规则进行初始化
	// 如果 configURL 为空，直接返回空结果
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	result = mergeUsers(dict, order)
	log.Debug().Msgf("更新规则完毕 📦")
	return result
}

// onlyLocalRules 仅使用本地规则
//
// 该函数仅从本地文件加载规则，不访问远程 API。
//
// 参数:
//   - rulesFile: 本地规则文件路径
//
// 返回:
//   - []define.AllowListUser: 从本地文件加载的用户列表
func onlyLocalRules(rulesFile string) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // 维护顺序列表

	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	result = mergeUsers(dict, order)
	log.Debug().Msgf("更新规则完毕 📦")
	return result
}
