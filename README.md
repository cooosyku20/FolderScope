# FolderScope 🔍

FolderScopeは、ディレクトリ構造を分析し、詳細なレポートを生成するGUIツールです。

## 機能 ✨

- 📁 GUIによる直感的なディレクトリ選択
- 📊 ファイルシステム構造の詳細な分析
- 📝 分析結果の構造化レポート生成
- 🔍 バイナリファイルの自動検出
- 📋 JSONフォーマットでのログ出力

## インストール 🚀

```bash
go install github.com/yourusername/FolderScope/cmd/folderscope@latest
```

## 使用方法 💡

1. アプリケーションを起動します：
```bash
folderscope
```

2. GUIダイアログが表示され、以下を選択します：
   - 調査対象のディレクトリ
   - レポート出力先のディレクトリ

3. 選択完了後、自動的に分析が開始され、指定した出力先にレポートが生成されます。

## アーキテクチャ 🏗

FolderScopeは、クリーンアーキテクチャの原則に従って設計されています：

- `cmd/folderscope/`: メインアプリケーションのエントリーポイント
- `internal/`: 内部パッケージ
  - `domain/`: ドメインモデルとビジネスロジック
  - `usecase/`: アプリケーションのユースケース
  - `infrastructure/`: 外部依存（ファイルシステム、ロギングなど）
  - `gui/`: グラフィカルユーザーインターフェース

## 開発環境のセットアップ 🛠

### 前提条件

- Go 1.21以上
- Fyne GUI toolkit

### セットアップ手順

1. リポジトリのクローン：
```bash
git clone https://github.com/yourusername/FolderScope.git
cd FolderScope
```

2. 依存関係のインストール：
```bash
go mod download
```

3. ビルド：
```bash
go build ./cmd/folderscope
```

## ライセンス 📄

このプロジェクトはMITライセンスの下で公開されています。詳細は[LICENSE](LICENSE)ファイルをご覧ください。

## 貢献について 🤝

バグ報告や機能リクエストは、GitHubのIssueでお願いします。
プルリクエストも歓迎します。

## 作者 👤

- [littleironwaltz](https://github.com/littleironwaltz)

---

**注意**: このプロジェクトは開発中であり、APIは変更される可能性があります。
