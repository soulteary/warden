# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

ローカルおよびリモート設定ソースからのデータ同期とマージをサポートする高性能な許可リスト（AllowList）ユーザーデータサービス。

![Warden](.github/assets/banner.jpg)

> **Warden**（看守者）—— スターゲートの看守者であり、誰が通過でき、誰が拒否されるかを決定します。スターゲートの看守者がスターゲートを守るように、Warden はあなたの許可リストを守り、承認されたユーザーのみが通過できるようにします。

## 📋 概要

Warden は、Go で開発された軽量な HTTP API サービスで、主に許可リストユーザーデータ（電話番号とメールアドレス）の提供と管理に使用されます。このサービスは、ローカル設定ファイルとリモート API からのデータ取得をサポートし、リアルタイムのパフォーマンスと信頼性を確保するための複数のデータマージ戦略を提供します。

Warden は**独立して使用**することも、より大きな認証アーキテクチャの一部として他のサービス（Stargate や Herald など）と統合することもできます。詳細なアーキテクチャ情報については、[アーキテクチャドキュメント](docs/enUS/ARCHITECTURE.md)を参照してください。

## ✨ 主要機能

- 🚀 **高性能**: 平均レイテンシ 21ms で毎秒 5000 以上のリクエストをサポート
- 🔄 **複数のデータソース**: ローカル設定ファイルとリモート API
- 🎯 **柔軟な戦略**: 6 つのデータマージモード（リモート優先、ローカル優先、リモートのみ、ローカルのみなど）
- ⏰ **スケジュール更新**: Redis 分散ロックによる自動データ同期
- 📦 **コンテナ化デプロイ**: 完全な Docker サポート、すぐに使用可能
- 🌐 **多言語サポート**: 7 つの言語をサポートし、自動言語検出

## 🚀 クイックスタート

### オプション 1: Docker（推奨）

最も簡単な方法は、事前に構築された Docker イメージを使用することです：

```bash
# 最新のイメージをプル
docker pull ghcr.io/soulteary/warden:latest

# データファイルを作成
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# コンテナを実行
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **ヒント**: Docker Compose の完全な例については、[例ディレクトリ](example/README.md)を参照してください。

### オプション 2: ソースから

1. **クローンしてビルド**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **データファイルを作成**
`data.json` ファイルを作成（`data.example.json` を参照）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **サービスを実行**
```bash
go run . --api-key your-api-key-here
```

## ⚙️ 基本設定

Warden は、コマンドライン引数、環境変数、設定ファイルによる設定をサポートしています。以下は最も重要な設定です：

| 設定 | 環境変数 | 説明 | 必須 |
|------|---------|------|------|
| ポート | `PORT` | HTTP サーバーポート（デフォルト: 8081） | いいえ |
| API キー | `API_KEY` | API 認証キー（本番環境で推奨） | 推奨 |
| Redis | `REDIS` | キャッシュと分散ロック用の Redis アドレス（例: `localhost:6379`） | オプション |
| データファイル | - | ローカルデータファイルのパス（デフォルト: `data.json`） | はい* |
| リモート設定 | `CONFIG` | データ取得用のリモート API URL | オプション |

\* リモート API を使用しない場合は必須

完全な設定オプションについては、[設定ドキュメント](docs/enUS/CONFIGURATION.md)を参照してください。

## 📡 API の使用

Warden は、ユーザーリストのクエリ、ページネーション、ヘルスチェック用の RESTful API を提供します。サービスは、クエリパラメータ `?lang=xx` または `Accept-Language` ヘッダーによる多言語応答をサポートします。

**例**:
```bash
# ユーザーをクエリ
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# ヘルスチェック
curl "http://localhost:8081/health"
```

完全な API ドキュメントについては、[API ドキュメント](docs/enUS/API.md)または[OpenAPI 仕様](openapi.yaml)を参照してください。

## 📊 パフォーマンス

wrk ストレステストに基づく（30秒、16スレッド、100接続）：
- **リクエスト/秒**: 5038.81
- **平均レイテンシ**: 21.30ms
- **最大レイテンシ**: 226.09ms

## 📚 ドキュメント

### 主要ドキュメント

- **[アーキテクチャ](docs/enUS/ARCHITECTURE.md)** - 技術アーキテクチャと設計決定
- **[API リファレンス](docs/enUS/API.md)** - 完全な API エンドポイントドキュメント
- **[設定](docs/enUS/CONFIGURATION.md)** - 設定リファレンスと例
- **[デプロイ](docs/enUS/DEPLOYMENT.md)** - デプロイガイド（Docker、Kubernetes など）

### 追加リソース

- **[開発ガイド](docs/enUS/DEVELOPMENT.md)** - 開発環境のセットアップと貢献ガイド
- **[セキュリティ](docs/enUS/SECURITY.md)** - セキュリティ機能とベストプラクティス
- **[SDK](docs/enUS/SDK.md)** - Go SDK の使用ドキュメント
- **[例](example/README.md)** - クイックスタート例（基本と高度）

## 📄 ライセンス

詳細については、[LICENSE](LICENSE) ファイルを参照してください。

## 🤝 貢献

Issues と Pull Request の提出を歓迎します！ガイドラインについては、[CONTRIBUTING.md](docs/enUS/CONTRIBUTING.md)を参照してください。
