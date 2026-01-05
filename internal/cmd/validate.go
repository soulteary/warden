package cmd

import (
	// 标准库
	"fmt"
	"net/url"
	"strconv"
	"strings"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
)

// ValidateConfig 验证配置的有效性
func ValidateConfig(cfg *Config) error {
	var errors []string

	// 验证端口
	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		errors = append(errors, fmt.Sprintf("无效的端口号: %s (必须是 1-65535 之间的整数)", cfg.Port))
	}

	// 验证 Redis 地址格式
	if cfg.Redis != "" {
		parts := strings.Split(cfg.Redis, ":")
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("无效的 Redis 地址格式: %s (应为 host:port)", cfg.Redis))
		} else {
			if port, err := strconv.Atoi(parts[1]); err != nil || port < 1 || port > 65535 {
				errors = append(errors, fmt.Sprintf("无效的 Redis 端口: %s", parts[1]))
			}
		}
	}

	// 验证远程配置 URL
	if cfg.RemoteConfig != "" && cfg.RemoteConfig != define.DefaultRemoteConfig {
		if _, err := url.ParseRequestURI(cfg.RemoteConfig); err != nil {
			errors = append(errors, fmt.Sprintf("无效的远程配置 URL: %s (%v)", cfg.RemoteConfig, err))
		}
	}

	// 验证任务间隔
	if cfg.TaskInterval < 1 {
		errors = append(errors, fmt.Sprintf("任务间隔必须大于 0，当前值: %d", cfg.TaskInterval))
	}

	// 验证模式
	validModes := map[string]bool{
		"DEFAULT":                          true,
		"REMOTE_FIRST":                     true,
		"ONLY_REMOTE":                      true,
		"ONLY_LOCAL":                       true,
		"LOCAL_FIRST":                      true,
		"REMOTE_FIRST_ALLOW_REMOTE_FAILED": true,
		"LOCAL_FIRST_ALLOW_REMOTE_FAILED":  true,
	}
	if !validModes[cfg.Mode] {
		errors = append(errors, fmt.Sprintf("无效的模式: %s (有效值: DEFAULT, REMOTE_FIRST, ONLY_REMOTE, ONLY_LOCAL, LOCAL_FIRST, REMOTE_FIRST_ALLOW_REMOTE_FAILED, LOCAL_FIRST_ALLOW_REMOTE_FAILED)", cfg.Mode))
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证失败:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
