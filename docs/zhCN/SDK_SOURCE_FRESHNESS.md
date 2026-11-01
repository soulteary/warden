# SDK 可选来源新鲜度约束

默认保持兼容：不请求来源元数据，不检查来源年龄，沿用本地 CacheTTL 和状态处理。

```go
opts := warden.DefaultOptions().WithBaseURL("https://warden.example").
    WithSourceFreshness(2*time.Minute, 5*time.Second)
client, err := warden.NewClient(opts)
```

最大来源年龄为正时启用。第二个时长是相对时钟偏差裕量，从截止时间扣除，不作为额外保旧时间。两个值须非负，启用时最大年龄须大于裕量；最大年龄为零则关闭。启用后 WithClock 同时控制来源和缓存时间，自定义时钟须支持并发调用，运行环境应保持时钟同步。

先部署服务端可选快照响应元数据，再开启 SDK 约束。数据请求协商 `Warden-Snapshot-Metadata: v1`，要求快照摘要、来源成功时间及有效截止时间。旧服务端或 V1 恢复缺失、重复、损坏的证据会被拒绝，不填入收包时刻；超出时钟裕量的未来来源时间也拒绝。

有效截止为 `min(服务端截止时间, 来源成功时间 + 调用方最大年龄) - 裕量`。列表缓存再取其与本地 CacheTTL 的较早值。收包、重试、相同内容响应和解码均不延长来源期限。读取/解码期间过期的响应不返回、不缓存，也不删除其他并发请求的较新缓存代际。零 CacheTTL 仍不复用缓存。

列表、分页、单用户查询都检查证据。新鲜 404 保持 NOT_FOUND；年龄未知或陈旧的 404 返回 SOURCE_FRESHNESS，不把旧数据中的缺失当作可靠结论。401/503 语义不变，不新增过期缓存回退。CheckUserInList 仍在错误时返回 false；需要区分原因时使用查询方法，读取 Error.Code 或 errors.Is 的 ErrSourceUnknown、ErrSourceStale、ErrSourceFuture。

此功能约束被接受的最终集合年龄，不代表上游内容刚产生或远端一定成功。响应头依赖可信服务和传输，摘要不是响应体签名；本地回退证据含义不变。下游会话期限需要单独约束，不因此启用服务端严格读取或修改 JSON。
