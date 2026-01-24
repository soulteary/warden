# Warden OpenTelemetry Tracing

Warden サービスは、サービス間の呼び出しチェーンの監視とデバッグのための OpenTelemetry 分散トレーシングをサポートしています。

## 機能

- **自動 HTTP リクエストトレーシング**: すべての HTTP リクエストに対して自動的に span を作成
- **ユーザークエリトレーシング**: `/user` エンドポイントに詳細なトレーシング情報を追加
- **コンテキスト伝播**: W3C Trace Context 標準をサポートし、Stargate および Herald サービスとシームレスに統合
- **設定可能**: 環境変数または設定ファイルで有効/無効化

## 設定

### 環境変数

```bash
# OpenTelemetry トレーシングを有効化
OTLP_ENABLED=true

# OTLP エンドポイント（例：Jaeger、Tempo、OpenTelemetry Collector）
OTLP_ENDPOINT=http://localhost:4318
```

### 設定ファイル（YAML）

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

## コア Span

### HTTP リクエスト Span

すべての HTTP リクエストは、以下の属性を含む span を自動的に作成します：
- `http.method`: HTTP メソッド
- `http.url`: リクエスト URL
- `http.status_code`: レスポンスステータスコード
- `http.user_agent`: ユーザーエージェント
- `http.remote_addr`: クライアントアドレス

### ユーザークエリ Span (`warden.get_user`)

`/user` エンドポイントへのクエリは、以下を含む専用の span を作成します：
- `warden.query.phone`: クエリされた電話番号（マスク済み）
- `warden.query.mail`: クエリされたメールアドレス（マスク済み）
- `warden.query.user_id`: クエリされたユーザー ID
- `warden.user.found`: ユーザーが見つかったかどうか
- `warden.user.id`: 見つかったユーザー ID

## 使用例

### トレーシングを有効にして Warden を起動

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### コードでトレーシングを使用

```go
import "github.com/soulteary/warden/internal/tracing"

// 子 span を作成
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// 属性を設定
span.SetAttributes(attribute.String("key", "value"))

// エラーを記録
if err != nil {
    tracing.RecordError(span, err)
}
```

## Stargate および Herald との統合

Warden のトレーシングは、Stargate および Herald サービスのトレーシングコンテキストと自動的に統合されます：

1. **Stargate** が Warden を呼び出す際、HTTP ヘッダー経由で trace context を渡します
2. **Warden** が自動的に抽出し、トレースチェーンを継続します
3. 3 つのサービスの span が同じ trace に表示されます

## サポートされているトレーシングバックエンド

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **その他の OTLP 互換バックエンド**

## パフォーマンスの考慮事項

- トレーシングはデフォルトでバッチエクスポートを使用し、パフォーマンスへの影響を最小限に抑えます
- サンプリング率でトレースデータ量を制御できます
- 本番環境ではサンプリング戦略を使用することを推奨します（現在は全サンプリング、開発環境に適しています）

## トラブルシューティング

### トレーシングが有効になっていない

環境変数を確認：
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### トレースデータがバックエンドに到達しない

1. OTLP エンドポイントがアクセス可能か確認
2. ネットワーク接続を確認
3. Warden ログのエラーメッセージを確認

### Span が欠落している

リクエスト処理で新しい context を作成するのではなく、`r.Context()` を使用してコンテキストを渡すようにしてください。
