# Redis 8 快照集成测试

CI 在独立 Linux job 中启动全新的 Redis 8 服务：

```sh
WARDEN_REDIS_INTEGRATION_ADDR=localhost:6379 \
WARDEN_REDIS_INTEGRATION_REQUIRE_COLD=true \
go test -race -count=1 -v -run '^TestRedisSnapshotIntegration' ./internal/cache
```

使用隔离的 Redis 8 实例，并允许创建临时 ACL 用户。未指定地址时，普通单元测试跳过这些用例；指定地址后，连接、版本或 ACL 条件不满足均失败，不静默跳过。cold 选项要求空脚本缓存，测试本身不清空数据库或脚本缓存。每个用例使用随机命名空间，只删除自建键和临时 ACL 用户。

覆盖 NOSCRIPT 回退、原子区分缺失与空集、发布恢复时间、仅 TTL 续期、租约竞争和过期、旧所有者写入与释放、旧修订续期、损坏/超限/错类型载荷、脚本 ACL 拒绝及取消读取。不改变应用模式、快照格式或生产业务行为。
