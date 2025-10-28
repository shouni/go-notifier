# 🔔 Go Notifier

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-notifier)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-notifier)](https://github.com/shouni/go-notifier/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go Notifier は、Web コンテンツを自動で抽出・整形し、複数のチャネル（Slack, Backlog）に**堅牢**に通知・投稿するための Go 言語製 CLI アプリケーションです。

**主要な機能強化点:**

* **堅牢性**: 指数バックオフによるリトライ処理を備えた HTTP クライアントを使用。
* **セキュリティ**: Backlog APIキーを URL クエリから **HTTPヘッダー** に移動。
* **表現力**: Slack への通知は **Block Kit** 形式に対応。
* **柔軟性**: タイムアウト設定、Backlog課題種別IDなどを **CLIフラグ** から指定可能。

-----

## 🚀 セットアップと実行

### 1\. ビルド

プロジェクトルートで以下のコマンドを実行し、実行ファイルを生成します。

```bash
go build -o bin/go_notifier ./cmd
```

### 2\. 環境変数の設定

本アプリケーションは、Notifier ごとに以下の環境変数に依存します。

| 環境変数名 | 役割 | 必須/任意 | 例 |
| :--- | :--- | :--- | :--- |
| **SLACK\_WEBHOOK\_URL** | Slack への通知用 Webhook URL | `slack` コマンドで必須 | `https://hooks.slack.com/services/TXXXX/...` |
| **BACKLOG\_BASE\_URL** | Backlog スペースのベース URL (APIパスは内部で付与) | `backlog` コマンドで必須 | `https://[space_id].backlog.jp` |
| **BACKLOG\_API\_KEY** | Backlog への投稿に使用する API キー | `backlog` コマンドで必須 | `xxxxxxxxxxxxxxxxxxxxxxxx` |

### 3\. 実行（CLIコマンド）

ビルドした実行ファイル (`bin/go_notifier`) を使用し、サブコマンドとフラグで操作します。

#### 🔹 Slack への投稿

```bash
# 環境変数 SLACK_WEBHOOK_URL が必要
./bin/go_notifier slack --message "これはSlackに投稿するメッセージです。" \
  --username "Notifier Bot" \
  --icon-emoji ":bell:"
```

#### 🔹 Backlog への課題登録

課題登録に必要な ID は CLI フラグで指定します。

```bash
# 環境変数 BACKLOG_BASE_URL と BACKLOG_API_KEY が必要
# 複数行メッセージの場合、最初の行がサマリーになります。
./bin/go_notifier backlog \
  --message "新規課題のサマリー\nこれは課題の説明文です。" \
  --project-id 10 \
  --issue-type-id 101 \
  --priority-id 3
```

| フラグ名 | 役割 | デフォルト値 |
| :--- | :--- | :--- |
| `--project-id` | **必須**: 課題を登録する **プロジェクト ID**。 | (環境変数から取得) |
| `--issue-type-id` | 新規課題の **課題種別 ID**。 | `101` (タスク) |
| `--priority-id` | 新規課題の **優先度 ID**。 | `3` (中) |

-----

## 📐 プロジェクト構成

Cobra CLI と DI の原則に基づき、責務が明確に分離されています。

```
go_notifier/
├── cmd/
│   ├── root.go       # グローバルなフラグ定義とエントリーポイント (Cobra)
│   ├── slack.go      # Slack サブコマンドのロジック
│   └── backlog.go    # Backlog サブコマンドのロジック
└── pkg/
    └── notifier/     # コア通知ロジック (Notifier インターフェース実装)
        ├── slack.go  # Slack 通知クライアント (Block Kit対応)
        ├── backlog.go # Backlog 投稿クライアント (ヘッダー認証対応)
        ├── client.go # ContentNotifier (Web抽出と通知の統合)
        └── interface.go # Notifier インターフェース定義
```

### 外部依存パッケージ

本プロジェクトは、以下の主要な外部パッケージに依存しています。

* **`github.com/shouni/go-web-exact`**: 堅牢な HTTP クライアント（リトライ/タイムアウト）および Web コンテンツ抽出機能を提供。
* **`github.com/slack-go/slack`**: Slack Block Kit 形式のメッセージ構築をサポート。
* **`github.com/spf13/cobra`**: 堅牢な CLI インターフェースを提供。

-----

## 📚 処理フロー

1.  ユーザーが `go_notifier [subcommand] --message ...` を実行。
2.  `cmd/root.go` でグローバルな `httpclient.Client` がタイムアウト設定に基づいて初期化される。
3.  サブコマンド（例: `backlog`）のロジックが実行され、Notifier が初期化される。
4.  メッセージが `Notifier` の `SendText` または `SendIssue` メソッドに渡される。
5.  各 `Notifier` は、メッセージを整形（SlackはBlock Kit、Backlogは絵文字除去）し、APIリクエストを構築。
6.  APIリクエストは、**指数バックオフ** リトライロジックを持つ共有 **`httpclient`** を通じて実行される。
7.  Backlog の場合、APIキーはセキュリティのために HTTP **ヘッダー** で送信される。

