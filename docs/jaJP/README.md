# ドキュメントインデックス

Warden AllowList ユーザーデータサービスのドキュメントへようこそ。

## 🌐 多言語ドキュメント

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 ドキュメントリスト

### コアドキュメント

- **[README.md](../../README.jaJP.md)** - プロジェクト概要とクイックスタートガイド
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 技術アーキテクチャと設計決定

### 詳細ドキュメント

- **[API.md](API.md)** - 完全な API エンドポイントドキュメント
  - ユーザーリストクエリエンドポイント
  - ページネーション機能
  - ヘルスチェックエンドポイント
  - エラーレスポンス形式

- **[CONFIGURATION.md](CONFIGURATION.md)** - 設定リファレンス
  - 設定方法
  - 必須設定項目
  - オプション設定項目
  - データマージ戦略
  - 設定例
  - 設定のベストプラクティス

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - デプロイメントガイド
  - Docker デプロイメント（GHCR イメージを含む）
  - Docker Compose デプロイメント
  - ローカルデプロイメント
  - 本番環境デプロイメント
  - Kubernetes デプロイメント
  - パフォーマンス最適化

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - 開発ガイド
  - 開発環境のセットアップ
  - コード構造の説明
  - テストガイド
  - 貢献ガイド

- **[SDK.md](SDK.md)** - SDK 使用ドキュメント
  - Go SDK のインストールと使用
  - API インターフェースの説明
  - サンプルコード

- **[SECURITY.md](SECURITY.md)** - セキュリティドキュメント
  - セキュリティ機能
  - セキュリティ設定
  - ベストプラクティス

- **[CODE_STYLE.md](CODE_STYLE.md)** - コードスタイルガイド
  - コード標準
  - 命名規則
  - ベストプラクティス

## 🌐 多言語サポート

Warden は完全な国際化（i18N）機能をサポートしています。すべての API レスポンス、エラーメッセージ、ログが国際化をサポートしています。

### サポートされている言語

- 🇺🇸 英語 (en) - デフォルト言語
- 🇨🇳 中国語 (zh)
- 🇫🇷 フランス語 (fr)
- 🇮🇹 イタリア語 (it)
- 🇯🇵 日本語 (ja)
- 🇩🇪 ドイツ語 (de)
- 🇰🇷 韓国語 (ko)

### 言語検出

Warden は次の優先順位で 2 つの言語検出方法をサポートしています：

1. **クエリパラメータ**: URL クエリパラメータ `?lang=ja` で言語を指定
2. **Accept-Language ヘッダー**: ブラウザまたはクライアントの言語設定を自動検出
3. **デフォルト言語**: 指定されていない場合は英語

### 使用例

#### クエリパラメータで言語を指定

```bash
# 日本語を使用
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# 中国語を使用
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=zh"

# フランス語を使用
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

#### Accept-Language ヘッダーで自動検出

```bash
# ブラウザが自動的に Accept-Language ヘッダーを送信
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: ja-JP,ja;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### 国際化の範囲

次のコンテンツが複数の言語をサポートしています：

- ✅ API エラーレスポンスメッセージ
- ✅ HTTP ステータスコードエラーメッセージ
- ✅ ログメッセージ（リクエストコンテキストに基づく）
- ✅ 設定と警告メッセージ

### 技術実装

- リクエストコンテキストを使用して言語情報を保存し、グローバル状態を回避
- スレッドセーフな言語切り替えをサポート
- 英語への自動フォールバック（翻訳が見つからない場合）
- すべての翻訳はコードに組み込まれており、外部ファイルは不要

### 開発ノート

新しい翻訳を追加したり、既存の翻訳を変更したりするには、`internal/i18n/i18n.go` ファイルの `translations` マップを編集してください。

## 🚀 クイックナビゲーション

### はじめに

1. [README.jaJP.md](../../README.jaJP.md) を読んでプロジェクトを理解する
2. [クイックスタート](../../README.jaJP.md#クイックスタート) セクションを確認する
3. [設定](../../README.jaJP.md#設定) を参照してサービスを設定する

### 開発者

1. [ARCHITECTURE.md](ARCHITECTURE.md) を読んでアーキテクチャを理解する
2. [API.md](API.md) を確認して API インターフェースを理解する
3. [開発ガイド](../../README.jaJP.md#開発ガイド) を参照して開発する

### 運用

1. [DEPLOYMENT.md](DEPLOYMENT.md) を読んでデプロイメント方法を理解する
2. [CONFIGURATION.md](CONFIGURATION.md) を確認して設定オプションを理解する
3. [パフォーマンス最適化](DEPLOYMENT.md#パフォーマンス最適化) を参照してサービスを最適化する

## 📖 ドキュメント構造

```
warden/
├── README.md              # プロジェクトメインドキュメント（日本語）
├── README.jaJP.md         # プロジェクトメインドキュメント（日本語）
├── docs/
│   ├── enUS/
│   │   ├── README.md       # ドキュメントインデックス（英語）
│   │   ├── ARCHITECTURE.md # アーキテクチャドキュメント（英語）
│   │   ├── API.md          # API ドキュメント（英語）
│   │   ├── CONFIGURATION.md # 設定リファレンス（英語）
│   │   ├── DEPLOYMENT.md   # デプロイメントガイド（英語）
│   │   ├── DEVELOPMENT.md  # 開発ガイド（英語）
│   │   ├── SDK.md          # SDK ドキュメント（英語）
│   │   ├── SECURITY.md     # セキュリティドキュメント（英語）
│   │   └── CODE_STYLE.md   # コードスタイル（英語）
│   └── jaJP/
│       ├── README.md       # ドキュメントインデックス（日本語、このファイル）
│       ├── ARCHITECTURE.md # アーキテクチャドキュメント（日本語）
│       ├── API.md          # API ドキュメント（日本語）
│       ├── CONFIGURATION.md # 設定リファレンス（日本語）
│       ├── DEPLOYMENT.md   # デプロイメントガイド（日本語）
│       ├── DEVELOPMENT.md  # 開発ガイド（日本語）
│       ├── SDK.md          # SDK ドキュメント（日本語）
│       ├── SECURITY.md     # セキュリティドキュメント（日本語）
│       └── CODE_STYLE.md   # コードスタイル（日本語）
└── ...
```

## 🔍 トピック別検索

### 設定関連

- 環境変数設定: [CONFIGURATION.md](CONFIGURATION.md)
- データマージ戦略: [CONFIGURATION.md](CONFIGURATION.md)
- 設定例: [CONFIGURATION.md](CONFIGURATION.md)

### API 関連

- API エンドポイントリスト: [API.md](API.md)
- エラーハンドリング: [API.md](API.md)
- ページネーション機能: [API.md](API.md)

### デプロイメント関連

- Docker デプロイメント: [DEPLOYMENT.md#docker-デプロイメント](DEPLOYMENT.md#docker-デプロイメント)
- GHCR イメージ: [DEPLOYMENT.md#事前構築済みイメージの使用推奨](DEPLOYMENT.md#事前構築済みイメージの使用推奨)
- 本番環境: [DEPLOYMENT.md#本番環境デプロイメント推奨事項](DEPLOYMENT.md#本番環境デプロイメント推奨事項)
- Kubernetes: [DEPLOYMENT.md#kubernetes-デプロイメント](DEPLOYMENT.md#kubernetes-デプロイメント)

### アーキテクチャ関連

- 技術スタック: [ARCHITECTURE.md](ARCHITECTURE.md)
- プロジェクト構造: [ARCHITECTURE.md](ARCHITECTURE.md)
- コアコンポーネント: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 使用推奨事項

1. **初めてのユーザー**: [README.jaJP.md](../../README.jaJP.md) から始めて、クイックスタートガイドに従う
2. **サービスを設定**: [CONFIGURATION.md](CONFIGURATION.md) を参照してすべての設定オプションを理解する
3. **サービスをデプロイ**: [DEPLOYMENT.md](DEPLOYMENT.md) を確認してデプロイメント方法を理解する
4. **拡張機能を開発**: [ARCHITECTURE.md](ARCHITECTURE.md) を読んでアーキテクチャ設計を理解する
5. **SDK を統合**: [SDK.md](SDK.md) を参照して SDK の使用方法を学ぶ

## 📝 ドキュメント更新

ドキュメントはプロジェクトの進化に伴って継続的に更新されます。エラーを見つけたり、追加が必要な場合は、Issue または Pull Request を提出してください。

## 🤝 貢献

ドキュメントの改善を歓迎します：

1. エラーや改善が必要な領域を見つける
2. 問題を説明する Issue を提出する
3. または直接 Pull Request を提出する
