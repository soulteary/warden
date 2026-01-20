# Warden

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

ローカルおよびリモート設定ソースからのデータ同期とマージをサポートする高性能な許可リスト（AllowList）ユーザーデータサービス。

![Warden](.github/assets/banner.jpg)

> **Warden**（看守者）—— スターゲートの看守者であり、誰が通過でき、誰が拒否されるかを決定します。スターゲートの看守者がスターゲートを守るように、Warden はあなたの許可リストを守り、承認されたユーザーのみが通過できるようにします。

## 📋 プロジェクト概要

Warden は、Go で開発された軽量な HTTP API サービスで、主に許可リストユーザーデータ（電話番号とメールアドレス）の提供と管理に使用されます。このサービスは、ローカル設定ファイルとリモート API からのデータ取得をサポートし、リアルタイムのパフォーマンスと信頼性を確保するための複数のデータマージ戦略を提供します。

## ✨ 主要機能

- 🚀 **高性能**: 平均レイテンシ 21ms で毎秒 5000 以上のリクエストをサポート
- 🔄 **複数のデータソース**: ローカル設定ファイルとリモート API の両方をサポート
- 🎯 **柔軟な戦略**: 6 つのデータマージモード（リモート優先、ローカル優先、リモートのみ、ローカルのみなど）を提供
- ⏰ **スケジュール更新**: Redis 分散ロックベースのスケジュールタスクによる自動データ同期
- 📦 **コンテナ化デプロイ**: 完全な Docker サポート、すぐに使用可能
- 📊 **構造化ログ**: zerolog を使用して詳細なアクセスログとエラーログを提供
- 🔒 **分散ロック**: Redis を使用して、分散環境でスケジュールタスクが繰り返し実行されないようにします

## 🏗️ アーキテクチャ設計

Warden は、HTTP 層、ビジネス層、インフラストラクチャ層を含む階層型アーキテクチャ設計を使用します。システムは、複数のデータソース、マルチレベルキャッシュ、分散ロックメカニズムをサポートします。

詳細なアーキテクチャドキュメントについては、以下を参照してください: [アーキテクチャ設計ドキュメント](docs/enUS/ARCHITECTURE.md)

## 📦 インストールと実行

> 💡 **クイックスタート**: Warden をすぐに体験したいですか？[クイックスタート例](example/README.en.md)をご覧ください:
> - [簡単な例](example/basic/README.en.md) - 基本的な使用、ローカルデータファイルのみ
> - [高度な例](example/advanced/README.en.md) - 完全な機能、リモート API と Mock サービスを含む

### 前提条件

- Go 1.25+ ([go.mod](go.mod) を参照)
- Redis (分散ロックとキャッシュ用)
- Docker (オプション、コンテナ化デプロイ用)

### クイックスタート

1. **プロジェクトをクローン**
```bash
git clone <repository-url>
cd warden
```

2. **依存関係をインストール**
```bash
go mod download
```

3. **ローカルデータファイルを設定**
`data.json` ファイルを作成（`data.example.json` を参照）:
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **サービスを実行**
```bash
go run main.go
```

詳細な設定とデプロイの手順については、以下を参照してください:
- [設定ドキュメント](docs/enUS/CONFIGURATION.md) - すべての設定オプションを学ぶ
- [デプロイドキュメント](docs/enUS/DEPLOYMENT.md) - デプロイ方法を学ぶ

## ⚙️ 設定

Warden は、コマンドライン引数、環境変数、設定ファイルなど、複数の設定方法をサポートします。システムは、柔軟な設定戦略を持つ 6 つのデータマージモードを提供します。

詳細な設定ドキュメントについては、以下を参照してください: [設定ドキュメント](docs/enUS/CONFIGURATION.md)

## 📡 API ドキュメント

Warden は、ユーザーリストクエリ、ページネーション、ヘルスチェックなどをサポートする完全な RESTful API を提供します。プロジェクトは、OpenAPI 3.0 仕様ドキュメントも提供します。

詳細な API ドキュメントについては、以下を参照してください: [API ドキュメント](docs/enUS/API.md)

OpenAPI 仕様ファイル: [openapi.yaml](openapi.yaml)

## 🔌 SDK の使用

Warden は、他のプロジェクトでの統合を容易にする Go SDK を提供します。SDK は、キャッシュ、認証などの機能をサポートするシンプルな API インターフェースを提供します。

詳細な SDK ドキュメントについては、以下を参照してください: [SDK ドキュメント](docs/enUS/SDK.md)

## 🐳 Docker デプロイ

Warden は、完全な Docker と Docker Compose デプロイをサポートし、すぐに使用できます。

### プリビルドイメージでクイックスタート（推奨）

GitHub Container Registry (GHCR) が提供するプリビルドイメージを使用して、ローカルビルドなしで迅速に開始:

```bash
# 最新バージョンのイメージをプル
docker pull ghcr.io/soulteary/warden:latest

# コンテナを実行（基本例）
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **ヒント**: プリビルドイメージを使用すると、ローカルビルド環境なしで迅速に開始できます。イメージは自動的に更新され、最新バージョンを使用していることを確認します。

### Docker Compose の使用

> 🚀 **クイックデプロイ**: 完全な Docker Compose 設定例については、[例ディレクトリ](example/README.en.md) を確認してください

詳細なデプロイドキュメントについては、以下を参照してください: [デプロイドキュメント](docs/enUS/DEPLOYMENT.md)

## 📊 パフォーマンス指標

wrk 負荷テスト結果に基づく（30 秒テスト、16 スレッド、100 接続）:

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
平均レイテンシ: 21.30ms
最大レイテンシ: 226.09ms
```

## 📁 プロジェクト構造

```
warden/
├── main.go                 # プログラムエントリーポイント
├── data.example.json      # ローカルデータファイルの例
├── config.example.yaml    # 設定ファイルの例
├── openapi.yaml           # OpenAPI 仕様ファイル
├── go.mod                 # Go モジュール定義
├── docker-compose.yml     # Docker Compose 設定
├── LICENSE                # ライセンスファイル
├── README.*.md            # 多言語プロジェクトドキュメント（中国語/英語/フランス語/イタリア語/日本語/ドイツ語/韓国語）
├── CONTRIBUTING.*.md      # 多言語貢献ガイド
├── docker/
│   └── Dockerfile         # Docker イメージビルドファイル
├── docs/                  # ドキュメントディレクトリ（多言語）
│   ├── enUS/              # 英語ドキュメント
│   └── zhCN/              # 中国語ドキュメント
├── example/               # クイックスタート例
│   ├── basic/             # 簡単な例（ローカルファイルのみ）
│   └── advanced/          # 高度な例（完全な機能、Mock API を含む）
├── internal/
│   ├── cache/             # Redis キャッシュとロック実装
│   ├── cmd/               # コマンドライン引数解析
│   ├── config/            # 設定管理
│   ├── define/            # 定数定義とデータ構造
│   ├── di/                # 依存性注入
│   ├── errors/            # エラー処理
│   ├── logger/            # ログ初期化
│   ├── metrics/           # メトリクス収集
│   ├── middleware/        # HTTP ミドルウェア
│   ├── parser/            # データパーサー（ローカル/リモート）
│   ├── router/            # HTTP ルーティング処理
│   ├── validator/         # バリデーター
│   └── version/           # バージョン情報
├── pkg/
│   ├── gocron/            # スケジュールタスクスケジューラー
│   └── warden/            # Warden SDK
├── scripts/               # スクリプトディレクトリ
└── .github/               # GitHub 設定（CI/CD、Issue/PR テンプレートなど）
```

## 🔒 セキュリティ機能

Warden は、API 認証、SSRF 保護、レート制限、TLS 検証などの複数のセキュリティ機能を実装しています。

詳細なセキュリティドキュメントについては、以下を参照してください: [セキュリティドキュメント](docs/enUS/SECURITY.md)

## 🔧 開発ガイド

> 📚 **参照例**: さまざまな使用シナリオの完全な例コードと設定については、[例ディレクトリ](example/README.en.md) を確認してください。

詳細な開発ドキュメントについては、以下を参照してください: [開発ドキュメント](docs/enUS/DEVELOPMENT.md)

### コード標準

プロジェクトは、Go の公式コード標準とベストプラクティスに従います。詳細な標準については、以下を参照してください:

- [CODE_STYLE.md](docs/enUS/CODE_STYLE.md) - コードスタイルガイド
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - 貢献ガイド

## 📄 ライセンス

詳細については、[LICENSE](LICENSE) ファイルを参照してください。

## 🤝 貢献

Issues と Pull Request の提出を歓迎します！

## 📞 連絡先

質問や提案については、Issues を通じてお問い合わせください。

---

**バージョン**: プログラムは起動時にバージョン、ビルド時間、コードバージョンを表示します（`warden --version` または起動ログを介して）
