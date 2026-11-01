# 当前数据契约

本文记录快照存储或读取策略变更前的现有行为，不新增配置、接口或运行行为。

## 合并与撤权

`ONLY_REMOTE` 和 `ONLY_LOCAL` 选择单一来源类型；其他五种模式在两个来源成功时
保留来源独有记录。`REMOTE_FIRST` 不会用远程名单全量替换本地名单；远程名单作为
唯一权威来源时应使用 `ONLY_REMOTE`。

当前加载路径中，同一手机号/邮箱键的冲突结果如下（通用明文路径使用
parser-kit v3.0.2，加密远程响应使用 v2 envelope）：

| 模式 | 明文保留记录 | 加密保留记录 |
| --- | --- | --- |
| `DEFAULT` | 本地 | 远程 |
| `REMOTE_FIRST` | 远程 | 远程 |
| `ONLY_REMOTE` | 远程 | 远程 |
| `ONLY_LOCAL` | 本地 | 本地（应用不传远程 URL） |
| `LOCAL_FIRST` | 远程 | 本地 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | 远程 | 远程 |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | 远程 | 本地 |

`DEFAULT` 及两种本地优先模式在两条加载路径中结果不同，明文结果也与模式文档描述的
优先级不同。这是现有兼容契约，不是模式名称的预期定义。同键本地和远程记录冲突时，
切换传输方式可能改变 status、role、scope 等最终字段，不能认为加密只改变线上格式。
修正这些差异需要独立的行为变更评审和迁移说明。

`internal/loader/transport_merge_contract_test.go` 中的 `TestTransportMergeContract`
经 HTTP 明文及真实 v2 envelope 加解密加载相同记录，覆盖手机号键和仅邮箱键，
比较完整用户字段、保留来源独有用户，并明确断言两条路径何时相同、何时不同。
测试不比较结果顺序和时间戳。`ONLY_LOCAL` 与 `App` 一样不传远程 URL，
不代表验证了向底层 loader 直接传远程 URL 的行为。

`TestBaselinePlaintextMergeModeContract` 继续保留明文基线；现有
`TestLoadWithResult_RemoteFailure_Modes` 和严格模式测试覆盖刷新失败。
这些测试不改变加载或授权行为。

## 查询与健康检查

`SNAPSHOT_MAX_AGE` 影响健康检查，不限制数据查询。`ONLY_REMOTE` 和
`REMOTE_FIRST` 中，旧快照可使健康接口返回 HTTP 503，而通过鉴权的列表及用户接口
仍返回缓存数据。查询成功不能证明数据源最近校验过。

成功加载的空名单是有效数据：列表返回 `[]`，不存在的用户返回 404，新鲜空快照可以
保持健康。这与从未获得有效集合不同。`baseline_contract_test.go` 中的
`TestBaselineSnapshotReadContract` 覆盖陈旧非空及新鲜空快照的全部健康和查询路由别名。

## 校验与共享缓存恢复

身份冲突拒绝整次刷新并保留旧集合；逐条格式校验可丢弃记录。非空输入全部被拒绝时，
`EMPTY_RULESET_POLICY` 决定采纳空集还是保留旧集合。来源真正返回空集时，两种策略都
会生效。`empty_ruleset_policy_test.go` 覆盖这些场景及 Redis 发布和恢复流程，详见
[配置说明](CONFIGURATION.md#空规则集策略)。

Redis 保存生效用户，不保存原始来源时间。恢复时 `LoadedAt` 记录本副本的采纳时间，
不代表最近一次成功校验上游的时间。后台刷新失败不会推进旧快照时间。
共享缓存 TTL 不构成来源新鲜度保证。

## 调用方缓存与会话

Go SDK 可缓存 `GetUsers` 列表响应，默认选项 TTL 为五分钟；缓存命中时可以不访问
Warden。用户状态仍需调用方校验。Warden 不会使已签发的网关会话自动失效；调用方的
缓存和重验证策略决定撤权何时到达已有会话。Warden 单独不提供端到端撤权时限保证。
