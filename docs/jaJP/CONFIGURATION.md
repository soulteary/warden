# 設定

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

本ドキュメントでは、動作モード、設定ファイルの形式、環境変数など、Warden の設定オプションについて詳しく説明します。

**設定の優先順位**: コマンドライン引数 > 環境変数 > 設定ファイル（YAML）> 既定値。

**オプションの完全な一覧表**（YAML パス、環境変数、既定値、検証ルール）は [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md) を参照してください。概要は次のとおりです。

| カテゴリ | YAML / Env | 備考 |
|----------|------------|--------|
| サーバー | `server.*` / `PORT` | port、read_timeout、write_timeout、shutdown_timeout、idle_timeout、max_header_bytes |
| Redis | `redis.*` / `REDIS`、`REDIS_PASSWORD`、`REDIS_PASSWORD_FILE`、`REDIS_ENABLED` | addr、password、password_file、db。Redis は既定で有効（`true`）。ただし REDIS 未設定の ONLY_LOCAL を除く |
| キャッシュ | `cache.ttl`、`cache.update_interval` | 環境変数による上書きなし。update_interval の既定値は 5s |
| レート制限 | `rate_limit.rate`、`rate_limit.window` | 既定は 60/分、ウィンドウ 1m |
| HTTP クライアント | `http.*` / `HTTP_TIMEOUT`、`HTTP_MAX_IDLE_CONNS`、`HTTP_INSECURE_TLS` | timeout、max_idle_conns、insecure_tls、max_retries、retry_delay |
| リモート | `remote.*` / `CONFIG`、`KEY`、`MERGE_MODE`、`REMOTE_DECRYPT_ENABLED`、`REMOTE_RSA_PRIVATE_KEY_FILE`、`REMOTE_RSA_PRIVATE_KEY` | url、key、mode、decrypt_enabled、rsa_private_key_file |
| タスク | `task.interval` | 設定ファイル使用時は環境変数で上書きできません。`INTERVAL` は設定ファイルを使わない場合のみ有効 |
| アプリ | `app.*` / `API_KEY`、`DATA_FILE`、`DATA_DIR`、`RESPONSE_FIELDS` | mode、api_key、data_file、data_dir、response_fields |
| トレーシング | `tracing.enabled`、`tracing.endpoint` / `OTLP_ENABLED`、`OTLP_ENDPOINT` | `--config-file` 使用時、`CONFIG_FILE` が同じパスに設定されていない限り、そのファイルから tracing は読み込まれません |
| サービス間認証 | — / `WARDEN_HMAC_KEYS`、`WARDEN_HMAC_TIMESTAMP_TOLERANCE`、`WARDEN_TLS_*` | **環境変数のみ**（YAML キーなし） |
| ヘルス | — / `SNAPSHOT_MAX_AGE` | 許容されるスナップショットの最大経過時間。Go の duration 形式、既定値は `max(30s, タスク間隔 × 3)` |

## 動作モード（MERGE_MODE）

システムは 7 種類のデータマージモードに対応しており、`MERGE_MODE` で選択します（`MODE` は非推奨）。

| モード | 説明 | 利用シーン |
|------|-------------|----------|
| `DEFAULT` | 従来のリモート優先かつ寛容な動作 | モードを明示的に選択していなかった既存デプロイとの後方互換 |
| `REMOTE_FIRST` | 読み込みが成功した場合はリモートを優先。リモートエラーは致命的で、直前の有効なスナップショットを保持 | リモートが権威となる厳格なデプロイ |
| `ONLY_REMOTE` | リモートデータソースのみを使用 | リモート設定に完全に依存 |
| `ONLY_LOCAL` | ローカル設定ファイルのみを使用し、**Redis は既定で無効**（`REDIS` アドレスを明示的に設定するか `REDIS_ENABLED=true` にすると有効化） | オフライン環境またはテスト環境 |
| `LOCAL_FIRST` | ローカル優先。ローカルデータが存在しない場合にリモートデータで補完 | ローカル設定を主、リモートを従とする構成 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | リモート優先。リモート失敗時はローカルへのフォールバックを許可 | 高可用性シナリオ |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | ローカル優先。ローカル失敗時はリモートへのフォールバックを許可 | ハイブリッドモード |

### スナップショットの鮮度とリモート障害

`REMOTE_FIRST` と `ONLY_REMOTE` は厳格なモードです。定期的なリモート更新が失敗した場合、
Warden は直前の有効なメモリ内スナップショットを提供し続け、更新失敗を記録し、
スナップショットの `loaded_at` を進めません。スナップショットが `SNAPSHOT_MAX_AGE`
（既定値 `max(30s, タスク間隔 × 3)`）を超えると、ヘルスエンドポイントは HTTP 503 を
返します。値は `2m` のような Go の duration 形式で設定します。

`REMOTE_FIRST_ALLOW_REMOTE_FAILED` は検証済みのローカルデータへ明示的にフォールバック
し、スナップショットを `degraded` としてマークしつつ HTTP 200 で稼働を継続します。
`LOCAL_FIRST` と `LOCAL_FIRST_ALLOW_REMOTE_FAILED` は、リモートによる補完が利用できなく
ても、主となるローカルデータの読み込みが成功していれば正常状態を維持できます。
`DEFAULT` は互換性のために従来の寛容な動作（上記の平文ローカル成功時の挙動を含む）を
保持しています。新しい本番デプロイでは明示的にモードを選択してください。暗号化された
リモートの失敗によりローカルへフォールバックした場合は、いずれの寛容モードでも
`degraded` として報告されます。

複数レプリカ構成では、各レプリカがプロセスローカルのキャッシュとスナップショットを
それぞれ更新します。分散 Redis ロックは共有 Redis キャッシュの書き込み担当を選出する
だけなので、書き込み担当でないレプリカも自身のスナップショットの鮮度を進めます。

### 設定方法

動作モードは次の方法で設定できます。

**コマンドライン引数**:
```bash
go run . --mode DEFAULT
```

**環境変数**:
```bash
export MERGE_MODE=DEFAULT
# MODE は非推奨の互換エイリアスとして残っています。
```

**設定ファイル**:
```yaml
remote:
  mode: "DEFAULT"
# または
app:
  mode: "DEFAULT"
```

## 設定ファイルの形式

### ローカルユーザーデータファイル（`data.json`）

ローカルユーザーデータファイル `data.json` の形式（`data.example.json` を参照）:

**最小構成**（必須フィールドのみ）:
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**完全な構成**（すべての任意フィールドを含む）:
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**フィールドの説明**:
- `phone`（必須）: ユーザーの電話番号
- `mail`（必須）: ユーザーのメールアドレス
- `user_id`（任意）: ユーザーの一意な識別子。指定がない場合は `phone` または `mail` から自動生成されます
- `status`（任意）: ユーザーの状態。値が省略された場合は安全側に倒して `"inactive"` となります。アクセスを許可するには `"active"` を明示的に設定してください。
- `scope`（任意）: ユーザーの権限スコープの配列。既定値は空の配列
- `role`（任意）: ユーザーのロール。既定値は空文字列

### アプリケーション設定ファイル（`config.yaml`）

YAML 形式の設定ファイルに対応しており、`--config-file` パラメーターで指定します。

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1MB
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # 環境変数 REDIS_PASSWORD または REDIS_PASSWORD_FILE の利用を推奨
  password_file: ""  # パスワードファイルのパス（password より優先）
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # 1 分あたりのリクエスト数
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # 開発環境専用
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # リモートレスポンスを RSA で復号（rsa_private_key_file または REMOTE_RSA_PRIVATE_KEY と併用）
  rsa_private_key_file: ""    # PEM ファイルのパス（インライン PEM には環境変数 REMOTE_RSA_PRIVATE_KEY を使用）

task:
  interval: 5s

app:
  mode: "DEFAULT"  # データマージモード。本番向けのポリシーは ENVIRONMENT で選択します
  api_key: ""      # 環境変数 API_KEY の利用を推奨
  data_file: "./data.json"
  data_dir: ""     # 任意: ディレクトリ内のすべての *.json をマージ（data_file と併用可）
  response_fields: []  # 任意: API レスポンスのフィールドホワイトリスト。空ならすべてのフィールド

tracing:
  enabled: false
  endpoint: ""     # 例: "http://localhost:4318"
```

**設定の優先順位**: コマンドライン引数 > 環境変数 > 設定ファイル > 既定値。

**トレーシングに関する注意**: `--config-file` を使用する場合、環境変数 `CONFIG_FILE` が同じパスに設定されているか、`OTLP_ENABLED` + `OTLP_ENDPOINT` を使用しない限り、メインプログラムはそのファイルの `tracing` セクションを読み込みません。

サンプルファイルを参照してください: [config.example.yaml](../../config.example.yaml)。

## コマンドライン引数

```bash
go run . \
  --port 8081 \                    # Web サービスのポート（既定値: 8081）
  --redis localhost:6379 \         # Redis アドレス（既定値: localhost:6379）
  --redis-password "password" \    # Redis パスワード（任意。環境変数の利用を推奨）
  --redis-enabled=true \           # Redis の有効化／無効化（既定値: true）
  --config http://example.com/api \ # リモート設定の URL
  --key "Bearer token" \           # リモート設定の認証ヘッダー
  --interval 5 \                   # 定期タスクの間隔（秒。既定値: 5）
  --mode DEFAULT \                 # 動作モード（上記の説明を参照）
  --http-timeout 5 \               # HTTP リクエストのタイムアウト（秒。既定値: 5）
  --http-max-idle-conns 100 \     # HTTP の最大アイドル接続数（既定値: 100）
  --http-insecure-tls \           # TLS 証明書の検証をスキップ（開発環境専用）
  --api-key "your-secret-api-key" \ # 認証用の API キー（任意。環境変数の利用を推奨）
  --config-file config.yaml        # 設定ファイルのパス（YAML 形式に対応）
```

**注意**:
- 設定ファイルのサポート: `--config-file` パラメーターで YAML 形式の設定ファイルを指定できます
- Redis パスワードのセキュリティ: コマンドライン引数ではなく環境変数 `REDIS_PASSWORD` または `REDIS_PASSWORD_FILE` の利用を推奨します
- TLS 証明書の検証: `--http-insecure-tls` は開発環境専用であり、本番環境では使用しないでください

## 環境変数

環境変数による設定に対応しており、コマンドライン引数よりも優先度は低くなります。オプションの完全な一覧表（検証ルールを含む）は [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md) を参照してください。

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis パスワード（任意）
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis パスワードファイルのパス（任意。優先順位: REDIS_PASSWORD > REDIS_PASSWORD_FILE > 設定ファイル）
export REDIS_ENABLED=true               # Redis の有効化／無効化（任意。既定値: true。true/false/1/0 に対応）
                                        # 注意: ONLY_LOCAL モードでは既定値は false
                                        #       ただし REDIS アドレスを明示的に設定した場合は自動的に有効化されます
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # ローカルユーザーデータファイルのパス
export DATA_DIR=                      # 任意: すべての *.json をマージするディレクトリ（DATA_FILE と併用可）
export RESPONSE_FIELDS=               # 任意: API レスポンスのフィールドホワイトリスト（カンマ区切り。例: phone,mail,user_id,status,name）。空ならすべて
export REMOTE_DECRYPT_ENABLED=false   # 任意: リモートレスポンスを RSA で復号
export REMOTE_RSA_PRIVATE_KEY_FILE=   # 任意: RSA 秘密鍵 PEM のパス（インライン PEM には REMOTE_RSA_PRIVATE_KEY を使用）
export REMOTE_RSA_PRIVATE_KEY=        # 任意: インラインの RSA 秘密鍵 PEM（REMOTE_RSA_PRIVATE_KEY_FILE 未設定時に使用）
export HTTP_TIMEOUT=5                  # HTTP リクエストのタイムアウト（秒）
export HTTP_MAX_IDLE_CONNS=100         # HTTP の最大アイドル接続数
export HTTP_INSECURE_TLS=false         # TLS 証明書の検証をスキップするか（true/false または 1/0）
export API_KEY="your-secret-api-key"   # 認証用の API キー（強く推奨）
export CONFIG_FILE=config.yaml         # 任意。`--config-file` を使わない場合に YAML からトレーシングを読み込む、または `--config-file` と同じファイルからトレーシングを有効化する用途
export OTLP_ENABLED=false              # OpenTelemetry を有効化（true/false または 1/0）
export OTLP_ENDPOINT=http://localhost:4318  # OTLP エンドポイント（OTLP_ENABLED が true の場合は必須）
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # 信頼するプロキシ IP のリスト（カンマ区切り）
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # ヘルスチェックエンドポイントの IP 許可リスト（任意）
export IP_WHITELIST="192.168.1.0/24"  # グローバル IP 許可リスト（任意）
export LOG_LEVEL="info"                # ログレベル（任意。既定値: info。選択肢: trace, debug, info, warn, error, fatal, panic）
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # 本番環境のシークレットは 32 バイト以上が必要
export WARDEN_HMAC_ALLOW_V1=false                # 既定値: false。期間を限定した v1 移行時のみ true に設定
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # HMAC タイムスタンプの許容誤差（秒）
export WARDEN_TLS_CERT=/path/to/warden.crt    # サービス間認証: サーバー TLS 証明書（KEY と併せて TLS を有効化）
export WARDEN_TLS_KEY=/path/to/warden.key     # サーバー TLS 秘密鍵
export WARDEN_TLS_CA=/path/to/ca.crt          # クライアント CA（mTLS）
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # クライアント証明書を必須にする（mTLS）
```

**環境変数の優先順位**:
- Redis パスワード: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > コマンドライン引数 `--redis-password`

**セキュリティ設定に関する注意**:
- `API_KEY`: 機微なエンドポイント（`/`、`/log/level`）を保護します。本番環境では強く推奨されます
- `TRUSTED_PROXY_IPS`: 信頼するリバースプロキシの IP を設定し、クライアントの実 IP を正しく取得します
- `HEALTH_CHECK_IP_WHITELIST`: ヘルスチェックエンドポイントへのアクセス元 IP を制限します（任意。CIDR 範囲に対応）
- `IP_WHITELIST`: グローバル IP 許可リスト（任意。CIDR 範囲に対応）

## リモート設定 API の要件

リモート設定 API は同じ形式の JSON 配列を返す必要があり、任意で Authorization ヘッダーによる認証に対応できます。

API のレスポンス形式は `data.json` ファイルの形式と一致させてください。

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

環境変数 `KEY` または `--key` パラメーターが設定されている場合、リクエストに `Authorization` ヘッダーが自動的に追加されます。

```http
Authorization: Bearer your-token-here
```

## 任意のサービス連携設定

他のサービス（Stargate など）と連携する場合、サービス間認証を設定できます。関連する設定項目は次のとおりです。

**注意**: Warden を単独で利用する場合、以下の設定は任意です。

### mTLS の設定（推奨）

相互 TLS 証明書をサービス間認証に使用します。**環境変数のみ対応**（アプリケーション設定に YAML キーはありません）。

```bash
# Warden のサーバー証明書
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# クライアント証明書を必須にする（mTLS）
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### HMAC 署名の設定

HMAC-SHA256 署名をサービス間認証に使用します。**環境変数のみ対応**（YAML キーはありません）。

```bash
# HMAC キー（JSON 形式。ローテーション用に複数キーに対応）
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# タイムスタンプの許容誤差（秒）。HMAC キーが設定されている場合の既定値は 60
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# 旧来の v1 は既定で無効です。古い呼び出し元の移行中のみ有効にしてください。
export WARDEN_HMAC_ALLOW_V1=false
```

### Stargate からの呼び出し設定

Stargate 側では Warden のサービスアドレスと認証情報を設定する必要があります。

**Stargate の設定例**（環境変数）:
```bash
# Warden のサービスアドレス
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# サービス間認証の方式（mTLS または HMAC）
export STARGATE_WARDEN_AUTH_TYPE=hmac

# HMAC の設定（HMAC を使う場合）
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# mTLS の設定（mTLS を使う場合）
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### 設定の優先順位

1. **mTLS**: TLS 証明書が設定されている場合は mTLS が優先されます
2. **HMAC**: mTLS が未設定の場合は HMAC 署名が使用されます
3. **API キー**: いずれも未設定の場合は API キー認証にフォールバックします（サービス間呼び出しには推奨されません）

### 設定の検証

Warden は起動時にサービス間認証の設定を検査します。

- 不完全な TLS 設定を拒否します。証明書と秘密鍵は必ず両方設定する必要があり、mTLS ではさらにクライアント CA が必要です
- HMAC が設定されている場合、キーの形式が正しいか検証します
- `ENVIRONMENT=production` では、API キー、HMAC、mTLS のいずれの認証も設定されていない場合、起動を拒否します

## 詳細な設定ドキュメント

パラメーター解析の仕組み、優先順位のルール、利用例の詳細については以下を参照してください。

- [パラメーター解析の設計ドキュメント](CONFIG_PARSING.md) - パラメーター解析の仕組みに関する詳細な説明
- [アーキテクチャ設計ドキュメント](ARCHITECTURE.md) - 全体のアーキテクチャと設定の影響を理解する
- [セキュリティドキュメント](SECURITY.md) - サービス間認証の詳細
