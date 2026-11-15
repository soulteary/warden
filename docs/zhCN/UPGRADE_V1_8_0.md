# Warden v1.8.0 升级与回滚

本版本对环境变量做严格校验，限流按认证结果拆分，审计记录写入应用日志，新增 HMAC
签名版本 v3，并修正多副本向 Redis 发布规则集的行为。部分在 v1.7.0 下可以启动的配置
现在会被拒绝，若干响应、日志和指标也有变化，生产升级前请逐项核对。

HTTP 路径和成功响应的 JSON 结构不变，唯一例外是下文提到的空列表。v2 Redis 快照格式
不变；legacy 模式新增一个键。不需要转换数据。依赖、Go 版本（1.27.4）和构建镜像与
v1.7.0 相同。Go SDK 新增方法和选项，另有三处行为变化，见“SDK 使用方”。

## 新增配置

- `RATE_LIMIT_AUTHENTICATED` 与 `rate_limit.authenticated_rate`（默认 600）：单个凭据
  在每个 `RATE_LIMIT_WINDOW` 内允许的请求数。环境变量优先于配置文件；小于 1 或不是
  整数会终止启动。
- `TRUSTED_PROXY_IPS=none`（不区分大小写，且必须是唯一取值）：不信任任何对端，忽略
  `X-Forwarded-For` 和 `X-Real-IP`，客户端地址取 TCP 对端地址。`none` 与其他条目
  同时出现会终止启动。
- `WARDEN_HMAC_AUDIENCES`：v3 签名中可接受的本实例名称，逗号分隔，最多 16 个。名称
  为 1 到 255 个可打印 ASCII 字符，不含空格和逗号，按小写比较。默认为空，此时拒绝
  v3 签名。
- `WARDEN_HMAC_REQUIRE_AUDIENCE`（默认 `false`）：设为 true 后拒绝 v1、v2 和未带版本
  的签名。它要求同时设置 `WARDEN_HMAC_AUDIENCES`，否则终止启动。

## 升级前核对

### 现在会拒绝启动的情况

升级前请检查每个副本的环境变量。以下任一情况都会使进程退出，错误信息会指明对应的
配置项。

- 布尔型环境变量的值不是 `true`/`false`、`1`/`0`、`yes`/`no`、`on`/`off`、`t`/`f`
  之一（不区分大小写）。涉及的变量：`REDIS_ENABLED`、`REDIS_TLS_ENABLED`、
  `HTTP_INSECURE_TLS`、`REMOTE_DECRYPT_ENABLED`、`REMOTE_ENCRYPTION_REQUIRED`、
  `REQUIRE_EXPLICIT_USER_ID`、`WARDEN_TLS_REQUIRE_CLIENT_CERT`、
  `WARDEN_HMAC_ALLOW_V1`、`WARDEN_HMAC_REQUIRE_AUDIENCE`、
  `WARDEN_METRICS_REQUIRE_AUTH`、`OTLP_ENABLED`。空值视为未设置。v1.7.0 对其中多数
  变量把无法识别的值当作 false；`WARDEN_METRICS_REQUIRE_AUTH` 则记录警告并回退到
  环境默认值。
- 同一改动也会让原本未生效的开关生效：`yes`、`on`、`t` 在 v1.7.0 中被读作 false，
  现在读作 true，例如 `REDIS_ENABLED=yes`、`REMOTE_DECRYPT_ENABLED=on`。请确认这些
  取值符合部署的本意。
- `PORT` 不是 1 到 65535 的整数；`INTERVAL` 或 `WARDEN_HMAC_TIMESTAMP_TOLERANCE`
  不是不小于 1 的整数；`HTTP_MAX_IDLE_CONNS` 不是不小于 0 的整数。v1.7.0 会不加提示
  地改用默认值，`PORT=abc` 因而监听 8081。`PORT=0`、`INTERVAL=0`、
  `WARDEN_HMAC_TIMESTAMP_TOLERANCE=0` 此前表示使用默认值，现在被拒绝；要使用默认值，
  请删除该变量。
- `HTTP_TIMEOUT` 既不是整数秒，也不是 `30s` 这样的时长，或者超出 1 秒到 1 小时的
  范围。`HTTP_TIMEOUT=0` 此前表示默认的 5 秒，现在被拒绝。3600 秒的上限同样适用于
  `--http-timeout` 和 `http.timeout`。
- `CONFIG`（或 `--config`、`remote.url`）设为 `http://localhost:8080/data.json`。
  该地址曾是内置默认值，并且不经过校验；现在它与其他地址一样接受校验并被拒绝，因为
  回环地址不能作为远程数据源。v1.7.0 实际上也无法连接它。请删除该配置，或改为真实
  数据源的地址。
- `MERGE_MODE` 为 `ONLY_REMOTE` 或 `REMOTE_FIRST`，但没有配置远程地址。`CONFIG`
  不再有默认值。v1.7.0 在这种情况下可以启动，但每次加载都失败。
- `ENVIRONMENT=production`，设置了 `IP_WHITELIST`，而 `TRUSTED_PROXY_IPS` 为空。
  请将 `TRUSTED_PROXY_IPS` 设为反向代理地址；客户端直连 Warden 时设为 `none`。
  `HEALTH_CHECK_IP_WHITELIST` 不在此规则之内。
- `RESPONSE_FIELDS` 或 `app.response_fields` 中出现 `phone`、`mail`、`user_id`、
  `status`、`scope`、`role`、`name`、`dingtalk_userid` 以外的名称。名称区分大小写，
  `email`、`Mail` 都会被拒绝。v1.7.0 接受这类条目，结果是对应字段从所有响应中消失。
- `RATE_LIMIT_AUTHENTICATED`、`WARDEN_HMAC_AUDIENCES`、
  `WARDEN_HMAC_REQUIRE_AUDIENCE` 的值无效。这三个变量是新增的，只有环境中已经定义
  了它们时才需要检查。

### 远程数据源

- 未设置 `CONFIG` 即表示没有远程数据源。此时 `DEFAULT`、`LOCAL_FIRST` 和两个
  `*_ALLOW_REMOTE_FAILED` 模式只加载本地数据，不发起远程请求，也不会被标记为降级。
  v1.7.0 下这类部署每次加载都会请求内置地址并记录失败。
- 后台刷新的期限为 `HTTP_TIMEOUT` 加 7 秒重试退避，最短 10 秒。此前固定为 10 秒，
  远程数据源即使在更长的 `HTTP_TIMEOUT` 之内返回，刷新也会被取消。
- `HTTP_MAX_IDLE_CONNS` 现在真正作用于远程数据源客户端。此前该值被忽略，客户端始终
  使用 20；20 现在是文档中的默认值。设置了该变量的部署会按所设的值运行。
- 远程主机名解析到连接策略拒绝的地址时，加载立即失败。
  `warden_refresh_failures_total` 以新的 `reason` 取值 `address_policy` 计数；此前会
  一直重试到请求超时，并计为超时。按 reason 逐项列举的告警和看板需要补上这个取值。

### 限流与错误响应

- 限流按认证结果拆分。通过认证的请求按凭据计数，上限为 `RATE_LIMIT_AUTHENTICATED`
  （默认每个窗口 600 次），不再占用按 IP 计算的配额。一个凭据指一个 HMAC key ID、
  一张 mTLS 客户端证书，或者 API key 加客户端地址。认证失败的请求、匿名访问
  `/metrics` 的请求和未知路径仍按客户端 IP 计数，上限为 `RATE_LIMIT`（默认 60），
  与此前一致。
- 默认配置下的效果：从单一地址访问 Warden 的网关，由原来该地址每分钟 60 次，变为
  每个凭据每分钟 600 次。使用共享 `API_KEY` 的调用方，每个客户端地址每分钟 600 次。
  一个 HMAC key ID 或一张客户端证书，不论从多少个地址使用，共用一份配额。此前靠
  调低 `RATE_LIMIT` 来约束已认证调用方的部署，需要设置 `RATE_LIMIT_AUTHENTICATED`
  才能保持原有约束。
- 某个地址的按 IP 配额被认证失败的请求耗尽后，该地址的请求在校验凭据之前就会收到
  429，携带有效凭据的请求也不例外。
- 所有 429 响应都带 `Retry-After`：距离限流器再次放行的秒数，向上取整。v1.7.0 不带
  该响应头。
- 生产环境下，由中间件写出的错误响应（401、413、429 等）以
  `Content-Type: application/json` 发送。响应体此前已经是 JSON 对象，但标为
  `text/plain`。其他环境不变。
- `warden_rate_limit_hits_total` 以 `scope="credential"` 统计按凭据的限流；
  `scope="ip"` 含义不变。
- 设置了 `RESPONSE_FIELDS` 时，没有用户的列表响应为 `[]`，此前为 `null`。

### 日志与链路追踪

- 审计记录写入应用日志。每次用户查询（`/user`、`/v1/user`、`/v1/lookup`）以及每次
  对 `/log/level` 的 `POST` 或 `PUT`，都会产生一行带 `"audit": true` 的 JSON。此前
  这些记录被丢弃。该行固定以 `info` 级别写出，不受 `LOG_LEVEL` 以及通过
  `/log/level` 设置的级别影响。每次查询会多出一行日志，请据此评估日志存储容量。
  记录中的手机号、邮箱和 user ID 均已脱敏。
- 应用日志中的 `user_id`，以及链路追踪属性 `warden.user.id`、
  `warden.query.user_id`，现在只保留前 4 个字符，其余替换为 `*`。按完整 user ID
  检索日志或链路的查询将不再命中。
- 每行日志的 `version` 字段现在是构建版本，此前固定为 `0.7.0`。依赖该字段的日志
  过滤规则需要更新。
- 生产环境下，“详细信息已隐藏”这一日志行对 4xx 响应改为 `info` 级别，5xx 仍为
  `error`。按这类日志计算错误率的告警，命中数量会减少。
- 首次加载的日志消息不再提及数据来源，来源由 `source` 字段给出。消息原为
  `从远程 API 加载数据 ✓`（从本地文件加载时也是这句），现为
  `从已配置的数据源加载数据 ✓`。按旧文本匹配的日志解析规则，需要改用新文本或该
  字段。
- 新增若干启动警告：`TRUSTED_PROXY_IPS` 为空（所有环境都会记录，取代原先仅在设置
  `IP_WHITELIST` 时才出现的警告）；非生产环境未配置任何服务凭据；`INTERVAL` 小于
  5 秒；配置文件中存在会被解析但不生效的键；`RESPONSE_FIELDS` 未包含 `status`；
  Redis 数据库中存在非 Warden 写入的键；生产环境使用未设密码的 Redis；某个
  `user_id` 与另一用户的手机号或邮箱相同。这些警告都不会终止启动。
- `TRUSTED_PROXY_IPS` 为空时的行为在本版本中不变：信任来自任意私网或回环对端的转发
  地址。后续版本将在该变量为空时不再信任转发头，建议现在就把它设为代理地址或
  `none`。

### 健康检查

- 并发的健康检查请求共用一次 Redis 探测。健康结果复用 3 秒，失败结果复用 1 秒。
  Redis 故障最多延迟 3 秒才反映到健康检查响应中，恢复最多延迟 1 秒。
- 强制读取（`SNAPSHOT_READ_POLICY=enforce`）接受来源时间比本地时钟超前不超过 5 秒
  的 v2 快照。此前这样的副本会返回 503，直到本地时钟追上为止。

### Redis

- legacy 与 shadow 模式（`REDIS_SNAPSHOT_MODE=legacy` 为默认值）新增键
  `warden:users:cache:source`，与 `warden:users:cache`、
  `warden:users:cache:version` 在同一个事务中写入。逐个列出键名的 Redis ACL 必须在
  升级前加入该键，否则事务被拒绝，共享规则集不再写入；各副本继续使用内存中的数据
  提供服务。
- 在这两种模式下，写入者会先执行一个脚本（`EVALSHA`，脚本未缓存时为 `EVAL`），
  脚本内调用 `GET`、`PEXPIRE`、`PERSIST` 和 `SET`。逐个列出命令的 ACL，除 v1.7.0
  已用到的命令外，还应放行 `EVALSHA` 和 `PEXPIRE`。脚本被拒绝时，加载成功得到的
  规则集仍会在每次刷新时整体写入，而副本保留下来的规则集不会写入。Redis 中已经是
  当前规则集时，只续期 TTL：列表不再重写，`warden:users:cache:version` 也不再随
  每次刷新递增。
- 启动时 Warden 会执行一次 `SCAN`，检查是否存在非自身写入的键。无权执行 `SCAN`
  的用户会在 `ACL LOG` 中留下一条记录；检查被跳过，启动继续。
- v2 与 shadow 模式：快照续期时只用 `GETRANGE` 读取已存值的开头。该命令是可选的。
  ACL 不允许时，脚本按原方式读取整个值，但每次续期都会在 `ACL LOG` 和
  `acl_access_denied_cmd` 计数器中增加一条被拒记录；授予 `+getrange` 即可避免。
  其余命令与 v1.7.0 相同，见
  [快照存储调用的 Redis 命令](REDIS_SNAPSHOT_PROTOCOL.md)。
- Redis 写入锁（`warden:rules:refresh`，以及 v2 的 lease）的有效期：`HTTP_TIMEOUT`
  不超过 23 秒时为 1 分钟，超过时为 `HTTP_TIMEOUT` 加 37 秒。此前一律为 1 分钟。
  副本在刷新过程中停止时，锁会保留到过期为止；在此期间其他副本照常刷新自身数据，
  但跳过对 Redis 的写入。`HTTP_TIMEOUT` 不要设得比远程数据源实际需要的更大。
- 刷新结果全部被拒绝、因而保留上一份有效规则集的副本，不再用它覆盖来源加载时间
  更新的规则集或快照。三种模式均适用，前提是各副本时钟一致。
- v2 与 shadow 模式：快照键缺失，或其中的值不是本数据集、本策略的发布结果时，lease
  持有者会在下一个周期重新发布。因此，快照策略不同却共用同一个
  `REDIS_SNAPSHOT_NAMESPACE` 的副本，会在每个周期互相替换对方的快照。请按
  [迁移指南](REDIS_SNAPSHOT_MIGRATION.md)的要求，在升级前为它们分配不同的命名空间。

### HMAC 签名

- v2 默认继续被接受，现有客户端无需改动。v3 把签名绑定到 `WARDEN_HMAC_AUDIENCES`
  中列出的名称。未设置该变量的实例对 v3 请求返回 401，v1.7.0 也是如此。
- 顺序：先升级全部副本，再在每个副本上设置 `WARDEN_HMAC_AUDIENCES`，然后把客户端
  切换到 v3（Go SDK 使用 `WithHMACAudience`）；所有客户端都使用 v3 签名之后，才设置
  `WARDEN_HMAC_REQUIRE_AUDIENCE=true`。

### SDK 使用方

- `BaseURL` 不带协议时，`NewClient` 返回 `INVALID_CONFIG`。此前会自动补上
  `http://`。只写了 `host:port` 的配置需要加上 `http://` 或 `https://`。
- 429 响应返回错误码 `RATE_LIMITED`，并带有 `Error.RetryAfter`；此前返回
  `REQUEST_FAILED`。SDK 不会对 429 重试。比较错误码的代码需要相应调整。
- 使用 `WithTLSConfig` 或 `WithTLSClientCert` 且未提供 transport 的客户端，现在
  保留默认 transport 的设置：环境变量中的代理配置和默认连接超时都会生效。此前使用
  的 transport 不含这些设置。不希望走环境代理的客户端需要自行提供 transport。
  `NewClient` 不再修改调用方传入的 transport。
- 新增：`CheckUserInListWithError`，可以区分“用户被拒绝”和“无法向 Warden 查询”，
  并在用户响应缺少 `status` 字段时返回 `INVALID_RESPONSE`；`Lookup`，对应
  `GET /v1/lookup`；`WithRequireHTTPS`，拒绝通过明文 HTTP 向本机以外的主机发送
  API key（默认关闭，未开启时客户端会向所配置的 `Logger` 输出警告）；以及
  `WithHMACAudience`。`CheckUserInList` 的签名和返回结果不变。

根目录的 Compose 文件不再挂载 `/etc/timezone`，其健康检查端口跟随 `PORT`。

## 升级与回滚

多副本部署的升级顺序：

1. 替换第一个副本之前，先修正“现在会拒绝启动的情况”中列出的配置，并扩充 Redis
   ACL（键 `warden:users:cache:source`；v2 模式加上 `+getrange`）。这两项改动与
   v1.7.0 兼容。
2. 逐个替换副本，顺序不限。在确定不再回滚到 v1.7.0 之前，不要使用新增的配置
   （`TRUSTED_PROXY_IPS=none`、`WARDEN_HMAC_AUDIENCES`、
   `WARDEN_HMAC_REQUIRE_AUDIENCE`，以及布尔值写法 `yes`、`on`、`t`），原因见下文。
3. 服务升级完成后再升级 SDK 使用方，执行
   `go get github.com/soulteary/warden@v1.8.0` 并重建；`WithHMACAudience` 要等第 4
   步完成后才能启用。v1.8.0 的 SDK 只要仍使用 v2 签名，就可以访问 v1.7.0 的服务；
   由于 v1.7.0 不发送 `Retry-After`，此时 `Error.RetryAfter` 为零。
4. 全部副本运行 v1.8.0 之后，设置 `WARDEN_HMAC_AUDIENCES`，再把客户端切换到 v3。

两个版本并存时的行为：

- legacy 与 shadow 模式：v1.7.0 副本照旧读写 `warden:users:cache`，忽略
  `warden:users:cache:source`。它写入的规则集没有来源记录，v1.8.0 副本会像 v1.7.0
  那样将其替换。v1.7.0 副本仍会用保留下来的旧规则集覆盖较新的规则集。只有全部副本
  都运行 v1.8.0，这项保护才完整。
- v2 模式：快照格式和 lease 不变，v1.7.0 副本可以接受 v1.8.0 副本发布的全部内容。
  持有 lease 的 v1.7.0 副本保持 v1.7.0 的行为：可能用保留的规则集覆盖较新的快照，
  键缺失时也不会重新发布。
- 限流在各进程内分别计数。v1.7.0 副本仍把所有请求按客户端 IP 计入 `RATE_LIMIT`。
- 两个版本的写入锁使用同一个键。v1.8.0 副本取得的锁有效期可能超过 1 分钟，v1.7.0
  副本同样遵守这把锁。

回滚到 v1.7.0：

1. v1.8.0 不会写入 v1.7.0 无法读取的状态。v1.7.0 忽略
   `warden:users:cache:source`（该键在 1 小时内过期），并照旧覆盖 legacy 规则集。
   v2 快照无需处理。
2. 先替换 `TRUSTED_PROXY_IPS=none`：v1.7.0 遇到该值会拒绝启动。请改为代理地址，
   或删除该变量；删除后 v1.7.0 信任来自任意私网或回环对端的转发头。
3. 布尔变量请写成 `true`/`false` 或 `1`/`0`。v1.7.0 对多数变量会不加提示地把
   `yes`、`on`、`t` 读作 false，对 `REDIS_TLS_ENABLED` 和 `WARDEN_HMAC_ALLOW_V1`
   则会因这些值拒绝启动。
4. 回滚服务之前，先让客户端改回 v2 签名：v1.7.0 对 v3 返回 401。v1.7.0 忽略
   `WARDEN_HMAC_AUDIENCES` 和 `WARDEN_HMAC_REQUIRE_AUDIENCE`，会重新接受 v2。
5. v1.7.0 忽略 `RATE_LIMIT_AUTHENTICATED` 和 `rate_limit.authenticated_rate`，所有
   请求都按客户端 IP 计入 `RATE_LIMIT`。从单一地址发出、每个窗口超过 `RATE_LIMIT`
   次请求的网关会再次收到 429。
6. 未设置 `CONFIG` 时，v1.7.0 重新使用内置地址 `http://localhost:8080/data.json`，
   每次刷新都会记录一次远程加载失败。
7. 审计记录不再写出，`user_id` 在日志和链路追踪中恢复为明文，日志的 `version`
   字段回到 `0.7.0`。
8. SDK 使用方回滚时，恢复应用升级前的 `go.mod`/`go.sum`，移除对新增方法和选项的
   调用后重建，并用 `go list -m` 核实实际选中的版本。

从早于 v1.7.0 的版本升级时，还须遵循 [v1.7.0 升级说明](UPGRADE_V1_7_0.md)和
[v1.6.0 迁移要求](../releases/v1.6.0.md)。英文说明见
[v1.8.0 release notes](../releases/v1.8.0.md)。发布后核对签名 tag、源码、Go 模块
校验和、公开资产哈希、SBOM 和 OCI digest 签名；`warden --version` 可输出二进制的
版本、提交和构建时间，日志的 `version` 字段现在与之一致。
