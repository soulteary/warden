# SDK 用户缓存隔离

Cache.Set 与 Cache.Get 同时复制外层用户列表和各用户的 Scope。Set 返回后修改输入，或修改 Get 返回的 Scope，不再影响内部缓存或其他读取者。

用户顺序、Scope 的 nil/空集合区别、TTL、零 TTL 和 Clear 语义不变；Set(nil) 仍缓存非 nil 的空列表。调用者仍须同步 Set 读取期间对自身输入的并发修改。
