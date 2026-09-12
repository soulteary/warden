# API ドキュメント

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](../itIT/API.md) | [日本語](API.md) | [Deutsch](../deDE/API.md) | [한국어](../koKR/API.md)

本ドキュメントでは、Warden が提供するすべての API エンドポイントについて詳しく説明します。

## OpenAPI ドキュメント

本プロジェクトは `openapi.yaml` ファイルに完全な OpenAPI 3.0 仕様を用意しています。

API の閲覧とテストには次のツールを利用できます。

1. **Swagger UI**: [Swagger Editor](https://editor.swagger.io/) で `openapi.yaml` ファイルを開く
2. **Postman**: `openapi.yaml` ファイルを Postman にインポートする
3. **Redoc**: Redoc を使って見やすい API ドキュメントページを生成する

## 認証

一部の API エンドポイントは API キー認証を必要とします。認証情報は次の 2 つの方法で渡せます。

1. **X-API-Key ヘッダー**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer ヘッダー**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

API キーは環境変数 `API_KEY` またはコマンドライン引数 `--api-key` で設定できます。

## ルーティング契約

以下に記載するエンドポイントが、Warden が提供するパスのすべてです。それ以外のパスは
ユーザーデータを一切含まない JSON ボディとともに `404 Not Found` を返します。

```http
GET /not-a-route
X-API-Key: your-secret-api-key
```

```json
{
  "error": "Requested resource does not exist"
}
```

> **動作の変更**: ルートパス `/` は以前サブツリーパターンとして登録されていたため、一致
> しないパス（`/foo`、`/user/`、`/v1/`）はすべてユーザー一覧ハンドラーが処理し、**許可
> リスト全体**を返していました。現在 `/` は完全一致となり、一致しないパスには上記の 404
> が返されます。任意のパスでユーザーデータが返ることに依存していたクライアントは、`/`、
> `/data.json`、`/v1/users` のいずれかを使用してください。

なお Go のルーターは一致処理の前にパスを正規化するため、`/metrics/../user` は `/user` に
解決されてそちらへリダイレクトされ、404 ハンドラーには到達しません。

## API エンドポイント

### ユーザー一覧の取得

全ユーザー、またはページ分割されたユーザー一覧を取得します。

**リクエスト**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**クエリパラメーター**:
- `page`（任意）: ページ番号。1 から始まり、既定値は 1
- `page_size`（任意）: 1 ページあたりの件数。既定ではすべてのデータ（ページ分割なし）

**注意**: このエンドポイントは API キー認証が必要です。

**レスポンス（ページ分割なし）**
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

**レスポンス（ページ分割あり）**
```json
{
    "data": [
        {
            "phone": "13800138000",
            "mail": "admin@example.com"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 100,
        "total": 200,
        "total_pages": 2
    }
}
```

**ステータスコード**: `200 OK`

**Content-Type**: `application/json`

### 単一ユーザーの取得

電話番号、メールアドレス、またはユーザー ID で単一のユーザーを照会します。

**リクエスト**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**クエリパラメーター**（ちょうど 1 つを指定する必要があります）:
- `phone`: ユーザーの電話番号
- `mail`: ユーザーのメールアドレス
- `user_id`: ユーザーの一意な識別子

**注意**:
- このエンドポイントは API キー認証が必要です
- クエリパラメーター（`phone`、`mail`、`user_id`）は 1 つのみ指定できます

**レスポンス（ユーザーが存在する場合）**
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**フィールドの説明**:
- `phone`: ユーザーの電話番号
- `mail`: ユーザーのメールアドレス
- `user_id`: ユーザーの一意な識別子（指定がない場合は自動生成）
- `status`: ユーザーの状態。取り得る値:
  - `"active"`: 有効。ユーザーはログインしてシステムを利用できます
  - `"inactive"`: 無効。ユーザーはログインできません
  - `"suspended"`: 停止中。ユーザーはログインできません
  - 未設定の場合の既定値は `"inactive"` です。ログインを許可するには `"active"` を明示的に設定する必要があります
- `scope`: ユーザーの権限スコープの配列（任意）。きめ細かい認可に使用します。例: `["read", "write", "admin"]`
- `role`: ユーザーのロール（任意）。例: `"admin"`、`"user"`、`"guest"`

**補足**:
- `status` が `"active"` のユーザーのみが認証チェックを通過します
- `scope` と `role` は Stargate が下流サービス向けの認可ヘッダー（`X-Auth-Scopes` と `X-Auth-Role`）を設定するために使用します

**任意の連携シナリオ**:
他のサービス（Stargate など）と連携する場合、ログインフローの中でこのエンドポイントを呼び出してユーザー情報を照会できます。
1. ユーザーが識別子（メール／電話番号／ユーザー名）を入力したら、`GET /user?phone=xxx` または `GET /user?mail=xxx` を呼び出す
2. Warden がユーザー情報（`user_id`、`mail`、`phone`、`status` を含む）を返す
3. ユーザーが存在し、状態が `"active"` であれば、後続の認証フローを続行できる
4. 返された `scope` と `role` は認可ヘッダーの設定に利用できる

**レスポンス（ユーザーが見つからない場合）**
- **ステータスコード**: `404 Not Found`
- **レスポンスボディ**: `User not found`

**エラーレスポンス（パラメーター不足）**
- **ステータスコード**: `400 Bad Request`
- **レスポンスボディ**: `Bad Request: missing identifier (phone, mail, or user_id)`

**エラーレスポンス（パラメーターが複数）**
- **ステータスコード**: `400 Bad Request`
- **レスポンスボディ**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### ヘルスチェック

Redis、データキャッシュ、スナップショットの出所、およびスナップショットの鮮度を確認します。

**リクエスト**
```http
GET /health
GET /healthcheck
```

**注意**: このエンドポイントは認証を必要としませんが、アクセス元 IP は環境変数 `HEALTH_CHECK_IP_WHITELIST` で制限できます。本番環境のレスポンスでは個々のチェック結果は隠されます。

**レスポンス**
```json
{
    "status": "ok",
    "service": "warden",
    "checks": {
        "redis": {
            "name": "redis",
            "status": "ok",
            "latency_ms": 1,
            "timestamp": "2026-08-31T00:00:00Z"
        },
        "snapshot": {
            "name": "snapshot",
            "status": "ok",
            "latency_ms": 0,
            "timestamp": "2026-08-31T00:00:00Z",
            "metadata": {
                "source": "merged",
                "version": "a1b2c3d4",
                "age_seconds": 2.5
            }
        }
    },
    "timestamp": "2026-08-31T00:00:00Z",
    "total_latency_ms": 1
}
```

本番環境でのレスポンス:

```json
{"status":"ok","service":"warden"}
```

**ステータスコード**:

- `200 OK`: 総合ステータスが `ok` または `degraded`。`degraded` はサービスが依然として機能していることを意味します。
- `503 Service Unavailable`: 重要なチェックが失敗しました。
- `403 Forbidden`: クライアントが `HEALTH_CHECK_IP_WHITELIST` の範囲外です。

**レスポンスフィールドの説明**:
- `status`: `ok`、`degraded`、`unhealthy` のいずれか。
- `service`: サービス名（`warden`）。
- `checks`: 開発／テスト環境でのみ返されるマップ。`redis`、`data`、`snapshot`、`snapshot_freshness` の結果を含みます。
- `checks.snapshot.metadata`: 低カーディナリティの出所・バージョン・経過時間と、安定した更新理由コード。生のリモートエラー、URL、資格情報が公開されることはありません。
- `timestamp`、`total_latency_ms`: 開発／テスト環境でのみ返される総合的な所要時間フィールド。

`REMOTE_FIRST` と `ONLY_REMOTE` では `snapshot_freshness` が重要な項目になります。出所が
不明な場合や、経過時間が `SNAPSHOT_MAX_AGE` を超えた場合は 503 を返します。寛容なモード
では、検証済みのローカルデータまたは直前の有効なスナップショットを `degraded` として
HTTP 200 で提供できます。

### ログレベルの管理

ログレベルを動的に取得・設定します。

#### 現在のログレベルを取得

**リクエスト**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**レスポンス**
```json
{
    "level": "info"
}
```

**注意**: このエンドポイントは API キー認証が必要です。

#### ログレベルを設定

**リクエスト**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**リクエストボディ**:
```json
{
    "level": "debug"
}
```

**サポートするログレベル**: `trace`、`debug`、`info`、`warn`、`error`、`fatal`、`panic`

**レスポンス**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**注意**:
- このエンドポイントは API キー認証が必要です
- ログレベルの変更操作はすべてセキュリティ監査ログに記録されます

### Prometheus メトリクス

Prometheus 形式の監視メトリクスデータを取得します。

**リクエスト**
```http
GET /metrics
```

**レスポンス**: Prometheus 形式のメトリクスデータ

**認証**: デプロイ環境によって異なります。

| `ENVIRONMENT` | `/metrics` の既定値 |
| --- | --- |
| `production` | 認証が必要（データエンドポイントと同じ方式） |
| `development`、`test`、未設定 | 匿名のスクレイピングを許可 |

`WARDEN_METRICS_REQUIRE_AUTH` は既定値を双方向に上書きします。認証が必要なエンドポイントに
未認証でスクレイピングを行うと `401 Unauthorized` が返ります。レスポンスに含まれるのは
低カーディノリティかつ非機微な系列のみです。`endpoint` と `method` のラベルは許可リストに
基づいて正規化され、認識されない値はすべて `other` にまとめられます。

**レスポンス例**:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",le="0.005"} 1000
http_request_duration_seconds_bucket{method="GET",path="/",le="0.01"} 1200
...
```

## エラーレスポンス

すべての API エンドポイントは、以下のエラーレスポンスを返す可能性があります。

### 401 Unauthorized

API キー認証に失敗した場合に返されます。

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

リクエストがレート制限を超えた場合に返されます。

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

サーバー内部エラーが発生した場合に返されます。

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

本番モードでは、情報漏えいを防ぐために詳細なエラー情報は隠されます。

## レート制限

既定では、API リクエストはレート制限によって保護されています。

- **上限**: 1 分あたり 60 リクエスト
- **ウィンドウ**: 1 分
- **超過時**: `429 Too Many Requests` を返します

レート制限は設定ファイルで調整できます。

```yaml
rate_limit:
  rate: 60  # 1 分あたりのリクエスト数
  window: 1m
```

## IP 許可リスト

IP 許可リストは以下の環境変数で設定できます。

- `IP_WHITELIST`: グローバル IP 許可リスト（すべてのエンドポイントへのアクセスを制限）
- `HEALTH_CHECK_IP_WHITELIST`: ヘルスチェックエンドポイントの IP 許可リスト（`/health` と `/healthcheck` のみを制限）

CIDR 範囲形式に対応しており、複数の IP アドレスや範囲はカンマで区切ります。

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## レスポンス圧縮

すべての API レスポンスは自動圧縮（gzip）に対応しています。クライアントはリクエストヘッダー `Accept-Encoding: gzip` で圧縮を有効にできます。

## 任意の連携例

### 他サービスとの連携における呼び出し例（任意）

他のサービス（Stargate など）と連携する必要がある場合、ログインフローの中で Warden の `/user` エンドポイントを呼び出してユーザー情報を照会できます。

**シナリオ 1: 電話番号で照会**

```bash
# Stargate が Warden を呼び出す
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?phone=13800138000"
```

**レスポンス例**:
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**シナリオ 2: メールアドレスで照会**

```bash
# Stargate が Warden を呼び出す
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?mail=admin@example.com"
```

### Go SDK による連携例

Stargate は Warden の Go SDK を使って連携できます。

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Warden クライアントを作成
    opts := warden.DefaultOptions().
        WithBaseURL("http://warden:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second)
    
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // ログインフローでユーザーを照会
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            // ユーザーが見つからないためログインを拒否
            fmt.Println("User not found in allowlist")
            return
        }
        panic(err)
    }
    
    // ユーザーの状態を確認
    if !user.IsActive() {
        // 状態が有効でないためログインを拒否
        fmt.Printf("User status is %s, cannot login\n", user.Status)
        return
    }
    
    // ユーザーが存在し状態も有効なのでログインフローを続行
    fmt.Printf("User found: %s, Status: %s, Role: %s, Scopes: %v\n",
        user.UserID, user.Status, user.Role, user.Scope)
    
    // 次の手順: Herald を呼び出して確認コードを送信
    // ...
}
```

### ログインフロー全体の例（任意の連携シナリオ）

任意の連携シナリオでは、ログインフロー全体は次のようになります。

1. **ユーザーが識別子を入力** → 認証サービスが受け取る
2. **認証サービス → Warden**: ユーザー情報を照会
   ```go
   user, err := wardenClient.GetUserByIdentifier(ctx, phone, mail, "")
   ```
3. **ユーザーの状態を検証**: `user.Status == "active"` を確認
4. **認証サービス → OTP サービス**: チャレンジを作成し確認コードを送信（任意）
5. **ユーザーが確認コードを送信** → 認証サービスが受け取る（任意）
6. **認証サービス → OTP サービス**: 確認コードを検証（任意）
7. **認証サービス**: セッションを発行し、`user.Scope` と `user.Role` で認可ヘッダーを設定

**注意**: Warden は単独でも利用でき、上記の連携フローは任意です。

## 関連ドキュメント

- [OpenAPI 仕様](../../openapi.yaml) - 完全な OpenAPI 3.1 仕様
- [設定ドキュメント](CONFIGURATION.md) - API キーやその他のオプションの設定方法
- [セキュリティドキュメント](SECURITY.md) - セキュリティ機能とベストプラクティス
- [アーキテクチャドキュメント](ARCHITECTURE.md) - サービス連携のアーキテクチャ
