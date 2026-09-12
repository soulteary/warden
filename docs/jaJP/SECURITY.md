# セキュリティドキュメント

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

本ドキュメントでは、Warden のセキュリティ機能、セキュリティ設定、ベストプラクティスについて説明します。

## 実装済みのセキュリティ機能

1. **API 認証**: API キー認証に対応し、機微なエンドポイントを保護します
2. **SSRF 対策**: リモート設定 URL を厳密に検証し、サーバーサイドリクエストフォージェリを防ぎます
3. **入力検証**: すべての入力パラメーターを厳密に検証し、インジェクション攻撃を防ぎます
4. **レート制限**: IP 単位のレート制限により DDoS 攻撃を防ぎます
5. **TLS 検証**: 本番環境では TLS 証明書の検証を強制します
6. **エラー処理**: 本番環境では詳細なエラー情報を隠し、情報漏えいを防ぎます
7. **セキュリティレスポンスヘッダー**: セキュリティ関連の HTTP レスポンスヘッダーを自動的に付与します
8. **IP 許可リスト**: ヘルスチェックエンドポイント向けの IP 許可リストを設定できます
9. **設定ファイルの検証**: パストラバーサル攻撃を防ぎます
10. **JSON サイズ制限**: JSON レスポンスボディのサイズを制限し、メモリ枯渇攻撃を防ぎます
11. **ユーザー照会パラメーターの長さ制限**: 単一のパラメーター（`phone`／`mail`／`user_id`）は 512 バイトを超えてはならず、DoS やログ・キャッシュの肥大化を防ぎます
12. **監査ログの PII サニタイズ**: 監査に書き込まれる識別子は phone／mail をマスクし、監査ストレージが侵害された場合でも個人情報が露出しないようにします

## セキュリティのベストプラクティス

### 1. 本番環境の設定

**必須の設定**:
- 本番向けのハードニングを有効にするため、`ENVIRONMENT=production` を**必ず**設定してください。
- サービス認証の仕組みを**必ず**少なくとも 1 つ設定してください: `API_KEY`、HMAC v2、または mTLS。
- クライアント IP を正しく取得するため、`TRUSTED_PROXY_IPS` を**必ず**設定してください
- `HEALTH_CHECK_IP_WHITELIST` によってヘルスチェックへのアクセスを**必ず**制限してください（または `/health`、`/healthcheck` をネットワークやリバースプロキシで制限してください）
- `/metrics` を**必ず**制限してください。`ENVIRONMENT=production` では既定で認証が必要です。その状態を維持するか（あるいはリバースプロキシ／ネットワーク層でパスを制限し）、本番環境では `WARDEN_METRICS_REQUIRE_AUTH=false` を**設定しないでください**。

**設定例**:
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. 機微情報の取り扱い

**推奨される方法**:
- ✅ パスワードと鍵は環境変数に保存する
- ✅ Redis のパスワードにはパスワードファイル（`REDIS_PASSWORD_FILE`）を使う
- ✅ 設定ファイルではプレースホルダーやコメントを使う
- ✅ 設定ファイルのパーミッションを正しく設定する（例: `chmod 600`）

**推奨されない方法**:
- ❌ 設定ファイルにパスワードをハードコードする
- ❌ コマンドライン引数でパスワードを渡す（プロセス一覧に表示されます）
- ❌ 機微情報を含む設定ファイルをバージョン管理にコミットする

**例**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # 環境変数 REDIS_PASSWORD または REDIS_PASSWORD_FILE を使用

app:
  # api_key: ""  # 環境変数 API_KEY を使用
```

### 3. ネットワークセキュリティ

**必須の設定**:
- 本番環境では必ず HTTPS を使用する
- ファイアウォールの規則を設定してアクセスを制限する
- 既知の脆弱性を修正するため、依存関係を定期的に更新する

**推奨される設定**:
- リバースプロキシ（Nginx など）で SSL/TLS を処理する
- `TRUSTED_PROXY_IPS` を設定し、クライアントの実 IP を正しく取得する
- 強固なパスワードと API キーを使用する
- `HTTP_INSECURE_TLS` を無効にする（本番環境では `false` でなければなりません）

### 4. 監視と監査

**推奨される方法**:
- セキュリティイベントのログを監視する
- アクセスログを定期的に確認する
- CI/CD でセキュリティスキャンツールを使用する
- アラートの仕組みを整える

**ログレベルの管理**:
- 本番環境では `info` または `warn` レベルを推奨します
- ログレベルの変更操作はすべてセキュリティ監査ログに記録されます
- ログレベルは `/log/level` API で動的に変更できます（API キー認証が必要）

## API のセキュリティ

### API キー認証

一部の API エンドポイントは API キー認証を必要とします。

**認証が必要なエンドポイント**:
- `GET /` - ユーザー一覧の取得
- `GET /user` - 単一ユーザーの照会
- `GET /log/level` - ログレベルの取得
- `POST /log/level` - ログレベルの設定

**認証が不要なエンドポイント**（本番環境では別の手段で必ず保護してください）:
- `GET /health` - ヘルスチェック（`HEALTH_CHECK_IP_WHITELIST` またはネットワーク分離を**必ず**設定してください）
- `GET /healthcheck` - ヘルスチェック（同上）
- `GET /metrics` - Prometheus メトリクス（スクレイピング用の API キーを**必ず**設定するか、リバースプロキシ／ネットワークで制限してください。公開しないでください）

**認証方式**:
1. **X-API-Key ヘッダー**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer ヘッダー**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### レート制限

既定では、API リクエストはレート制限によって保護されています。

- **上限**: 1 分あたり 60 リクエスト
- **ウィンドウ**: 1 分
- **超過時**: `429 Too Many Requests` を返します

設定ファイルで調整できます。

```yaml
rate_limit:
  rate: 60  # 1 分あたりのリクエスト数
  window: 1m
```

### IP 許可リスト

2 種類の IP 許可リストを設定できます。

1. **グローバル IP 許可リスト**（`IP_WHITELIST`）:
   - すべてのエンドポイントへのアクセスを制限します
   - CIDR 範囲形式に対応します

2. **ヘルスチェック IP 許可リスト**（`HEALTH_CHECK_IP_WHITELIST`）:
   - `/health` と `/healthcheck` エンドポイントのみを制限します
   - CIDR 範囲形式に対応します

**設定例**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## データセキュリティ

### リモート設定 API のセキュリティ

- リモート設定 API では認証の仕組み（Authorization ヘッダー）を使用してください
- HTTPS プロトコルの利用を推奨します
- リモート API の TLS 証明書を検証してください（本番環境では必須）

### Redis のセキュリティ

- Redis にはパスワード保護を設定してください
- 環境変数 `REDIS_PASSWORD` または `REDIS_PASSWORD_FILE` を使用してください
- Redis へのネットワークアクセスを制限してください（アプリケーションサーバーからのみ許可）
- 既知の脆弱性を修正するため、Redis を定期的に更新してください

### データファイルのセキュリティ

- `data.json` ファイルのパーミッションが正しく設定されていることを確認してください
- 機微なデータをバージョン管理にコミットしないでください
- データファイルを定期的にバックアップしてください

## セキュリティレスポンスヘッダー

Warden は以下のセキュリティ関連 HTTP レスポンスヘッダーを自動的に付与します。

- `X-Content-Type-Options: nosniff` - MIME タイプのスニッフィングを防ぎます
- `X-Frame-Options: DENY` - クリックジャッキングを防ぎます
- `X-XSS-Protection: 1; mode=block` - XSS 対策

## エラー処理

### 本番モード

本番モード（`ENVIRONMENT=production`）では次のようになります。

- 情報漏えいを防ぐため、詳細なエラー情報を隠します
- 汎用的なエラーメッセージを返します
- 詳細なエラー情報はログにのみ記録されます

### 開発モード

開発モードでは次のようになります。

- デバッグのために詳細なエラー情報を表示します
- スタックトレース情報を含みます

## セキュリティ監査

リリースのハードニングと検証の指針については [Release Security](../RELEASE_SECURITY.md) を参照してください。

## 脆弱性の報告

セキュリティ上の脆弱性を発見した場合は、次の方法で報告してください。

1. 非公開のセキュリティ Issue を作成する（対応している場合）
2. プロジェクトのメンテナーにメールを送る
3. 修正されるまで脆弱性を公開しない

## サービス間認証（任意）

他のサービス（Stargate など）と連携する場合、サービス間認証によって安全性を確保できます。**mTLS と HMAC が実装済み**で、認証の優先順位は **mTLS > HMAC > API キー** です。Warden は次の方式に対応しています。

**注意**: Warden を単独で利用する場合、サービス間認証は任意です。

### mTLS（推奨）

相互 TLS 証明書による認証を行い、より高い安全性を確保します。

**設定**:

1. **証明書を生成する**:
   ```bash
   # CA 証明書を生成
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Warden のサーバー証明書を生成
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Stargate のクライアント証明書を生成
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Warden の設定**（環境変数）:
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Stargate の設定**:
   - クライアント証明書のパスを設定する
   - Warden のサーバー証明書を検証するため、CA 証明書のパスを設定する

### HMAC 署名

HMAC-SHA256 署名でリクエストを検証します。導入がより容易です。

**署名アルゴリズム**:
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**リクエストヘッダー**:
- `X-Signature`: HMAC 署名の値
- `X-Timestamp`: Unix タイムスタンプ（秒）
- `X-Key-Id`: キー ID（署名に含まれ、安全な鍵ローテーションに使用）
- `X-Nonce`: 一意な 128 ビットの 16 進数ノンス
- `X-Signature-Version`: `v2`

**Warden の設定**（環境変数）:
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # タイムスタンプの許容誤差（秒）。既定値 60
export WARDEN_HMAC_ALLOW_V1=false          # 既定値。期間を限定した旧方式の移行時のみ true に設定
```

本番環境の設定では、各 HMAC シークレットに少なくとも 32 バイトの生バイト列が必要です。
シークレットは暗号論的に安全な乱数源で生成し、シークレットマネージャーに保管してください。
上記の例示値をそのまま使い回さないでください。

**Go SDK の例**:
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**検証ルール**:
- Warden はタイムスタンプが許容範囲内か検証します（既定では ±60 秒）
- Warden はキー ID を含むすべての正規化フィールドを検証し、再利用されたノンスを拒否します
- 署名の検証に失敗した場合は `401 Unauthorized` を返します

### 設定の優先順位

1. **mTLS**: TLS 証明書が設定されている場合は mTLS が優先されます
2. **HMAC**: mTLS が未設定の場合は HMAC 署名が使用されます
3. **API キー**: いずれも未設定の場合は API キー認証にフォールバックします（サービス間呼び出しには推奨されません）

### セキュリティに関する推奨事項

1. **本番環境**: サービス間認証には mTLS の利用を強く推奨します
2. **鍵管理**: 鍵と証明書の保管には鍵管理サービス（HashiCorp Vault など）を使用してください
3. **鍵のローテーション**: HMAC キーと TLS 証明書を定期的にローテーションしてください
4. **ネットワーク分離**: 可能であれば、ネットワークポリシーで Warden へのアクセスを Stargate のみに制限してください

## 関連ドキュメント

- [設定ドキュメント](CONFIGURATION.md) - セキュリティ関連の設定オプション
- [デプロイドキュメント](DEPLOYMENT.md) - 本番環境へのデプロイに関する推奨事項
- [API ドキュメント](API.md) - API のセキュリティ機能
- [アーキテクチャドキュメント](ARCHITECTURE.md) - サービス連携のアーキテクチャ
