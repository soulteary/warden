# 开发指南

> 🌐 **Language / 语言**: [English](../enUS/DEVELOPMENT.md) | [中文](DEVELOPMENT.md) | [Français](../frFR/DEVELOPMENT.md) | [Italiano](../itIT/DEVELOPMENT.md) | [日本語](../jaJP/DEVELOPMENT.md) | [Deutsch](../deDE/DEVELOPMENT.md) | [한국어](../koKR/DEVELOPMENT.md)

本文档为 Warden 项目的开发者提供开发指南，包括项目结构、开发流程、测试方法等。

## 项目结构

```
warden/
├── main.go                 # 程序入口
├── main_routes.go         # 路由注册与健康检查配置
├── main_test.go           # 包测试
├── data.example.json      # 本地数据文件示例
├── config.example.yaml     # 应用配置文件示例
├── go.mod                 # Go 模块定义
├── docker-compose.yml     # Docker Compose 配置
├── docker/
│   └── Dockerfile         # Docker 镜像构建文件
├── example/                # 快速开始示例
│   ├── README.md          # 示例说明文档
│   ├── basic/             # 简单示例（仅本地文件）
│   └── advanced/          # 高级示例（完整功能）
├── internal/
│   ├── auditlog/          # 审计日志
│   ├── cache/             # 用户缓存（内存 + Redis）与分布式锁
│   ├── cmd/               # 命令行与配置解析
│   ├── config/            # YAML 配置文件加载
│   ├── define/            # 常量与数据结构
│   ├── di/                # 依赖注入（服务器、处理器、健康检查）
│   ├── errors/            # 错误类型与 i18n
│   ├── i18n/              # 国际化
│   ├── loader/            # 数据加载器（parser-kit，多数据源）
│   ├── logger/            # 日志初始化
│   ├── metrics/           # Prometheus 指标
│   ├── middleware/        # HTTP 中间件（i18n、错误处理、指标、IP 白名单）
│   ├── router/            # HTTP 路由（用户列表、查询、健康、指标、日志级别）
│   ├── tracing/           # OpenTelemetry 追踪
│   └── validator/         # URL 与输入校验
└── pkg/
    ├── gocron/            # 定时任务调度器
    └── warden/            # Go SDK（客户端、缓存、类型）
```

## 开发环境设置

### 1. 克隆项目

```bash
git clone <repository-url>
cd warden
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行开发服务器

```bash
go run .
```

## 添加新功能

### 代码组织

1. **核心业务逻辑**: 在 `internal/` 目录下
2. **路由处理**: 在 `internal/router/` 目录
3. **数据加载逻辑**: 在 `internal/loader/` 目录（基于 parser-kit）
4. **公共包**: 在 `pkg/` 目录

### 开发流程

1. 创建功能分支
2. 实现功能并编写测试
3. 运行测试确保通过
4. 提交代码并创建 Pull Request

## 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试最佳实践

- 为每个新功能编写单元测试
- 保持测试覆盖率在合理水平
- 使用表驱动测试（table-driven tests）
- 测试边界条件和错误情况

## 代码规范

项目遵循 Go 官方代码规范和最佳实践。详细规范请参考：

- [CODE_STYLE.md](CODE_STYLE.md) / [CODE_STYLE.md](../enUS/CODE_STYLE.md) - 代码风格指南
- [CONTRIBUTING.md](../zhCN/CONTRIBUTING.md) / [CONTRIBUTING.md](../enUS/CONTRIBUTING.md) - 贡献指南

### 代码格式化

```bash
# 格式化代码
go fmt ./...

# 运行静态分析工具
go vet ./...

# 使用 golangci-lint（如果已安装）
golangci-lint run
```

## API 文档

项目提供了完整的 OpenAPI 3.0 规范文档：

- [openapi.yaml](../openapi.yaml) - OpenAPI 规范文件

可以使用以下工具查看：

- [Swagger Editor](https://editor.swagger.io/) - 在线查看和编辑
- [Redoc](https://github.com/Redocly/redoc) - 生成美观的文档页面
- Postman - 导入并测试 API

### 更新 API 文档

当添加或修改 API 端点时，需要同步更新 `openapi.yaml` 文件。

## 日志

服务使用结构化日志记录以下信息：

- **访问日志**: HTTP 请求方法、URL、状态码、响应大小、耗时
- **业务日志**: 数据更新、规则加载、错误信息
- **系统日志**: 服务启动、关闭、版本信息

### 日志级别

支持的日志级别：`trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

可以通过环境变量 `LOG_LEVEL` 或 API 端点 `/log/level` 设置。

## 参考示例

查看 [示例目录](../example/README.md) / [Examples Directory](../example/README.md) 了解不同使用场景的完整示例代码和配置。

## 性能测试

### 使用 wrk 进行压力测试

```bash
# 安装 wrk
# macOS: brew install wrk
# Linux: apt-get install wrk

# 运行压力测试
wrk -t16 -c100 -d30s --latency http://localhost:8081/health
```

### 性能基准

基于 wrk 压力测试结果（30秒测试，16线程，100连接）：

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
平均延迟:       21.30ms
最大延迟:       226.09ms
```

## 调试

### 启用调试日志

```bash
export LOG_LEVEL=debug
go run .
```

或通过 API 动态设置：

```bash
curl -X POST http://localhost:8081/log/level \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"level":"debug"}'
```

### 使用调试器

```bash
# 使用 Delve 调试器（需编译整个包）
dlv debug .
```

## 构建

请编译整个 `main` 包，使 `main.go` 与 `main_routes.go` 等一起参与编译。

### 本地构建

```bash
go build -o warden .
```

### 交叉编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o warden-linux-amd64 .

# macOS
GOOS=darwin GOARCH=amd64 go build -o warden-darwin-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o warden-windows-amd64.exe .
```

## Docker 开发

### 构建 Docker 镜像

```bash
docker build -f docker/Dockerfile -t warden-dev .
```

### 使用 Docker Compose 开发

```bash
docker-compose up
```

## 相关文档

- [架构设计文档](ARCHITECTURE.md) - 了解系统架构
- [配置文档](CONFIGURATION.md) - 了解配置选项
- [API 文档](API.md) - 了解 API 端点
- [安全文档](SECURITY.md) - 了解安全特性
