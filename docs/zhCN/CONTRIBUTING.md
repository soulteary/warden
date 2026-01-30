# 贡献指南

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

感谢你对 Warden 项目的关注！我们欢迎所有形式的贡献。

## 📋 目录

- [如何贡献](#如何贡献)
- [开发环境设置](#开发环境设置)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [问题报告与功能请求](#问题报告与功能请求)

## 🚀 如何贡献

你可以通过以下方式贡献：

- **报告 Bug**: 在 GitHub Issues 中报告问题
- **提出功能建议**: 在 GitHub Issues 中提出新功能想法
- **提交代码**: 通过 Pull Request 提交代码改进
- **改进文档**: 帮助改进项目文档
- **回答问题**: 在 Issues 中帮助其他用户

参与本项目时，请尊重所有贡献者，接受建设性的批评，并专注于对项目最有利的事情。

## 🛠️ 开发环境设置

### 前置要求

- Go 1.25 或更高版本
- Redis（用于测试）
- Git

### 快速开始

```bash
# 1. Fork 并克隆项目
git clone https://github.com/your-username/warden.git
cd warden

# 2. 添加上游仓库
git remote add upstream https://github.com/soulteary/warden.git

# 3. 安装依赖
go mod download

# 4. 运行测试
go test ./...

# 5. 启动本地服务（确保 Redis 正在运行）
go run .
```

## 📝 代码规范

请遵循以下代码规范：

1. **遵循 Go 官方代码规范**: [Effective Go](https://go.dev/doc/effective_go)
2. **格式化代码**: 运行 `go fmt ./...`
3. **代码检查**: 使用 `golangci-lint` 或 `go vet ./...`
4. **编写测试**: 新功能必须包含测试
5. **添加注释**: 公共函数和类型必须有文档注释
6. **常量命名**: 所有常量必须使用 `ALL_CAPS` (UPPER_SNAKE_CASE) 命名风格

详细的代码风格指南请参考 [CODE_STYLE.md](CODE_STYLE.md)。

## 📦 提交规范

### Commit Message 格式

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `style`: 代码格式调整（不影响代码运行）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 示例

```
feat(cache): 添加 Redis 缓存支持

实现了基于 Redis 的分布式缓存，支持数据持久化和多实例共享。

Closes #123
```

```
fix(router): 修复分页参数验证问题

修复了当 page_size 超过最大值时返回错误状态码的问题。

Fixes #456
```

## 🔄 Pull Request 流程

### 创建 Pull Request

```bash
# 1. 创建功能分支
git checkout -b feature/your-feature-name

# 2. 进行更改并提交
git add .
git commit -m "feat: 添加新功能"

# 3. 同步上游代码
git fetch upstream
git rebase upstream/main

# 4. 推送分支并创建 PR
git push origin feature/your-feature-name
```

### Pull Request 检查清单

在提交 Pull Request 之前，请确保：

- [ ] 代码遵循项目代码规范
- [ ] 所有测试通过（`go test ./...`）
- [ ] 代码已格式化（`go fmt ./...`）
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] Commit message 遵循 [提交规范](#提交规范)
- [ ] 代码已通过 lint 检查

所有 Pull Request 都需要经过代码审查，请及时响应审查意见。

## 🐛 问题报告与功能请求

在创建 Issue 之前，请先搜索现有的 Issues，确认问题或功能未被报告。

### Bug 报告模板

```markdown
**描述**
清晰简洁地描述 Bug。

**复现步骤**
1. 执行 '...'
2. 看到错误

**预期行为**
清晰简洁地描述你期望发生什么。

**实际行为**
清晰简洁地描述实际发生了什么。

**环境信息**
- OS: [e.g. macOS 12.0]
- Go 版本: [e.g. 1.25]
- Redis 版本: [e.g. 7.0]
```

### 功能请求模板

```markdown
**功能描述**
清晰简洁地描述你想要的功能。

**问题描述**
这个功能解决了什么问题？为什么需要它？

**建议的解决方案**
清晰简洁地描述你希望如何实现这个功能。
```

## 🎯 开始贡献

如果你想贡献但不知道从哪里开始，可以关注：

- 标记为 `good first issue` 的 Issues
- 标记为 `help wanted` 的 Issues
- 代码中的 `TODO` 注释
- 文档改进（修复错别字、改进清晰度、添加示例）

如有问题，请查看现有的 Issues 和 Pull Requests，或在相关 Issue 中提问。

---

再次感谢你对 Warden 项目的贡献！🎉
