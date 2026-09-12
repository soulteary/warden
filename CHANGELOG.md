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

### Added
- 新增环境变量 `EMPTY_RULESET_POLICY`，把「加载成功、返回了记录、但这些记录全部未通过逐条格式校验」
  这一歧义状态拆成两种可选策略：
  - `consistency-first`（默认，保持历史行为）：应用并发布空集，使各副本与共享缓存严格一致；合法的
    全员撤权立刻传播，上游格式变更也会立刻暴露为「全部拒绝」而不被陈旧数据掩盖。
  - `availability-first`：视为一次刷新失败，保留上一次有效的规则集，并按同一份数据续期共享 Redis
    缓存（否则它会随 `REDIS_CACHE_TTL` 过期，而重启的副本正是该策略要覆盖的场景）；失败计数递增、
    快照年龄继续增长、健康检查转为 degraded，`warden_refresh_failures_total` 记 `all_records_rejected`。
  数据源真的返回 0 条记录时不属于歧义场景，两种策略下都会照常生效，「撤销所有人」始终可用。
  启动阶段通常先从共享 Redis 引导；若首次读取临时失败、后续数据又全部未通过格式校验，可用性优先会在
  取得写入者锁后重试读取，恢复后从共享集合启动，仍不可读时也绝不写入空集，以免覆盖可能仍存在的最后
  有效数据。`MERGE_MODE=ONLY_LOCAL` 跳过常规 Redis 优先读取，因此直接执行同一保护流程。若数据同时存在
  身份冲突和格式错误，身份完整性错误优先报告，不会被误记为 `all_records_rejected`。
  从 Redis 成功引导时会建立 `source=redis` 的 degraded 快照，使严格模式从引导时开始计算快照新鲜度；
  若启动始终没有取得已知有效集合，首次后台刷新会再次读取 Redis，仍失败时跳过共享缓存写入，绝不续期
  进程的零值空缓存。启动时成功加载到真正的空集会立即应用、记录快照并发布到 Redis，不等待后台周期。
  Redis 返回空切片时进一步通过 `Exists` 区分合法存储的空集与键不存在：合法空集作为已知有效快照直接
  生效并保持健康，不会回退到可能陈旧的数据源；只有缓存未命中才继续加载其他来源。
  对 Redis 中非空但被当前格式校验全部过滤的旧数据同样遵循所选策略：一致性优先采纳生效空集并记录
  Redis 来源，可用性优先则拒绝其作为引导基线并继续尝试配置的数据源。
  非法取值在 `cmd.ValidateConfig` 阶段直接导致启动失败，避免运维以为选了可用性优先、实际仍是一致性优先。
- 新增 `cache.AcceptableCount`：在不写入缓存的前提下预测一份规则集的生效条数。判定必须发生在换入
  之前——缓存的 `Set` 是唯一的原子换入点，先写再回滚会让并发读者在这段窗口内看到空的允许列表。

### Fixed
- 修正后台刷新的变更检测：此前将加载到的原始记录数/哈希与缓存中「经校验去重后」的记录数/哈希比较，
  只要规则集中存在任意一条被格式校验或去重丢弃的记录，两者就永远不相等，导致每个周期都被判定为
  「数据已变化」并重复整体换入，且共享 Redis 缓存在启动后不再被刷新。现改为用 loader 已计算的
  `LoadResult.Version` 与快照版本比较（同为「加载到的原始集合」的哈希）。
- 后台刷新与启动加载写入共享 Redis 缓存时，改为写入缓存实际持有的生效集合，而不是原始加载结果；
  被格式校验拒绝的记录不会再被投递到其他副本。
- 成功刷新日志新增 `applied_count` 字段，与 `count`（加载条数）并列，使记录被丢弃的情况可观测；
  当加载到的记录全部未通过格式校验时额外输出 Warn 级别日志（此前该状态只在 Debug 级别可见）。
- 修正 `pkg/gocron` 中 `TestScheduler_WeekdaysTodayAfter` 的时间依赖缺陷：它用 `now.Minute()-1`
  构造「今天已过去的时刻」，在 00:00 这一分钟内该运算会回退到前一天 23:59，从而改变星期并使前提
  失效（23:59 今天尚未到来），导致每天有一分钟窗口必然失败。现改为在时刻上做减法，并按调度器契约
  （今天这个星期几、该时刻的第一个严格晚于当前时间的 occurrence）推导期望值，而不是硬编码「+7 天」。

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
- `/metrics` 文档补充认证矩阵；OpenAPI 契约改为同时声明匿名与各认证方案两种形态（此前固定为
  `security: []`，等于告诉生成的客户端与网关「该端点永不接受凭据」，与生产默认矛盾），并补充 `401` 响应。
  同时修正该端点的响应示例：原示例中的 `http_requests_total{path=...}`、`cache_size` 与实际导出的
  `warden_http_requests_total{endpoint=...}`、`warden_user_cache_size` 不符。
- `/metrics` 认证策略改为按部署环境取默认值：`ENVIRONMENT=production` 默认要求认证，其他环境默认匿名；
  `WARDEN_METRICS_REQUIRE_AUTH` 在两个方向上均可覆盖默认值。此前文档称默认匿名，但匿名分支沿用了服务
  API Key，只要配置了 `API_KEY`，`/metrics` 实际返回 `401`，Prometheus 抓取会静默失败。
- 补齐 de/fr/it/ja/ko 五种语言缺失的 18 个翻译键，并翻译此前在所有语言（含中文）中都保持英文原文的
  6 个 `http.*` 面向用户的错误消息。
- 新增 `locales` 包的完整性测试：校验各语言键集合与 `en.json` 一致、printf 占位符序列一致、且不存在
  与英文完全相同的未翻译值。
- CI 覆盖率报告改用 `soulteary/go-test-report-action` 取代 Codecov：该 action 自行运行测试、
  统计覆盖率并执行 80% 阈值门禁，同时产出 Markdown 报告、SVG 徽章与 JSON。PR 分支只校验不回写；
  默认分支由新增的 `.github/workflows/go-test-report.yml` 回写 `.github/go-test-report.md` 与
  `.github/coverage.svg`，与仓库既有的 `go-reportcard.yml` 分工一致。各语言 README 的覆盖率徽章
  改为指向仓库内的 `.github/coverage.svg`，不再依赖外部服务与 `CODECOV_TOKEN`。

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
