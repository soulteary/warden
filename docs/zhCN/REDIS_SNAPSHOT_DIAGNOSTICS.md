# Redis 快照迁移诊断

新增 `warden_redis_snapshot_operations_total{mode,operation,result}`，分别统计 V2/shadow 读取、发布、续期和租约决策，保留原缓存命中及刷新错误指标。legacy 不产生 V2 操作。标签通过固定白名单收敛，不使用用户、键名、命名空间、版本、URL 或原始错误。

操作为 `read`、`publish`、`renew`、`lease_acquire`、`lease_release`。

读取区分已恢复、缺失、后端不可用、schema/dataset/policy/时间拒绝、超限、损坏、生效集合拒绝和 I/O 错误。`restored` 表示应用完成生效集合验证并接受，而非只解码成功。拒绝恢复日志只包含固定结果码，不包含载荷。

发布区分成功、无租约、来源时间未知、shadow 旧写入失败、`superseded` 及有限错误原因；`superseded` 表示本副本在刷新被拒绝后保留了上一次有效集合，而 Redis 中已有 `source_succeeded_at` 相同或更晚的发布，因此未写入；跳过发布返回 nil 不再被误认为 V2 写成功。续期区分成功、键缺失、修订冲突、租约丢失、超限、损坏和 I/O 错误，不自动重建缺失或冲突值。租约获取区分 acquired/busy/unavailable/错误；释放成功仅表示命令完成，旧所有者安全释放可能未删除任何键。

既有 versioned 健康元数据新增已知的 `input_version`、`effective_version`、`effective_count` 及已知 `redis_revision`，保留兼容 `version`。V1 恢复缺少原始输入证据，省略 `input_version`。生产详情隐藏规则不变。本地元数据和历史成功发布不能证明 Redis 当前键仍存在，应结合操作指标和 Redis 监控判断。

不改变缓存选择、来源年龄、TTL、租约安全、默认 legacy、数据响应结构和 HTTP 状态；细分存储错误保留 invalid/contract/conflict 的 errors.Is 兼容。快照 schema 与持久化字段不变。
