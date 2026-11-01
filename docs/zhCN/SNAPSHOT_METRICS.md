# 快照年龄指标

应用 `/metrics` 在抓取时计算 `warden_snapshot_age_seconds`，即使刷新调度停止也会增长。保留原指标名与 gauge 类型，负年龄归零。年龄基准遵守既有模式：legacy/shadow 使用 LoadedAt，V2 使用持久化来源成功时间；恢复和 TTL 续期不重置 V2 年龄。

一次抓取固定同一个不可变快照根，用于年龄、降级状态和时间戳。每个 handler 独立持有来源，不替换全局回调。包级 registry 保留事件更新的兼容 gauge，应用 handler 将其导出样本替换为实时采集结果。

新增无标签指标：
- `warden_snapshot_loaded_timestamp_seconds` / `warden_snapshot_age_known`
- `warden_snapshot_source_succeeded_timestamp_seconds` / `warden_snapshot_source_time_known`
- `warden_snapshot_published_timestamp_seconds` / `warden_snapshot_publication_time_known`
- `warden_snapshot_restored_timestamp_seconds` / `warden_snapshot_restoration_time_known`

未知时间输出零，对应 known 指标也为零。旧 Redis 恢复不能提供来源成功证据。仅在 `source_time_known == 1` 时使用 `time() - source_succeeded_timestamp_seconds` 计算来源年龄，不把未知时间理解为来源新鲜。指标不改变查询新鲜度执行方式，也不证明共享 Redis 键当前仍存在。
