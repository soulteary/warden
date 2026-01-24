# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

一个高性能的允许列表（AllowList）用户数据服务，支持本地和远程配置源的数据同步与合并。

![Warden](.github/assets/banner.jpg)

> **Warden**（看守者）—— 守护星门的看守者，决定谁可以通过，谁将被拒绝。正如 Stargate 的看守者守护着星际之门，Warden 守护着你的允许列表，确保只有授权用户能够通过。

## 📋 概述

Warden 是一个基于 Go 语言开发的轻量级 HTTP API 服务，主要用于提供和管理允许列表用户数据（手机号和邮箱）。该服务支持从本地配置文件和远程 API 获取数据，并提供了多种数据合并策略，确保数据的实时性和可靠性。

Warden 可以**独立使用**，也可以选择性地与其他服务（如 Stargate 和 Herald）集成，作为更大认证架构的一部分。详细架构信息请参考 [架构文档](docs/zhCN/ARCHITECTURE.md)。

## ✨ 核心特性

- 🚀 **高性能**: 每秒 5000+ 请求，平均延迟 21ms
- 🔄 **多数据源**: 支持本地配置文件和远程 API
- 🎯 **灵活策略**: 6 种数据合并模式（远程优先、本地优先、仅远程、仅本地等）
- ⏰ **定时更新**: 基于 Redis 分布式锁的自动数据同步
- 📦 **容器化部署**: 完整的 Docker 支持，开箱即用
- 🌐 **多语言支持**: 支持 7 种语言，自动检测用户语言偏好

## 🚀 快速开始

### 方式一：Docker（推荐）

最快的方式是使用预构建的 Docker 镜像：

```bash
# 拉取最新镜像
docker pull ghcr.io/soulteary/warden:latest

# 创建数据文件
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# 运行容器
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **提示**: 查看 [示例目录](example/README.zhCN.md) 获取完整的 Docker Compose 配置示例。

### 方式二：从源码运行

1. **克隆并构建**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **创建数据文件**
创建 `data.json` 文件（可参考 `data.example.json`）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **运行服务**
```bash
go run main.go --api-key your-api-key-here
```

## ⚙️ 核心配置

Warden 支持通过命令行参数、环境变量和配置文件进行配置。以下是最核心的设置：

| 配置项 | 环境变量 | 说明 | 必需 |
|--------|---------|------|------|
| 端口 | `PORT` | HTTP 服务器端口（默认：8081） | 否 |
| API 密钥 | `API_KEY` | API 认证密钥（生产环境推荐） | 推荐 |
| Redis | `REDIS` | Redis 地址，用于缓存和分布式锁（如：`localhost:6379`） | 可选 |
| 数据文件 | - | 本地数据文件路径（默认：`data.json`） | 是* |
| 远程配置 | `CONFIG` | 用于获取数据的远程 API URL | 可选 |

\* 如果不使用远程 API，则必需

完整配置选项请参考 [配置文档](docs/zhCN/CONFIGURATION.md)。

## 📡 API 使用

Warden 提供了 RESTful API，支持查询用户列表、分页和健康检查。服务支持通过查询参数 `?lang=xx` 或 `Accept-Language` 头返回多语言响应。

**示例**：
```bash
# 查询用户
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# 健康检查
curl "http://localhost:8081/health"
```

完整 API 文档请参考 [API 文档](docs/zhCN/API.md) 或 [OpenAPI 规范](openapi.yaml)。

## 📊 性能指标

基于 wrk 压力测试（30秒，16线程，100连接）：
- **每秒请求数**: 5038.81
- **平均延迟**: 21.30ms
- **最大延迟**: 226.09ms

## 📚 文档

### 核心文档

- **[架构设计](docs/zhCN/ARCHITECTURE.md)** - 技术架构和设计决策
- **[API 参考](docs/zhCN/API.md)** - 完整的 API 端点文档
- **[配置说明](docs/zhCN/CONFIGURATION.md)** - 配置参考和示例
- **[部署指南](docs/zhCN/DEPLOYMENT.md)** - 部署指南（Docker、Kubernetes 等）

### 附加资源

- **[开发指南](docs/zhCN/DEVELOPMENT.md)** - 开发环境设置和贡献指南
- **[安全文档](docs/zhCN/SECURITY.md)** - 安全特性和最佳实践
- **[SDK 文档](docs/zhCN/SDK.md)** - Go SDK 使用文档
- **[示例](example/README.zhCN.md)** - 快速开始示例（基础和高级）

## 📄 许可证

查看 [LICENSE](LICENSE) 文件了解详情。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！请参考 [CONTRIBUTING.md](docs/zhCN/CONTRIBUTING.md) 了解贡献指南。
