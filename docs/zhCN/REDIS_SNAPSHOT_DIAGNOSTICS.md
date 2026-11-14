# Redis 快照迁移诊断

新增 `warden_redis_snapshot_operations_total{mode,operation,result}`，分别统计 V2/shadow 读取、发布、续期和租约决策，保留原缓存命中及刷新错误指标。legacy 不产生 V2 操作。标签通过固定白名单收敛，不使用用户、键名、命名空间、版本、URL 或原始错误。

操作为 `read`、`publish`、`renew`、`lease_acquire`、`lease_release`。

读取区分已恢复、缺失、后端不可用、schema/dataset/policy/时间拒绝、超限、损坏、生效集合拒绝和 I/O 错误。`restored` 表示应用完成生效集合验证并接受，而非只解码成功。拒绝恢复日志只包含固定结果码，不包含载荷。

发布区分成功、无租约、来源时间未知、shadow 旧写入失败、`superseded` 及有限错误原因；`superseded` 表示本副本在刷新被拒绝后保留了上一次有效集合，而 Redis 中已有 `source_succeeded_at` 相同或更晚的发布，因此未写入；跳过发布返回 nil 不再被误认为 V2 写成功。续期区分成功、键缺失、修订冲突、租约丢失、超限、损坏和 I/O 错误。续期结果为 `missing` 时，本副本在同一周期内重新发布自己持有的快照，记为 `publish`/`published`（或与其他发布相同的 `superseded`、错误原因），本地 `redis_revision` 随之指向新发布；写入的 `source_succeeded_at` 仍是原值，不会让数据显得更新。键可能因过期、淘汰或清库而消失；此时 Redis 中没有任何发布可被覆盖，而不写入则在来源持续使本副本保留旧集合期间 Redis 一直为空，重启的副本无可恢复。`revision_conflict` 则不同：键中是其他写入者在本副本之后的发布，既不续期也不替换，并返回续期错误。存储值超过大小上限（`too_large`）或无法被 Redis 解码为 JSON 对象或数组（`invalid_envelope`）时，按键缺失处理：任何副本都无法恢复它，它也不是一次发布，因此本副本在同一周期内重新发布自己持有的快照，规则和结果与 `missing` 之后相同。未持有修订号的副本同样会替换这样的值。判定由 Redis 内的 JSON 解码器作出：能被它解码但没有 `revision` 或修订号不同的值仍是 `revision_conflict`，即使 Warden 读取时会拒绝它（空对象、未知格式）。租约获取区分 acquired/busy/unavailable/错误；释放成功仅表示命令完成，旧所有者安全释放可能未删除任何键。

既有 versioned 健康元数据新增已知的 `input_version`、`effective_version`、`effective_count` 及已知 `redis_revision`，保留兼容 `version`。V1 恢复缺少原始输入证据，省略 `input_version`。生产详情隐藏规则不变。本地元数据和历史成功发布不能证明 Redis 当前键仍存在，应结合操作指标和 Redis 监控判断。

不改变缓存选择、来源年龄、TTL、租约安全、默认 legacy、数据响应结构和 HTTP 状态；细分存储错误保留 invalid/contract/conflict 的 errors.Is 兼容。快照 schema 与持久化字段不变。
