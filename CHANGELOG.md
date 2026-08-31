# Changelog

本项目变更记录遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## 维护流程

- 每次改动在 `Unreleased` 段落记录，按 `Added` / `Changed` / `Fixed` / `Security` / `Removed` 分类。
- 发布时（打 `vX.Y.Z` tag，触发 `.github/workflows/release.yml`）：
  1. 将 `Unreleased` 内容移动到新版本段落并标注日期；
  2. 重新建立空的 `Unreleased` 段落；
  3. 提交后再打 tag。
- Release 说明会引用本文件；请保持条目简洁、面向使用者。

## [Unreleased]

## [1.2.0] - 2026-08-31

### Added
- 健康检查新增 `snapshot` 与 `snapshot_freshness` 检查，暴露低基数的数据来源、版本、加载时间、连续刷新失败次数与稳定原因码。
- 新增 `SNAPSHOT_MAX_AGE`，用于限制严格远程模式可接受的快照年龄；默认值为 `max(30s, 3 × task interval)`。

### Changed
- `REMOTE_FIRST` 与 `ONLY_REMOTE` 刷新失败时严格失败并保留最后一次成功快照；`REMOTE_FIRST_ALLOW_REMOTE_FAILED` 才会回退本地数据并将健康状态标记为 `degraded`。
- 多副本刷新改为每个实例独立刷新进程内缓存与快照，仅共享 Redis 写入使用分布式锁，避免非锁持有实例的快照过期。
- GitHub Actions 与发布依赖升级；release workflow 支持对已有 tag 手动重跑发布。

### Fixed
- 配置 Redis 且启用 HMAC v2 时，将 Redis replay guard 保持为关键依赖；Redis 不可用或刷新锁异常时不再静默降低防重放保证。
- 修正生产默认远程 URL 的启动校验、SDK 空白 HMAC 凭据处理、响应体大小与重试延迟溢出边界。
- 移除 URL 校验测试的外部 DNS 依赖，并修正调度器跨月测试边界。
- 修正 Release 正文中的容器镜像标签，使拉取命令与实际生成的无 `v` semver 标签一致。

### Security
- 生产环境要求每个 HMAC secret 至少包含 32 个原始字节。
- 严格远程模式的未知或过期快照返回不健康，防止 Redis 引导占位状态被误报为新鲜数据。

## [1.1.0] - 2026-08-26

### Changed
- Go 工具链与构建镜像升级至 Go 1.27.0。

## [1.0.0] - 2026-08-26

### Security
- HMAC v2：canonical request 绑定 `X-Key-Id`，并以 nonce 和共享 Redis replay guard 阻止窗口内重放；旧 HMAC v1 默认关闭，仅可通过 `WARDEN_HMAC_ALLOW_V1=true` 临时开启。
- TLS/mTLS：证书、私钥、客户端 CA 与强制客户端证书配置必须完整，部分配置会在启动前失败，不再静默回退到 HTTP。
- 配置：生产环境使用独立的 `ENVIRONMENT` 判定安全策略；无 API Key、HMAC 或 mTLS 时拒绝启动，非法或空 HMAC key set 整体拒绝。
- 远程配置：生产环境强制使用加密 envelope v2，并检查解密开关、私钥及格式组合，避免配置组合绕过加密策略。
- 身份状态：缺失 `status` 的记录不再自动激活，改为按 `inactive` 处理；允许访问必须显式设置 `active`。
- 错误与健康：生产错误脱敏不再依赖数据合并模式；严格刷新失败会降级健康状态。
- SDK：不完整 HMAC 配置会返回配置错误；响应体超限会明确报错；重试使用有首轮延迟的指数退避。
- 容器：运行阶段以非 root 用户（uid/gid 10001）运行；基础镜像固定到明确版本并说明可选 digest 固定；`-trimpath` 构建去除本地路径信息。
- 容器：UPX 压缩默认关闭，改为显式 build arg `ENABLE_UPX=1`。
- 构建上下文：收紧 `.dockerignore`，排除源码无关文件、密钥、证书、真实数据与已提交产物。
- CI：新增容器镜像漏洞扫描（Trivy，HIGH/CRITICAL 失败）、SBOM（SPDX）生成、非 root 运行校验。
- Release：容器镜像新增 keyless cosign 签名与镜像 SBOM，校验和覆盖全部发布产物。

### Changed
- HMAC v2 canonical request 定义为 `METHOD`、escaped path/query、Key ID、timestamp、nonce、body SHA-256，各字段以换行分隔。
- 配置变量拆分为 `ENVIRONMENT`（部署安全策略）与 `MERGE_MODE`（数据合并策略）；旧 `MODE` 仅作为迁移兼容入口。
- Go 工具链与 lint 配置统一到 Go 1.26；共享 kit 依赖升级到兼容 Fiber v3 的 v2 模块线。
- Compose 镜像、README 和 release 标签策略统一；稳定标签更新 `latest`，预发布标签不会更新。
- Docker/Compose 中 Redis 从已 EOL 的 `6.2.4` 升级至受支持的 `7.4-alpine`（Warden 仅使用基础命令，向后兼容）。

### Removed
- 从版本库移除误提交的编译二进制 `example/advanced/mock-api/mock-api`，改由 `make mock-api` 或 Docker 构建生成。

### Added
- OpenAPI 3.1 契约声明 API Key、Bearer、HMAC v2 与 mTLS 安全方案。
- HMAC v2、配置拆分和远程加密 v2 迁移指南。
- `Makefile`：提供 `build`、`mock-api`、`vet`、`test-race`、`govulncheck`、`sbom`、`docker` 等目标。
- `.github/dependabot.yml`：自动更新 Go 依赖、GitHub Actions 与 Docker 基础镜像。
- `docs/RELEASE_SECURITY.md`：发布与分支保护建议。
