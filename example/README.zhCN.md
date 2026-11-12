# Warden 快速开始示例

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

本目录包含两个不同复杂度的 Warden 使用示例，帮助你快速上手。

## 🐳 Docker 镜像

所有示例使用 [GitHub Container Registry (GHCR)](https://github.com/soulteary/warden/pkgs/container/warden) 提供的预构建镜像：

- **默认镜像**: `ghcr.io/soulteary/warden:latest`
- **自定义**: 可通过设置 `WARDEN_IMAGE` 和 `WARDEN_IMAGE_TAG` 环境变量来覆盖镜像

生产环境部署建议使用具体的版本标签而非 `latest`。

## 📚 示例列表

### 1. [简单示例](./basic/) - 基础使用

**适合场景**：
- 快速测试和开发
- 仅使用本地数据文件
- 学习基本功能

**包含内容**：
- ✅ 本地数据文件配置
- ✅ 基本的 Docker Compose 部署
- ✅ 简单的启动脚本
- ✅ 完整的使用文档

**快速开始**：
```bash
cd basic
# 必须设置 API_KEY，否则 Compose 拒绝启动。
# 这是示例命令使用的公开演示密钥，仅限本机测试，其他场景请替换。
echo "API_KEY=your-secret-api-key-here" > .env
# 拉取最新镜像（可选）
docker-compose pull
docker-compose up -d
```

[查看详细文档 →](./basic/README.md)

### 2. [复杂示例](./advanced/) - 完整功能

**适合场景**：
- 使用模拟远程源的隔离开发部署；生产需另行配置 HTTPS、鉴权和网络隔离
- 需要远程 API 数据源
- 完整的监控和测试

**包含内容**：
- ✅ 本地 + 远程数据源
- ✅ Redis 缓存和分布式锁
- ✅ 定时任务自动同步
- ✅ Mock 远程 API 服务
- ✅ 完整的 Docker Compose 配置
- ✅ 自动化测试脚本
- ✅ 多种数据合并策略演示

**快速开始**：
```bash
cd advanced
cp env.example .env
# 拉取最新镜像（可选）
docker-compose pull
docker-compose up -d
```

[查看详细文档 →](./advanced/README.md)

两个 Compose 文件都要求设置 `API_KEY`，且端口仅发布到 `127.0.0.1`。示例中的演示密钥是公开的：修改端口绑定前请先生成自己的密钥（例如 `openssl rand -hex 32`）。`ENVIRONMENT=production` 会拒绝文档中的占位密钥。

## 🎯 选择指南

### 选择简单示例，如果你：
- 是第一次使用 Warden
- 只需要本地数据文件
- 想要快速验证功能
- 在开发环境中测试

### 选择复杂示例，如果你：
- 需要从远程 API 获取数据
- 需要了解完整的数据合并策略
- 想在隔离开发环境中学习远程源配置
- 需要完整的监控和测试方案

## 🚀 快速对比

| 特性 | 简单示例 | 复杂示例 |
|------|---------|---------|
| 本地数据文件 | ✅ | ✅ |
| 远程 API | ❌ | ✅ |
| Redis 缓存 | ✅ | ✅ |
| 定时任务 | ✅ | ✅ |
| Mock API | ❌ | ✅ |
| 测试脚本 | ❌ | ✅ |
| 完整配置 | ❌ | ✅ |
| 文档详细度 | 基础 | 完整 |

## 📖 学习路径

### 初学者路径
1. 从 [简单示例](./basic/) 开始
2. 理解基本概念和配置
3. 测试基本功能
4. 然后查看 [复杂示例](./advanced/) 了解高级功能

### 有经验用户路径
1. 直接查看 [复杂示例](./advanced/)
2. 根据需求调整配置
3. 参考主项目 [README](../README.md) 了解所有功能

## 🔗 相关资源

- [Warden 主文档](../README.md) - 完整的项目文档
- [API 文档](../openapi.yaml) - OpenAPI 规范
- [配置示例](../config.example.yaml) - 配置文件参考
- [代码风格指南](../docs/zhCN/CODE_STYLE.md) - 开发规范

## 💡 提示

- 所有示例都可以独立运行
- 建议先运行简单示例，确保环境配置正确
- 复杂示例使用开发专用的 HTTP 模拟源，生产安全要求需另行核对
- 可以根据实际需求修改配置和数据文件

## ❓ 需要帮助？

如果遇到问题：
1. 查看对应示例的 README 文档
2. 查看 [主项目 README](../README.md) 的故障排查部分
3. 提交 Issue 到项目仓库

---

**祝你使用愉快！** 🎉

