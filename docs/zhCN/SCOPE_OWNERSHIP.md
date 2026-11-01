# Scope 所有权与驻留内存

`PrepareRules` 在身份校验前复制输入外层数组和非 nil 的 Scope。候选输入与有效索引
分别拥有 Scope；私有不可变 ReadView rows 只与对应索引共享 Scope。缓存的 Get、Page、
Iterate 回调和索引查询会复制返回的 Scope。快照冻结也隔离输入来源记录，同时保留原
不可变 ReadView。SDK 的缓存写入和读取另行深拷贝 Scope。调用方仍需同步自己在接口
读取期间对输入的修改。

`ValidateAndIndexUsers` 不修改调用者记录，返回独立外层数组，但历史上并不深拷贝
已有 Scope。owned 接口消费私有外层数组：先规范化、再校验，成功后可原地重排；出错
时保留规范化后的原位置。嵌套切片由调用者负责，Prepare 已在该边界前完成深拷贝。

owned 的空结果非 nil，结果可见容量等于长度；容量裁剪并不释放原 backing array。
16 条记录、容量 65,536 的非空输入仍保留整个原数组。Prepare 按实际长度克隆，因此
该生产路径没有额外的大容量驻留。本次不新增自动复制阈值，也不扩展 Scope 共享。

测试覆盖重叠 Scope、零长但有容量、nil、候选拒绝、重复发布和旧视图；置换及历史哈希
差分继续验证列表顺序和双版本。Linux 测量命令与边界见
[英文所有权说明](../enUS/SCOPE_OWNERSHIP.md)及[分阶段基准](../enUS/CANDIDATE_BENCHMARKS.md)。
GC 后驻留指标不是峰值、RSS 或固定内存门禁。
