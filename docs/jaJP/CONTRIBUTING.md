# 貢献ガイド

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Warden プロジェクトへのご関心ありがとうございます！あらゆる形式の貢献を歓迎します。


## 📋 目次

- [貢献方法](#貢献方法)
- [開発環境のセットアップ](#開発環境のセットアップ)
- [コード規約](#コード規約)
- [コミット規約](#コミット規約)
- [Pull Request プロセス](#pull-request-プロセス)
- [バグ報告と機能リクエスト](#バグ報告と機能リクエスト)

## 🚀 貢献方法

以下の方法で貢献できます：

- **バグの報告**: GitHub Issues で問題を報告
- **機能の提案**: GitHub Issues で新機能のアイデアを提案
- **コードの提出**: Pull Request を通じてコードの改善を提出
- **ドキュメントの改善**: プロジェクトのドキュメントの改善を支援
- **質問への回答**: Issues で他のユーザーを支援

このプロジェクトに参加する際は、すべての貢献者を尊重し、建設的な批判を受け入れ、プロジェクトにとって最善のことに焦点を当ててください。

## 🛠️ 開発環境のセットアップ

### 前提条件

- Go 1.25 以上
- Redis（テスト用）
- Git

### クイックスタート

```bash
# 1. プロジェクトをフォークしてクローン
git clone https://github.com/your-username/warden.git
cd warden

# 2. 上流リポジトリを追加
git remote add upstream https://github.com/soulteary/warden.git

# 3. 依存関係をインストール
go mod download

# 4. テストを実行
go test ./...

# 5. ローカルサービスを起動（Redis が実行されていることを確認）
go run .
```

## 📝 コード規約

以下のコード規約に従ってください：

1. **Go 公式コード規約に従う**: [Effective Go](https://go.dev/doc/effective_go)
2. **コードのフォーマット**: `go fmt ./...` を実行
3. **コードチェック**: `golangci-lint` または `go vet ./...` を使用
4. **テストの記述**: 新機能にはテストを含める必要があります
5. **コメントの追加**: 公開関数と型にはドキュメントコメントが必要です
6. **定数の命名**: すべての定数は `ALL_CAPS` (UPPER_SNAKE_CASE) 命名スタイルを使用する必要があります

詳細なコードスタイルガイドについては、[CODE_STYLE.md](CODE_STYLE.md) を参照してください。

## 📦 コミット規約

### コミットメッセージの形式

[Conventional Commits](https://www.conventionalcommits.org/) 規約を使用します：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### タイプ

- `feat`: 新機能
- `fix`: バグ修正
- `docs`: ドキュメント更新
- `style`: コードフォーマット調整（コードの実行に影響しない）
- `refactor`: コードリファクタリング
- `perf`: パフォーマンス最適化
- `test`: テスト関連
- `chore`: ビルドプロセスまたは補助ツールの変更

## 🔄 Pull Request プロセス

### Pull Request の作成

```bash
# 1. 機能ブランチを作成
git checkout -b feature/your-feature-name

# 2. 変更を加えてコミット
git add .
git commit -m "feat: 新機能を追加"

# 3. 上流コードを同期
git fetch upstream
git rebase upstream/main

# 4. ブランチをプッシュして PR を作成
git push origin feature/your-feature-name
```

### Pull Request チェックリスト

Pull Request を提出する前に、以下を確認してください：

- [ ] コードがプロジェクトのコード規約に従っている
- [ ] すべてのテストが通過する（`go test ./...`）
- [ ] コードがフォーマットされている（`go fmt ./...`）
- [ ] 必要なテストが追加されている
- [ ] 関連ドキュメントが更新されている
- [ ] コミットメッセージが[コミット規約](#コミット規約)に従っている
- [ ] コードが lint チェックを通過している

すべての Pull Request にはコードレビューが必要です。レビューコメントには迅速に対応してください。

## 🐛 バグ報告と機能リクエスト

Issue を作成する前に、既存の Issues を検索して、問題や機能が報告されていないことを確認してください。

## 🎯 始める

貢献したいがどこから始めればよいかわからない場合は、以下に焦点を当ててください：

- `good first issue` とラベル付けされた Issues
- `help wanted` とラベル付けされた Issues
- コード内の `TODO` コメント
- ドキュメントの改善（誤字の修正、明確さの向上、例の追加）

質問がある場合は、既存の Issues と Pull Requests を確認するか、関連する Issue で質問してください。

---

Warden プロジェクトへの貢献ありがとうございます！🎉
