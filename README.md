# Warden

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

一个高性能的允许列表（AllowList）用户数据服务，支持本地和远程配置源的数据同步与合并。

![Warden](.github/assets/banner.jpg)

> **Warden**（看守者）—— 守护星门的看守者，决定谁可以通过，谁将被拒绝。正如 Stargate 的看守者守护着星际之门，Warden 守护着你的允许列表，确保只有授权用户能够通过。

## 📋 项目简介

Warden 是一个基于 Go 语言开发的轻量级 HTTP API 服务，主要用于提供和管理允许列表用户数据（手机号和邮箱）。该服务支持从本地配置文件和远程 API 获取数据，并提供了多种数据合并策略，确保数据的实时性和可靠性。

## ✨ 核心特性

- 🚀 **高性能**: 支持每秒 5000+ 请求，平均延迟 21ms
- 🔄 **多数据源**: 支持本地配置文件和远程 API 两种数据源
- 🎯 **灵活策略**: 提供 6 种数据合并模式（远程优先、本地优先、仅远程、仅本地等）
- ⏰ **定时更新**: 基于 Redis 分布式锁的定时任务，自动同步数据
- 📦 **容器化部署**: 完整的 Docker 支持，开箱即用
- 📊 **结构化日志**: 使用 zerolog 提供详细的访问日志和错误日志
- 🔒 **分布式锁**: 使用 Redis 确保定时任务在分布式环境下不会重复执行
- 🌐 **多语言支持**: 支持 7 种语言（英语、中文、法语、意大利语、日语、德语、韩语），自动检测用户语言偏好

## 🏗️ 架构设计

Warden 采用分层架构设计，包含 HTTP 层、业务层和基础设施层。系统支持多数据源、多级缓存和分布式锁机制。

详细架构说明请参考：[架构设计文档](docs/zhCN/ARCHITECTURE.md)

## 📦 安装与运行

> 💡 **快速开始**: 想要快速体验 Warden？查看我们的 [快速开始示例](example/README.md)：
> - [简单示例](example/basic/README.md) - 基础使用，仅本地数据文件
> - [复杂示例](example/advanced/README.md) - 完整功能，包含远程 API 和 Mock 服务

### 前置要求

- Go 1.25+ (参考 [go.mod](go.mod))
- Redis (用于分布式锁和缓存)
- Docker (可选，用于容器化部署)

### 快速开始

1. **克隆项目**
```bash
git clone <repository-url>
cd warden
```

2. **安装依赖**
```bash
go mod download
```

3. **配置本地数据文件**
创建 `data.json` 文件（可参考 `data.example.json`）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **运行服务**
```bash
go run main.go
```

详细配置和部署说明请参考：
- [配置文档](docs/zhCN/CONFIGURATION.md) - 了解所有配置选项
- [部署文档](docs/zhCN/DEPLOYMENT.md) - 了解部署方法

## ⚙️ 配置说明

Warden 支持多种配置方式：命令行参数、环境变量和配置文件。系统提供 6 种数据合并模式，支持灵活的配置策略。

详细配置说明请参考：[配置文档](docs/zhCN/CONFIGURATION.md)

## 📡 API 文档

Warden 提供了完整的 RESTful API，支持用户列表查询、分页、健康检查等功能。项目还提供了 OpenAPI 3.0 规范文档。

详细 API 文档请参考：[API 文档](docs/zhCN/API.md)

OpenAPI 规范文件：[openapi.yaml](openapi.yaml)

## 🌐 多语言支持

Warden 支持完整的多语言（i18N）功能，所有 API 响应、错误消息和日志都支持国际化。

### 支持的语言

- 🇺🇸 英语 (en) - 默认
- 🇨🇳 中文 (zh)
- 🇫🇷 法语 (fr)
- 🇮🇹 意大利语 (it)
- 🇯🇵 日语 (ja)
- 🇩🇪 德语 (de)
- 🇰🇷 韩语 (ko)

### 语言检测方式

Warden 支持两种语言检测方式，优先级如下：

1. **查询参数**: 通过 `?lang=zh` 指定语言
2. **Accept-Language 头**: 自动检测浏览器语言偏好
3. **默认语言**: 如果未指定，使用英语

### 使用示例

```bash
# 通过查询参数指定中文
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=zh"

# 通过 Accept-Language 头自动检测
curl -H "X-API-Key: your-key" -H "Accept-Language: zh-CN,zh;q=0.9" "http://localhost:8081/"

# 使用日语
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"
```

详细多语言文档请参考：[多语言文档](docs/zhCN/README.md#多语言支持)

## 🔌 SDK 使用

Warden 提供了 Go SDK，方便其他项目集成使用。SDK 提供了简洁的 API 接口，支持缓存、认证等功能。

详细 SDK 文档请参考：[SDK 文档](docs/zhCN/SDK.md)

## 🐳 Docker 部署

Warden 支持完整的 Docker 和 Docker Compose 部署，开箱即用。

### 使用预构建镜像快速开始（推荐）

使用 GitHub Container Registry (GHCR) 提供的预构建镜像，无需本地构建即可快速启动：

```bash
# 拉取最新版本的镜像
docker pull ghcr.io/soulteary/warden:latest

# 运行容器（基础示例）
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **提示**: 使用预构建镜像可以快速开始，无需本地构建环境。镜像会自动更新，确保使用最新版本。

### 使用 Docker Compose

> 🚀 **快速部署**: 查看 [示例目录](example/README.md) 获取完整的 Docker Compose 配置示例

详细部署文档请参考：[部署文档](docs/zhCN/DEPLOYMENT.md)

## 📊 性能指标

基于 wrk 压力测试结果（30秒测试，16线程，100连接）：

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
平均延迟:       21.30ms
最大延迟:       226.09ms
```

## 📁 项目结构

```
warden/
├── main.go                 # 程序入口
├── data.example.json      # 本地数据文件示例
├── config.example.yaml    # 配置文件示例
├── openapi.yaml           # OpenAPI 规范文件
├── go.mod                 # Go 模块定义
├── docker-compose.yml     # Docker Compose 配置
├── LICENSE                # 许可证文件
├── README.*.md            # 多语言项目文档（中文/英文/法语/意大利语/日语/德语/韩语）
├── CONTRIBUTING.*.md      # 多语言贡献指南
├── docker/
│   └── Dockerfile         # Docker 镜像构建文件
├── docs/                  # 文档目录（多语言）
│   ├── enUS/              # 英文文档
│   └── zhCN/              # 中文文档
├── example/               # 快速开始示例
│   ├── basic/             # 简单示例（仅本地文件）
│   └── advanced/          # 复杂示例（完整功能，包含 Mock API）
├── internal/
│   ├── cache/             # Redis 缓存和锁实现
│   ├── cmd/               # 命令行参数解析
│   ├── config/            # 配置管理
│   ├── define/            # 常量定义和数据结构
│   ├── di/                # 依赖注入
│   ├── errors/            # 错误处理
│   ├── i18n/              # 国际化支持
│   ├── logger/            # 日志初始化
│   ├── metrics/           # 指标收集
│   ├── middleware/        # HTTP 中间件
│   ├── parser/            # 数据解析器（本地/远程）
│   ├── router/            # HTTP 路由处理
│   ├── validator/         # 验证器
│   └── version/           # 版本信息
├── pkg/
│   ├── gocron/            # 定时任务调度器
│   └── warden/            # Warden SDK
├── scripts/               # 脚本目录
└── .github/               # GitHub 配置（CI/CD、Issue/PR 模板等）
```

## 🔒 安全特性

Warden 实现了多项安全功能，包括 API 认证、SSRF 防护、速率限制、TLS 验证等。

详细安全文档请参考：[安全文档](docs/zhCN/SECURITY.md)

## 🔧 开发指南

> 📚 **参考示例**: 查看 [示例目录](example/README.md) 了解不同使用场景的完整示例代码和配置。

详细开发文档请参考：[开发文档](docs/zhCN/DEVELOPMENT.md)

### 代码规范

项目遵循 Go 官方代码规范和最佳实践。详细规范请参考：

- [CODE_STYLE.md](docs/zhCN/CODE_STYLE.md) - 代码风格指南
- [CONTRIBUTING.md](CONTRIBUTING.md) - 贡献指南

## 📄 许可证

查看 [LICENSE](LICENSE) 文件了解详情。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 联系方式

如有问题或建议，请通过 Issue 联系。

---

**版本**: 程序启动时会显示版本、构建时间和代码版本（通过 `warden --version` 或查看启动日志）
