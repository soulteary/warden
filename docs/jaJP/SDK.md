# SDK 利用ドキュメント

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](../koKR/SDK.md)

Warden は他プロジェクトへ容易に組み込めるよう Go SDK を提供しています。SDK はキャッシュや認証などに対応した、すっきりとした API を備えています。

## 特長

- 🚀 **シンプルで扱いやすい**: すっきりとした API を提供
- ⚡ **高性能**: キャッシュを内蔵（GetUsers）。直接照会（GetUserByIdentifier）で API 呼び出しを削減
- 🔒 **安全**: API キー認証に対応。エラー処理で機微な情報が漏れません
- 📦 **柔軟**: タイムアウトやキャッシュ TTL などを設定可能
- 🔌 **拡張可能**: 独自のロガー実装に対応
- 🎯 **スマートなフォールバック**: CheckUserInList は電話番号が見つからない場合、自動的にメールアドレスへフォールバックします

## SDK のインストール

```bash
go get github.com/soulteary/warden/pkg/warden
```

## クイックスタート

### 基本的な使い方

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // クライアントのオプションを作成
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // クライアントを作成
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // ユーザー一覧を取得
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // ユーザーが一覧にあるか確認（phone、mail、またはその両方を指定可能）
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // phone だけ、または mail だけを使うこともできます
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // ユーザーの詳細を取得
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### 独自のロガーを使う

SDK は独自のロガー実装に対応しています。たとえば logrus を使う場合は次のとおりです。

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    logger := logrus.StandardLogger()
    
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithLogger(warden.NewLogrusAdapter(logger))
    
    client, err := warden.NewClient(opts)
    // ...
}
```

### ページ分割された照会

```go
// ページ分割されたユーザー一覧を取得
resp, err := client.GetUsersPaginated(ctx, 1, 10) // 1 ページ目、1 ページあたり 10 件
if err != nil {
    panic(err)
}

fmt.Printf("Total users: %d\n", resp.Pagination.Total)
fmt.Printf("Total pages: %d\n", resp.Pagination.TotalPages)
for _, user := range resp.Data {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
}
```

### 単一ユーザーの情報を取得する

```go
// 電話番号でユーザー情報を取得
user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
        println("User not found")
    } else {
        panic(err)
    }
} else {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
    if user.IsActive() {
        println("User is active")
    }
}

// メールアドレスでユーザー情報を取得
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// ユーザー ID でユーザー情報を取得
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### キャッシュをクリアする

```go
// クライアントのキャッシュを手動でクリア
client.ClearCache()

// またはエイリアスを使用
client.InvalidateCache()
```

### 独自の HTTP トランスポート

```go
import "net/http"

// 独自のトランスポートを作成
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### HMAC v2 によるリクエスト署名

```go
opts := warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET"))

client, err := warden.NewClient(opts)
```

SDK はキー ID、タイムスタンプ、ノンス、パス／クエリ、メソッド、ボディのハッシュに署名します。
キー ID とシークレットは両方とも設定する必要があります。片方だけの場合は署名なしのリクエストを送らず、
`ErrCodeInvalidConfig` を返します。

### リトライの設定

```go
// リトライオプションを設定
retryOpts := warden.DefaultRetryOptions()
retryOpts.MaxRetries = 3
retryOpts.RetryDelay = 100 * time.Millisecond
retryOpts.MaxRetryDelay = 5 * time.Second
retryOpts.BackoffMultiplier = 2.0

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithRetry(retryOpts)

client, err := warden.NewClient(opts)
```

### イベント駆動によるキャッシュ無効化

```go
// キャッシュ無効化イベント用のチャネルを作成
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // 重要: クローズしてバックグラウンドのリスナーを停止する

// あとから外部イベントでキャッシュ無効化をトリガー
invalidationCh <- struct{}{}

// シグナルを受け取るとキャッシュは自動的にクリアされます
```

## API リファレンス

### Options

`Options` 構造体はクライアントの設定に使用します。

- `BaseURL`: Warden サービスのアドレス（必須）
- `APIKey`: API キー（任意）
- `Timeout`: HTTP リクエストのタイムアウト（既定値 10 秒）
- `CacheTTL`: キャッシュの TTL（既定値 5 分）
- `Logger`: ロガーインターフェース（任意。既定は NoOpLogger）
- `Transport`: 独自の HTTP トランスポート（任意）
- `HMACKeyID` / `HMACSecret`: HMAC v2 の署名ペア（両方設定するか、どちらも設定しないか）
- `TLSConfig`: TLS／mTLS のクライアント設定（任意）
- `Retry`: リトライの設定（任意。既定はリトライなし）
- `CacheInvalidationChannel`: イベント駆動によるキャッシュ無効化用のチャネル（任意）

### クライアントのメソッド

#### `NewClient(opts *Options) (*Client, error)`

新しい Warden クライアントを作成します。

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

全ユーザーの一覧を取得します。キャッシュが有効な場合は、キャッシュされたデータをそのまま返します。

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

ページ分割されたユーザー一覧を取得します。

- `page`: ページ番号（1 から開始）
- `pageSize`: 1 ページあたりの件数

`PaginatedResponse` を返します。内容は次のとおりです。
- `Data`: ユーザー一覧
- `Pagination`: ページ情報（ページ番号、ページサイズ、総件数、総ページ数）

**注意:** このメソッドはキャッシュを使用しません。呼び出しのたびに API から最新のデータを取得します。

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

識別子を指定して単一ユーザーの情報を取得します。

- `phone`: ユーザーの電話番号（任意。ただし phone、mail、userID のいずれか 1 つは指定が必要）
- `mail`: ユーザーのメールアドレス（任意）
- `userID`: ユーザーの一意な識別子（任意）

**重要:** `phone`、`mail`、`userID` のうち、ちょうど 1 つを指定する必要があります。

`*AllowListUser` とエラーを返します。ユーザーが存在しない場合は `ErrCodeNotFound` エラーを返します。

**注意:** このメソッドはキャッシュを使用しません。呼び出しのたびに API から最新のデータを取得します。

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

ユーザーが許可リストに含まれているかを確認します。

- `phone`: ユーザーの電話番号（任意）
- `mail`: ユーザーのメールアドレス（任意）

ユーザーが存在する場合（電話番号またはメールアドレスで一致した場合）は `true`、そうでなければ `false` を返します。

**動作:**
- `phone` と `mail` の両方が指定された場合は `phone` が優先されます
- `phone` による照会が失敗し（`NotFound` エラー）、`mail` が空でない場合は、自動的に `mail` による照会へフォールバックします
- `phone` による照会が成功したもののユーザーの状態が有効でない場合、`mail` へはフォールバックしません（ユーザーは既に見つかっているため）
- `phone` による照会が失敗し、そのエラーが `NotFound` 以外（ネットワークエラーなど）の場合、`mail` へはフォールバックしません
- 入力は自動的に正規化されます。`phone` は前後の空白を除去し、`mail` は空白を除去して小文字に変換します
- 本メソッドは照会に `GetUserByIdentifier` を使用しており、ユーザー一覧を走査するより効率的です
- 状態が「active」のユーザーのみ `true` を返します

#### `ClearCache()`

クライアント内部のキャッシュをクリアします。

#### `InvalidateCache()`

`ClearCache()` のエイリアスです。イベント駆動の無効化との一貫性のために用意されています。

#### `Close()`

バックグラウンドの goroutine（キャッシュ無効化のリスナーなど）を停止し、リソースを解放します。
クライアントが不要になった時点で呼び出してください。

## 型定義

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // ユーザーの電話番号
    Mail   string   `json:"mail"`    // ユーザーのメールアドレス
    UserID string   `json:"user_id"` // ユーザーの一意な識別子（任意。未指定なら自動生成）
    Status string   `json:"status"`  // ユーザーの状態（例: "active"、"inactive"、"suspended"）
    Scope  []string `json:"scope"`   // ユーザーの権限スコープ（任意）
    Role   string   `json:"role"`    // ユーザーのロール（任意）
}
```

**メソッド:**
- `IsActive() bool`: ユーザーの状態が「active」かどうかを確認します
- `IsValid() bool`: ユーザーの状態が有効かどうかを確認します（現時点では「active」のみ対応）

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // 現在のページ番号（1 から開始）
    PageSize   int `json:"page_size"`   // 1 ページあたりの件数
    Total      int `json:"total"`       // 総レコード数
    TotalPages int `json:"total_pages"` // 総ページ数
}
```

## エラー処理

SDK はエラーコードと詳細情報を持つ独自のエラー型を使用します。

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // 認証エラーを処理
        case warden.ErrCodeRequestFailed:
            // リクエスト失敗を処理
        case warden.ErrCodeNotFound:
            // 「見つかりません」エラーを処理
        case warden.ErrCodeServerError:
            // サーバーエラーを処理
        // ...
        }
    }
}
```

### エラーコード

- `ErrCodeInvalidConfig`: 設定が不正
- `ErrCodeRequestFailed`: リクエスト失敗
- `ErrCodeInvalidResponse`: レスポンス形式が不正
- `ErrCodeUnauthorized`: 未認可
- `ErrCodeNotFound`: 見つかりません
- `ErrCodeServerError`: サーバーエラー

## ベストプラクティス

1. **クライアントを再利用する**: クライアントは一度だけ作成し、アプリケーションのライフサイクル全体で再利用します
2. **キャッシュ TTL を適切に設定する**: データの更新頻度に応じて適切なキャッシュ時間を設定します
3. **Context を使う**: コンテキストを渡し、キャンセルとタイムアウトを制御できるようにします
4. **エラー処理**: エラーは必ず確認して処理します
5. **ログ出力**: 本番環境では適切なロガー実装を使用します
6. **クライアントを閉じる**: クライアントが不要になったら `Close()` を呼び、バックグラウンドの goroutine を停止します
7. **リトライを設定する**: 本番環境ではリトライを有効にし、一時的な障害に備えます
8. **独自トランスポート**: 高度な用途（TLS、プロキシ、コネクションプールなど）では独自のトランスポートを使用します

## 設計ドキュメント

### 設計原則

1. **シンプルで扱いやすい**: すっきりとした API を提供
2. **高性能**: キャッシュの内蔵により API 呼び出しを削減
3. **スレッドセーフ**: すべてのメソッドは並行安全
4. **柔軟な設定**: タイムアウト、キャッシュ、ロガーなどをカスタマイズ可能

### アーキテクチャ設計

#### 主要コンポーネント

1. **Client**: HTTP クライアントのラッパー
2. **Cache**: スレッドセーフなインメモリキャッシュ
3. **Options**: 設定オプション（Builder パターン）
4. **Logger**: ロガーインターフェース（各種ロギングライブラリに対応）

#### 並行安全性

- `http.Client` は並行安全です
- `Cache` は `sync.RWMutex` によりスレッドセーフを担保します
- `Client` のフィールドは作成後すべて読み取り専用です
- すべてのメソッドはスレッドセーフで、複数の goroutine から並行して呼び出せます

#### キャッシュ戦略

1. **GetUsers()**: キャッシュを使用します
   - まずキャッシュを確認します
   - キャッシュが有効ならそのまま返します
   - キャッシュが無効または存在しない場合は API から取得してキャッシュを更新します

2. **GetUsersPaginated()**: キャッシュを使用しません
   - 理由: ページ分割のパラメーターが異なれば結果も異なるため
   - ページ分割パラメーター単位でキャッシュするのは複雑になります
   - 現在の設計: 毎回 API から取得し、データの正確性を確保します

3. **GetUserByIdentifier()**: キャッシュを使用しません
   - 理由: 単一ユーザーの最新情報を取得し、データの即時性を確保する必要があるため
   - 呼び出しのたびに API から取得し、キャッシュによる不整合を避けます

4. **CheckUserInList()**: キャッシュを使用しません
   - `GetUserByIdentifier()` を使って単一ユーザーを直接照会します
   - 呼び出しのたびに API へリクエストし、データの即時性を確保します
   - スマートなフォールバックに対応: 電話番号での照会が失敗（NotFound）し、mail が空でない場合は自動的に mail での照会へ切り替えます
   - 性能最適化: 単一ユーザーを直接照会するほうが、ユーザー一覧全体を走査するより効率的です

#### CheckUserInList の実装方針

`CheckUserInList()` メソッドは次の方針で動作します。

1. **入力の正規化**: phone と mail の前後の空白を自動的に除去し、mail は小文字に変換します
2. **優先順位**: phone と mail の両方が指定された場合は phone を優先します
3. **スマートなフォールバック**:
   - phone での照会が `NotFound` エラーを返し、mail が空でない場合は自動的に mail での照会へ切り替えます
   - phone での照会が成功したもののユーザーの状態が有効でない場合、mail へはフォールバックしません（ユーザーは既に見つかっているため）
   - phone での照会がそれ以外のエラー（ネットワークエラーなど）に遭遇した場合、mail へはフォールバックしません
4. **状態の検証**: 状態が「active」のユーザーのみ `true` を返します
5. **性能最適化**: `GetUserByIdentifier()` による直接照会を使い、ユーザー一覧全体の取得を避けます

### RetryOptions

`RetryOptions` 構造体はリトライの挙動を設定します。

- `MaxRetries`: 最大リトライ回数（既定値 0、リトライなし）
- `RetryDelay`: リトライ間隔の初期値（既定値 100ms）
- `MaxRetryDelay`: リトライ間隔の上限（既定値 5s）
- `BackoffMultiplier`: 指数バックオフの倍率（既定値 2.0）
- `RetryableStatusCodes`: リトライを発生させる HTTP ステータスコード（既定値: 5xx）

**注意:** ネットワークエラーは常にリトライ対象です。クライアントエラー（4xx）はリトライしません。

### 既知の制限

1. **ページ分割のキャッシュ**: `GetUsersPaginated()` はキャッシュを使用しません
   - データの正確性を確保するための意図的な設計です
   - ページ分割のキャッシュが必要な場合は、より複雑な戦略を実装できます

2. **単一ユーザー照会のキャッシュ**: `GetUserByIdentifier()` と `CheckUserInList()` はキャッシュを使用しません
   - データの即時性を確保するための意図的な設計です
   - キャッシュが必要な場合は、ユーザー識別子に基づく戦略を実装できます

### 今後の改善

1. リクエスト／レスポンスのミドルウェア対応
2. メトリクス収集への対応
3. コネクションプール設定への対応
4. サーキットブレーカーパターンへの対応

## 完全な例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // クライアントを作成
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    client, err := warden.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // 全ユーザーを取得
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // 電話番号で単一ユーザーを取得
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            fmt.Println("User not found")
        } else {
            log.Fatal(err)
        }
    } else {
        fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
    }

    // ページ分割された照会
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // ユーザーを確認
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // キャッシュをクリア
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## 関連ドキュメント

- [API ドキュメント](API.md) - API エンドポイントの詳細
- [設定ドキュメント](CONFIGURATION.md) - サーバーの設定オプション
