# デプロイドキュメント

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

本ドキュメントでは、Docker によるデプロイやローカルデプロイなど、Warden サービスのデプロイ方法を説明します。

## 前提条件

- Go 1.27 以上（[go.mod](../../go.mod) を参照）
- Redis（分散ロックとキャッシュ用）
- Docker（任意。コンテナでデプロイする場合）

## Docker によるデプロイ

> 🚀 **クイックデプロイ**: 完全な Docker Compose 設定例は[サンプルディレクトリ](../../example/README.md) / [示例目录](../../example/README.md) を参照してください。
> - [シンプルな例](../../example/basic/docker-compose.yml) / [简单示例](../../example/basic/docker-compose.yml) - 基本的な Docker Compose 設定
> - [応用例](../../example/advanced/docker-compose.yml) / [复杂示例](../../example/advanced/docker-compose.yml) - モック API を含む完全な設定

### ビルド済みイメージを使う（推奨）

Warden はビルド済みの Docker イメージを提供しており、GitHub Container Registry（GHCR）から直接取得できます。手動でのビルドは不要です。

```bash
# 最新バージョンのイメージを取得
docker pull ghcr.io/soulteary/warden:latest

# コンテナを実行
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api/data.json \
  -e KEY="Bearer your-token-here" \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **ヒント**: ビルド済みイメージを使えば、ローカルにビルド環境がなくてもすぐに始められます。イメージは自動的に更新されるため、常に最新バージョンを利用できます。

### Docker Compose を使う

1. **環境変数ファイルを準備する**
   
   プロジェクトのルートディレクトリに `.env.example` ファイルがある場合は、コピーできます。
   ```bash
   cp .env.example .env
   ```
   
   `.env.example` ファイルがない場合は、次の内容で `.env` ファイルを手動作成できます。
   ```env
   # サーバー設定
   PORT=8081
   
   # Redis 設定
   REDIS=warden-redis:6379
   # Redis パスワード（任意。設定ファイルではなく環境変数の利用を推奨）
   # REDIS_PASSWORD=your-redis-password
   # またはパスワードファイルを使用（より安全）
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # リモートデータ API
   CONFIG=http://example.com/api/data.json
   # リモート設定 API の認証キー
   KEY=Bearer your-token-here
   
   # タスク設定
   INTERVAL=5
   
   # アプリケーションモード
   MERGE_MODE=DEFAULT
   
   # HTTP クライアント設定（任意）
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # API キー（API 認証用。本番環境では必須）
   API_KEY=your-api-key-here
   
   # ヘルスチェックの IP 許可リスト（任意。カンマ区切り）
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # 信頼するプロキシ IP のリスト（任意。カンマ区切り。リバースプロキシ環境向け）
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # ログレベル（任意）
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **セキュリティに関する注意**: `.env` ファイルには機微な情報が含まれます。バージョン管理にコミットしないでください。`.env` ファイルはすでに `.gitignore` で除外されています。上記の内容をテンプレートとして `.env` ファイルを作成してください。

2. **サービスを起動する**
```bash
docker-compose up -d
```

### イメージを手動でビルドする

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### コンテナを実行する

```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api \
  -e KEY="Bearer token" \
  warden-release
```

## ローカルデプロイ

### 1. プロジェクトをクローンする

```bash
git clone <repository-url>
cd warden
```

### 2. 依存関係をインストールする

```bash
go mod download
```

### 3. ローカルデータファイルを設定する

`data.json` ファイルを作成します（`data.example.json` を参照）。
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**注意**: `data.json` ファイルは次のフィールドに対応しています。
- `phone`（必須）: ユーザーの電話番号
- `mail`（必須）: ユーザーのメールアドレス
- `user_id`（任意）: ユーザーの一意な識別子。指定がない場合は自動生成されます
- `status`（任意）: ユーザーの状態。「active」「inactive」「suspended」など。値を省略した場合の既定値は「inactive」です
- `scope`（任意）: ユーザーの権限スコープの配列。例: `["read", "write"]`
- `role`（任意）: ユーザーのロール。「admin」「user」など

完全な例については `data.example.json` ファイルを参照してください。

### 4. サービスを起動する

```bash
go run .
```

## 本番環境へのデプロイに関する推奨事項

### 1. リバースプロキシを使う

本番環境では Nginx や Traefik などのリバースプロキシの利用を推奨します。

**Nginx の設定例**:
```nginx
upstream warden {
    server localhost:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://warden;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. HTTPS を使う

本番環境では必ず HTTPS を使用してください。次の方法で実現できます。

- Let's Encrypt の無料証明書を利用する
- リバースプロキシ（Nginx など）で SSL/TLS を処理する
- 環境変数 `TRUSTED_PROXY_IPS` を設定し、クライアントの実 IP を正しく取得する

### 3. 監視を設定する

- Prometheus でメトリクスを収集する（`/metrics` エンドポイント経由）
- ヘルスチェックを設定する（`/health` エンドポイント経由）
- ログの収集と分析の仕組みを整える

### 4. 高可用性デプロイ

- 複数のインスタンスをデプロイし、ロードバランサーでリクエストを分散する
- 共有 Redis インスタンスを使用してデータの一貫性を確保する
- 自動再起動とフェイルオーバーを設定する

### 5. リソース制限

Docker Compose または Kubernetes でリソース制限を設定します。

```yaml
services:
  warden:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Kubernetes によるデプロイ

### 基本的なデプロイ

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "redis-service:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: warden-service
spec:
  selector:
    app: warden
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: LoadBalancer
```

## パフォーマンス最適化

### 1. Redis の設定

- Redis の永続化を使用する（RDB または AOF）
- 適切な Redis のメモリ上限を設定する
- 必要に応じて Redis クラスターを使用する

### 2. アプリケーションの設定

- `HTTP_MAX_IDLE_CONNS` を調整して接続プールを最適化する
- 適切な `INTERVAL` を設定し、即時性と効率のバランスを取る
- 適切なマージモード（`MERGE_MODE`）を使用する

### 3. 監視とチューニング

wrk による負荷試験の結果（30 秒間、16 スレッド、100 コネクション）:

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

実際の負荷に応じて設定パラメーターを調整してください。

## 任意の連携デプロイ（Stargate／Herald との組み合わせ）

Warden は単独でデプロイして利用することも、任意で Stargate や Herald と連携させることもできます。以下は任意の連携デプロイの設定例です。

**注意**: 以下の連携デプロイのシナリオは任意であり、Warden は完全に単独でデプロイして利用できます。

### Docker Compose による連携例

Stargate + Warden + Herald を連携させる完全なデプロイ設定:

```yaml
version: '3.8'

services:
  # Warden サービス
  warden:
    image: ghcr.io/soulteary/warden:latest
    container_name: warden
    ports:
      - "8081:8081"
    networks:
      - auth-network
    environment:
      - PORT=8081
      - REDIS=warden-redis:6379
      - API_KEY=${WARDEN_API_KEY}
      - MERGE_MODE=DEFAULT
      # サービス間認証の設定（HMAC の例）
      - WARDEN_HMAC_KEYS=${WARDEN_HMAC_KEYS}
      - WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
    volumes:
      - ./warden-data.json:/app/data.json:ro
    healthcheck:
      test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
      interval: 10s
      timeout: 1s
      retries: 3
    depends_on:
      - warden-redis

  # Warden 用 Redis
  warden-redis:
    image: redis:7.4-alpine
    container_name: warden-redis
    networks:
      - auth-network
    volumes:
      - warden-redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 1s
      retries: 3

  # Stargate サービス（設定例）
  stargate:
    image: ghcr.io/soulteary/stargate:latest
    container_name: stargate
    ports:
      - "8080:8080"
    networks:
      - auth-network
    environment:
      - STARGATE_WARDEN_BASE_URL=http://warden:8081
      - STARGATE_WARDEN_AUTH_TYPE=hmac
      - STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
      - STARGATE_WARDEN_HMAC_SECRET=${WARDEN_HMAC_SECRET}
      - STARGATE_HERALD_BASE_URL=http://herald:8082
    depends_on:
      - warden
      - herald

  # Herald サービス（設定例）
  herald:
    image: ghcr.io/soulteary/herald:latest
    container_name: herald
    ports:
      - "8082:8082"
    networks:
      - auth-network
    environment:
      - HERALD_REDIS_URL=redis://herald-redis:6379
    depends_on:
      - herald-redis

  # Herald 用 Redis
  herald-redis:
    image: redis:7.4-alpine
    container_name: herald-redis
    networks:
      - auth-network
    volumes:
      - herald-redis-data:/data

networks:
  auth-network:
    driver: bridge

volumes:
  warden-redis-data:
  herald-redis-data:
```

### 環境変数の設定

`.env` ファイルを作成します。

```bash
# Warden の API キー
WARDEN_API_KEY=your-warden-api-key-here

# Warden の HMAC キー（JSON 形式）
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Stargate が使用する HMAC シークレット（WARDEN_HMAC_KEYS のキーに対応）
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### ネットワーク設定

相互に通信できるよう、すべてのサービスを同じ Docker ネットワークに配置してください。

- **Warden**: ポート `8081` で待ち受け、Stargate から呼び出されます
- **Stargate**: ポート `8080` で待ち受け、Traefik の forwardAuth サービスとして機能します
- **Herald**: ポート `8082` で待ち受け、Stargate から呼び出されます

### サービスの依存関係

- **Stargate** は **Warden** と **Herald** に依存します
- **Warden** は **warden-redis** に依存します（Redis を有効にしている場合。任意）
- **Herald** は **herald-redis** に依存します

### ヘルスチェック

正常な稼働を確認するため、すべてのサービスでヘルスチェックを設定してください。

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### 本番環境に関する推奨事項

1. **独立した Redis インスタンスを使う**: データの競合を避けるため、Warden と Herald は独立した Redis インスタンスを使用してください
2. **サービス間認証を設定する**: 本番環境では mTLS または HMAC 署名の設定が必須です
3. **鍵管理サービスを使う**: HashiCorp Vault などのサービスで鍵と証明書を管理してください
4. **ネットワーク分離**: Docker のネットワークポリシーでサービス間のアクセスを制限してください
5. **監視とログ**: 統一された監視とログ収集の仕組みを整えてください

### Kubernetes での連携デプロイ

Kubernetes にデプロイする場合は、次を推奨します。

1. **Service を使う**: サービスごとに Kubernetes Service を作成する
2. **ConfigMap と Secret を使う**: 設定と鍵を保存する
3. **NetworkPolicy を使う**: サービス間のネットワークアクセスを制限する
4. **Ingress を使う**: Traefik Ingress を設定して Stargate へルーティングする

Kubernetes の設定例:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: warden
spec:
  selector:
    app: warden
  ports:
    - port: 8081
      targetPort: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: ghcr.io/soulteary/warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "warden-redis:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        - name: WARDEN_HMAC_KEYS
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: hmac-keys
```

## 関連ドキュメント

- [設定ドキュメント](CONFIGURATION.md) - 詳細な設定オプション
- [セキュリティドキュメント](SECURITY.md) - セキュリティ設定とベストプラクティス
- [アーキテクチャ設計ドキュメント](ARCHITECTURE.md) - システムアーキテクチャの理解
- [API ドキュメント](API.md) - API インターフェースと連携例
