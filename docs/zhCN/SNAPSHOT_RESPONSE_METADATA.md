# 可选快照响应元数据

已认证的数据客户端发送 `Warden-Snapshot-Metadata: v1` 后，列表、分页、单用户及 lookup 响应提供：

- `Warden-Snapshot-Metadata: v1`
- `Warden-Snapshot-Version`：整个生效快照摘要，不是响应体签名
- 已知的 `Warden-Source-Succeeded-At`：最终来源集合验证完成时间
- 已知的 `Warden-Snapshot-Valid-Until`：来源时间加服务端快照最大年龄

时间采用 RFC3339Nano。响应头与查询使用同一不可变代际；元数据位于认证之后，普通请求的 JSON 和状态码不变。所有已认证数据响应增加 `Vary: Warden-Snapshot-Metadata`，使私有缓存区分普通与协商变体；协商响应另加 `Cache-Control: private`。错误状态语义不变，合法 404 可携带判定不存在时使用的快照证据。

旧 Redis 恢复缺少来源时间，省略两个时间头，不用响应或恢复时刻冒充。本地回退只证明最终集合被接受，不证明远端成功。响应头不实施服务端严格读取、不改变 JSON，也不是上游签名版本。客户端仍依赖可信服务与传输，并以原始来源截止时间约束缓存，不能从收包时刻重新计算来源年龄。
