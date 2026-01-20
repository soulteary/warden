# 文档索引

欢迎查阅 Warden AllowList 用户数据服务的文档。

## 🌐 多语言文档 / Multi-language Documentation

- [English](../enUS/README.md) | [中文](README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 文档列表

### 核心文档

- **[README.md](../../README.md)** - 项目概述和快速开始指南
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 技术架构和设计决策

### 详细文档

- **[API.md](API.md)** - 完整的 API 端点文档
  - 用户列表查询端点
  - 分页功能
  - 健康检查端点
  - 错误响应格式

- **[CONFIGURATION.md](CONFIGURATION.md)** - 配置参考文档
  - 配置方式
  - 必需配置项
  - 可选配置项
  - 数据合并策略
  - 配置示例
  - 配置最佳实践

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - 部署指南
  - Docker 部署（包括 GHCR 镜像）
  - Docker Compose 部署
  - 本地部署
  - 生产环境部署
  - Kubernetes 部署
  - 性能优化

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - 开发指南
  - 开发环境设置
  - 代码结构说明
  - 测试指南
  - 贡献指南

- **[SDK.md](SDK.md)** - SDK 使用文档
  - Go SDK 安装和使用
  - API 接口说明
  - 示例代码

- **[SECURITY.md](SECURITY.md)** - 安全文档
  - 安全特性说明
  - 安全配置
  - 最佳实践

- **[CODE_STYLE.md](CODE_STYLE.md)** - 代码风格指南
  - 代码规范
  - 命名约定
  - 最佳实践

## 🌐 多语言支持

Warden 支持完整的多语言（i18N）功能，所有 API 响应、错误消息和日志都支持国际化。

### 支持的语言

- 🇺🇸 英语 (en) - 默认语言
- 🇨🇳 中文 (zh)
- 🇫🇷 法语 (fr)
- 🇮🇹 意大利语 (it)
- 🇯🇵 日语 (ja)
- 🇩🇪 德语 (de)
- 🇰🇷 韩语 (ko)

### 语言检测方式

Warden 支持两种语言检测方式，优先级如下：

1. **查询参数**: 通过 URL 查询参数 `?lang=zh` 指定语言
2. **Accept-Language 头**: 自动检测浏览器或客户端的语言偏好
3. **默认语言**: 如果未指定，使用英语

### 使用示例

#### 通过查询参数指定语言

```bash
# 使用中文
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=zh"

# 使用日语
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# 使用法语
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

#### 通过 Accept-Language 头自动检测

```bash
# 浏览器会自动发送 Accept-Language 头
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: zh-CN,zh;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### 国际化范围

以下内容支持多语言：

- ✅ API 错误响应消息
- ✅ HTTP 状态码对应的错误消息
- ✅ 日志消息（基于请求上下文）
- ✅ 配置和警告消息

### 技术实现

- 使用请求上下文（context）存储语言信息，避免全局状态
- 支持线程安全的语言切换
- 自动回退到英语（如果翻译不存在）
- 所有翻译内容内置在代码中，无需外部文件

### 开发说明

如需添加新的翻译或修改现有翻译，请编辑 `internal/i18n/i18n.go` 文件中的 `translations` map。

## 🚀 快速导航

### 新手入门

1. 阅读 [README.md](../../README.md) 了解项目
2. 查看 [快速开始](../../README.md#快速开始) 部分
3. 参考 [配置说明](../../README.md#配置说明) 配置服务

### 开发人员

1. 阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构
2. 查看 [API.md](API.md) 了解 API 接口
3. 参考 [开发指南](../../README.md#开发指南) 进行开发

### 运维人员

1. 阅读 [DEPLOYMENT.md](DEPLOYMENT.md) 了解部署方式
2. 查看 [CONFIGURATION.md](CONFIGURATION.md) 了解配置选项
3. 参考 [性能优化](DEPLOYMENT.md#性能优化) 优化服务

## 📖 文档结构

```
warden/
├── README.md              # 项目主文档（中文）
├── README.en.md           # 项目主文档（英文）
├── docs/
│   ├── enUS/
│   │   ├── README.md       # 文档索引（英文）
│   │   ├── ARCHITECTURE.md # 架构文档（英文）
│   │   ├── API.md          # API 文档（英文）
│   │   ├── CONFIGURATION.md # 配置参考（英文）
│   │   ├── DEPLOYMENT.md   # 部署指南（英文）
│   │   ├── DEVELOPMENT.md   # 开发指南（英文）
│   │   ├── SDK.md          # SDK 文档（英文）
│   │   ├── SECURITY.md     # 安全文档（英文）
│   │   └── CODE_STYLE.md   # 代码风格（英文）
│   └── zhCN/
│       ├── README.md       # 文档索引（中文，本文件）
│       ├── ARCHITECTURE.md # 架构文档（中文）
│       ├── API.md          # API 文档（中文）
│       ├── CONFIGURATION.md # 配置参考（中文）
│       ├── DEPLOYMENT.md   # 部署指南（中文）
│       ├── DEVELOPMENT.md  # 开发指南（中文）
│       ├── SDK.md          # SDK 文档（中文）
│       ├── SECURITY.md     # 安全文档（中文）
│       ├── CODE_STYLE.md   # 代码风格（中文）
│       └── CONFIG_PARSING.md # 配置解析（中文）
└── ...
```

## 🔍 按主题查找

### 配置相关

- 环境变量配置：[CONFIGURATION.md](CONFIGURATION.md)
- 数据合并策略：[CONFIGURATION.md](CONFIGURATION.md)
- 配置示例：[CONFIGURATION.md](CONFIGURATION.md)

### API 相关

- API 端点列表：[API.md](API.md)
- 错误处理：[API.md](API.md)
- 分页功能：[API.md](API.md)

### 部署相关

- Docker 部署：[DEPLOYMENT.md#docker-部署](DEPLOYMENT.md#docker-部署)
- GHCR 镜像：[DEPLOYMENT.md#使用预构建镜像推荐](DEPLOYMENT.md#使用预构建镜像推荐)
- 生产环境：[DEPLOYMENT.md#生产环境部署建议](DEPLOYMENT.md#生产环境部署建议)
- Kubernetes：[DEPLOYMENT.md#kubernetes-部署](DEPLOYMENT.md#kubernetes-部署)

### 架构相关

- 技术栈：[ARCHITECTURE.md](ARCHITECTURE.md)
- 项目结构：[ARCHITECTURE.md](ARCHITECTURE.md)
- 核心组件：[ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 使用建议

1. **首次使用**：从 [README.md](../../README.md) 开始，按照快速开始指南操作
2. **配置服务**：参考 [CONFIGURATION.md](CONFIGURATION.md) 了解所有配置选项
3. **部署服务**：查看 [DEPLOYMENT.md](DEPLOYMENT.md) 了解部署方式
4. **开发扩展**：阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构设计
5. **集成 SDK**：参考 [SDK.md](SDK.md) 了解如何使用 SDK

## 📝 文档更新

文档会随着项目的发展持续更新。如果发现文档有误或需要补充，欢迎提交 Issue 或 Pull Request。

## 🤝 贡献

欢迎贡献文档改进：

1. 发现错误或需要改进的地方
2. 提交 Issue 描述问题
3. 或直接提交 Pull Request
