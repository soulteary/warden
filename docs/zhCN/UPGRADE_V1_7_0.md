# Warden v1.7.0 升级与回滚

本版本新增限流与 Redis ACL/TLS 连接的环境变量，并收紧若干安全默认行为。部分在
v1.6.3 下可以启动的配置现在会被拒绝，少数响应也有变化，生产升级前请逐项核对。
公开 SDK 方法签名、HMAC 协议、Redis 快照格式及已存数据均不变，不需要转换数据。
Go 模块现声明 Go 1.27.4。

## 新增配置

- `RATE_LIMIT`、`RATE_LIMIT_WINDOW` 覆盖配置文件中的 `rate_limit.rate` 和
  `rate_limit.window`。窗口接受时长或秒数，最小 1 秒；无效值会终止启动。默认仍为
  每个客户端 IP 每分钟 60 次。
- `REDIS_USERNAME` 指定 Redis ACL 用户；`REDIS_TLS_ENABLED`、`REDIS_TLS_CA_FILE`、
  `REDIS_TLS_SERVER_NAME` 配置到 Redis 的 TLS 连接。服务端证书始终校验。TLS 默认
  关闭，CA 文件和服务器名称要求 `REDIS_TLS_ENABLED=true`。

## 升级前核对

以下情况现在会拒绝启动：

- 启用 Redis 且配置了 `REDIS_PASSWORD_FILE` 或 `redis.password_file`，但文件不可读
  或为空。此前该错误被忽略，Warden 会不带密码连接 Redis。
- `TRUSTED_PROXY_IPS` 中存在既非 IP 也非 CIDR 的条目（如主机名、带端口的地址）。
  此前这类条目会被丢弃。
- `ENVIRONMENT=production` 且 `API_KEY` 为文档或示例中的占位值。请生成随机密钥，
  例如 `openssl rand -hex 32`。短于 16 个字符的密钥会记录警告。
- `RATE_LIMIT`、`RATE_LIMIT_WINDOW` 或 `REDIS_TLS_ENABLED` 的值无效，或配置文件中
  `rate_limit.rate` 为负数、`rate_limit.window` 小于 1 秒。

以下请求处理行为发生变化：

- 配置文件中的 `rate_limit.rate` 和 `rate_limit.window` 现在生效。此前它们被解析，
  但限流始终按每分钟 60 次执行；文件中设置了其他值的部署会看到新值生效。
- `/metrics` 和 `/log/level` 纳入限流及 10 KB 请求体限制，并与数据接口共用同一
  客户端配额。抓取频率高于限流配置的采集器会收到 429。健康检查接口不限流。
- 带 HMAC 请求头且请求体超过 10 KB 的请求会在验签之前被拒绝。`Content-Length`
  超限的 `POST`、`PUT`、`PATCH`、`DELETE` 请求由请求体限制中间件返回 413，与此前
  一致；带请求体的 `GET`、`HEAD`、`OPTIONS` 请求，以及没有 `Content-Length` 的分块
  请求返回 401，此前它们的请求体会被完整读入。
- `RESPONSE_FIELDS` 现在作用于 `/v1/lookup`。`user_id`、`status`、`channel_hint`
  和 `destination` 对象始终返回。设置了 `RESPONSE_FIELDS` 并通过 `/v1/lookup`
  投递验证码的部署，必须包含 `mail` 和/或 `phone`。白名单为空时响应不变。
  `mail` 控制 `destination.email`，`phone` 控制 `destination.phone`，`name` 和
  `dingtalk_userid` 控制同名字段；通过钉钉投递验证码的部署还必须包含
  `dingtalk_userid`，文档中的示例值 `phone,mail,user_id,status,name` 不含该字段。
  字段名按原样精确匹配并区分大小写，`email`、`Mail` 这类未知名称不会报错，对应
  字段会被省略。
- 被限流的请求返回 429，响应体为纯文本 `Too Many Requests`，不带 `Retry-After`。
  生产环境下响应体是 JSON 对象 `{"error":"..."}`，但 `Content-Type` 仍为
  `text/plain`。
- 链路追踪属性 `http.url` 不再包含查询字符串。
- 1 KB 及以上的列表、用户和 lookup 响应在压缩时会带上 `Content-Encoding: gzip`。
- 新建的审计日志文件权限为 0600，新建的日志目录为 0700；已有文件和目录保持原
  权限。以其他账号读取审计日志的进程需要显式授权。

设置了 `IP_WHITELIST` 而 `TRUSTED_PROXY_IPS` 为空时，启动会记录警告。默认行为
不变：此时信任来自任意私网或回环对端的转发地址。建议将 `TRUSTED_PROXY_IPS`
设置为反向代理的确切地址。

`example/basic` 和 `example/advanced` 的 Compose 文件现在要求设置 `API_KEY`，
且端口仅发布到 `127.0.0.1`。

## 依赖

Go 更新至 1.27.4，修复 `net/http` 中的 GO-2026-6859；构建镜像为
`golang:1.27.4-alpine3.24`。共享 kit 更新至当前发布版本，具体版本见
[CHANGELOG](../../CHANGELOG.md)。部分 kit 的 Fiber 适配器现选用 Fiber v3.7.0；
Warden 使用 `net/http` 提供服务，不编译 Fiber 包。

## 升级与回滚

1. 服务升级为 v1.7.0；SDK 使用方需要 Go 1.27.4 或更高版本，执行
   `go get github.com/soulteary/warden@v1.7.0` 并重建。仅更新后端不会替换其他
   程序中已经编译的 SDK 或依赖。
2. 保留现有 Redis 数据和命名空间。此前的生产要求继续适用：来源须为 HTTPS，
   默认 `SNAPSHOT_READ_POLICY=enforce`。
3. v1.6.3 二进制使用相同的存储状态。v1.6.3 会忽略 `REDIS_USERNAME` 和
   `REDIS_TLS_*`，以默认用户、不使用 TLS 连接；回滚前请确认 Redis 服务端仍接受
   原有连接方式。v1.6.3 同样忽略 `RATE_LIMIT`、`RATE_LIMIT_WINDOW` 以及配置文件
   中的限流值。
4. 回滚会恢复上述各项的旧行为，包括 HMAC 校验时不设上限的请求体读取和
   `/v1/lookup` 的完整响应。
5. SDK 使用方回滚时应恢复应用升级前的 `go.mod`/`go.sum` 后重建，并用
   `go list -m` 核实实际选中版本。

早于 v1.6.0 的部署仍须遵循[该版本迁移要求](../releases/v1.6.0.md)。英文说明见
[v1.7.0 release notes](../releases/v1.7.0.md)。发布后核对签名 tag、源码、Go 模块
校验和、公开资产哈希、SBOM 和 OCI digest 签名；`warden --version` 可输出二进制的
版本、提交和构建时间。
