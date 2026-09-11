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

### Fixed
- 修正后台刷新的变更检测：此前将加载到的原始记录数/哈希与缓存中「经校验去重后」的记录数/哈希比较，
  只要规则集中存在任意一条被格式校验或去重丢弃的记录，两者就永远不相等，导致每个周期都被判定为
  「数据已变化」并重复整体换入，且共享 Redis 缓存在启动后不再被刷新。现改为用 loader 已计算的
  `LoadResult.Version` 与快照版本比较（同为「加载到的原始集合」的哈希）。
- 后台刷新与启动加载写入共享 Redis 缓存时，改为写入缓存实际持有的生效集合，而不是原始加载结果；
  被格式校验拒绝的记录不会再被投递到其他副本。
- 成功刷新日志新增 `applied_count` 字段，与 `count`（加载条数）并列，使记录被丢弃的情况可观测；
  当加载到的记录全部未通过格式校验时额外输出 Warn 级别日志（此前该状态只在 Debug 级别可见）。

### Security
- 根路径 `/` 改为精确匹配（`/{$}`），未注册路径由独立的兜底处理器返回本地化 JSON `404`。
  此前 `/` 为子树通配，任意未匹配路径（如 `/foo`、`/user/`）都会返回完整允许列表。
- Prometheus `endpoint` 与 `method` 标签改为按白名单归一化，未知取值统一计入 `other`。指标中间件位于
  认证之前，此前未认证调用方即可通过请求任意路径、或发送任意 HTTP 方法令牌（net/http 会原样透传给
  处理器），持续创建新的时间序列（标签基数无上界）。
- 路由注册改用独立的 `*http.ServeMux` 并显式设置为 `http.Server.Handler`，不再使用
  `http.DefaultServeMux`；注册过程恢复幂等，其他被链接进二进制的包（如 `net/http/pprof`）也无法
  再向服务对外的 mux 挂载路由。

### Changed
- 工具链版本声明统一到 Go 1.27：`.golangci.yml` 的 `run.go` 由 `1.26` 提升为 `1.27`，与 `go.mod` 的
  `go 1.27.0` 一致，避免 linter 按更旧的语言版本分析代码；Issue 模板与贡献指南中的 Go 版本示例
  同步更新。（`go.mod`、Dockerfile、README 徽章与各语言部署文档此前已为 1.27。）
- 主模块与 `example/advanced/mock-api` 新增 `toolchain go1.27.1` 指令，与构建镜像
  `golang:1.27.1-alpine3.24` 对齐。`go` 指令仍为 `1.27.0`（最低语言版本），`toolchain` 作为建议下限：
  本地 Go 更旧时会自动获取 1.27.1，更新时则直接使用本地工具链。使用 `GOTOOLCHAIN=local` 且本地
  低于 1.27.1 的环境需要自行升级 Go。
- `/metrics` 文档补充认证矩阵，OpenAPI 契约补充 `401` 响应；此前 API 文档声称该端点「不需要认证」。
- `/metrics` 认证策略改为按部署环境取默认值：`ENVIRONMENT=production` 默认要求认证，其他环境默认匿名；
  `WARDEN_METRICS_REQUIRE_AUTH` 在两个方向上均可覆盖默认值。此前文档称默认匿名，但匿名分支沿用了服务
  API Key，只要配置了 `API_KEY`，`/metrics` 实际返回 `401`，Prometheus 抓取会静默失败。
- 补齐 de/fr/it/ja/ko 五种语言缺失的 18 个翻译键，并翻译此前在所有语言（含中文）中都保持英文原文的
  6 个 `http.*` 面向用户的错误消息。
- 新增 `locales` 包的完整性测试：校验各语言键集合与 `en.json` 一致、printf 占位符序列一致、且不存在
  与英文完全相同的未翻译值。

### Removed
- 移除已不再使用的翻译键 `log.data_modified_during_update`。

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
- 修正加密远程源下 `DEFAULT` 的本地容错回退，以及显式 `--mode` 被 `MERGE_MODE` 环境变量覆盖的优先级错误。

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
