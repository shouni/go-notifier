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
* **柔軟性**: タイムアウト設定、Backlog課題種別IDなどを **CLIフラグ/ショートカット** から指定可能。
* **新機能**: **Backlogの既存課題へのコメント投稿** (`backlog comment`) に対応。
* **新機能**: Notifierインターフェースに**ヘッダー付きテキスト送信**機能を追加し、表現力を向上。

-----

## 🚀 セットアップと実行

### 1\. ビルド

プロジェクトルートで以下のコマンドを実行し、実行ファイルを生成します。

```bash
go build -o bin/notifier
```

### 2\. 環境変数の設定

本アプリケーションは、Notifier ごとに以下の環境変数に依存します。

| 環境変数名 | 役割 | 必須/任意 | 例 |
| :--- | :--- | :--- | :--- |
| **SLACK\_WEBHOOK\_URL** | Slack への通知用 Webhook URL | `slack` コマンドで必須 | `https://hooks.slack.com/services/TXXXX/...` |
| **BACKLOG\_SPACE\_URL** | Backlog スペースのベース URL (APIパスは内部で付与) | `backlog` コマンドで必須 | `https://[space_id].backlog.jp` |
| **BACKLOG\_API\_KEY** | Backlog への投稿に使用する API キー | `backlog` コマンドで必須 | `xxxxxxxxxxxxxxxxxxxxxxxx` |

### 3\. 実行（CLIコマンド）

ビルドした実行ファイル (`bin/notifier`) を使用し、サブコマンドとフラグで操作します。グローバルフラグとして、投稿メッセージ（`-m`, `--message`）、**投稿タイトル**（`-t`, `--title`）、タイムアウト時間（`--timeout`）が利用可能です。

#### 🔹 Slack への投稿

SlackNotifierは、内部でMarkdownをBlock Kitに変換します。

```bash
# 環境変数 SLACK_WEBHOOK_URL が必要
# ショートカット: -t (title), -m (message), -u (username), -e (icon-emoji), -c (channel)
./bin/notifier slack -t "Slack通知タイトル" \
  -m "これはSlackに投稿するメッセージです。" \
  -u "Notifier Bot" \
  -c "#general"
```

#### 🔹 Backlog への課題登録

**`-t` (タイトル)** が課題のサマリーに、**`-m` (メッセージ)** が課題の詳細になります。

**IDが省略された場合、プロジェクト設定から最初の課題種別ID/優先度IDが自動取得されます。**

```bash
# 環境変数 BACKLOG_SPACE_URL と BACKLOG_API_KEY が必要
# ショートカット: -p (project-id), -t (title), -m (message)
./bin/notifier backlog -t "新規課題のサマリー" \
 -m "これは課題の説明文です。" \
 -p "TEST"
```

#### 🔹 Backlog 既存課題へのコメント投稿

**`PostComment`** 機能を利用します。課題キーまたはIDをフラグで指定する必要があります。

```bash
# 課題キーを指定してコメントを投稿する例
# ショートカット: -i (issue-id), -m (message)
./bin/notifier backlog comment \
 -i "PROJECT-123" \
 -m "この課題に関する新しい情報を追記します。"
```

| フラグ名 | ショートカット | 役割 | デフォルト値 |
| :--- | :--- | :--- | :--- |
| **`--title`** | **`-t`** | **グローバル**: 投稿タイトル/課題サマリーとして使用。 | (なし) |
| **`--message`** | **`-m`** | **グローバル**: 投稿メッセージ/課題詳細として使用。 | (なし) |
| **`--timeout`** | (なし) | **グローバル**: HTTPリクエストのタイムアウト時間（秒）。 | 10 |
| **`--project-id`** | **`-p`** | **必須** (課題登録時): BacklogのプロジェクトID。 (ENV: `BACKLOG_PROJECT_ID`) | (なし) |
| **`--issue-id`** | **`-i`** | **必須** (コメント時): コメント対象の **課題キー** または **ID**。 | (なし) |
| **`--username`** | **`-u`** | **Slack**: 投稿時のユーザー名。 (ENV: `SLACK_USERNAME`) | (なし) |
| **`--icon-emoji`** | **`-e`** | **Slack**: 投稿時の絵文字アイコン。 (ENV: `SLACK_ICON_EMOJI`) | (なし) |
| **`--channel`** | **`-c`** | **Slack**: 投稿先のチャンネル。 (ENV: `SLACK_CHANNEL`) | (なし) |

-----

## 📐 プロジェクト構成

Cobra CLI と DI の原則に基づき、責務が明確に分離されています。

```
go-notifier/
├── cmd/
│   ├── root.go       # グローバルなフラグ定義とエントリーポイント (Cobra)
│   ├── slack.go      # Slack サブコマンドのロジック
│   └── backlog.go    # Backlog サブコマンドのロジック (課題登録/コメント投稿ロジック含む)
├── pkg/
│   └── notifier/     # コア通知ロジック (Notifier インターフェース実装)
│       ├── backlog.go    # Backlog 投稿/コメントクライアント
│       ├── client.go     # ContentNotifier (Web抽出と通知の統合)
│       ├── client_mock.go # MockNotifier (テスト用モック)
│       └── slack.go      # Slack 通知クライアント (Block Kit)
│   └── util/         # 汎用ヘルパー関数 (絵文字サニタイズなど)
└── main.go           # アプリケーションのエントリーポイント (Cobraコマンドの実行)
```

### 外部依存パッケージ

本プロジェクトは、以下の主要な外部パッケージに依存しています。

* **`github.com/shouni/go-web-exact`**: 堅牢な HTTP クライアント（リトライ/タイムアウト）および Web コンテンツ抽出機能を提供。
* **`github.com/slack-go/slack`**: Slack Block Kit 形式のメッセージ構築をサポート。
* **`github.com/forPelevin/gomoji`**: Backlog投稿時の絵文字サニタイズに使用。
* **`github.com/spf13/cobra`**: 堅牢な CLI インターフェースを提供。

-----

## 📚 処理フロー

1.  ユーザーが `notifier [subcommand] -t "タイトル" -m "メッセージ"` を実行。
2.  `cmd/root.go` でグローバルな `httpclient.Client` が初期化される。
3.  サブコマンド（例: `backlog`）が実行され、適切な `Notifier` が初期化される。
4.  メッセージとタイトルが `Notifier` の **`SendTextWithHeader`** や **`SendIssue`** メソッドに渡される。
5.  Backlog の場合、`SendIssue` は **プロジェクトIDと課題属性を自動で補完** する。
6.  APIリクエストは、**指数バックオフ** リトライロジックを持つ共有 **`httpclient`** を通じて実行される。
7.  APIキーはセキュリティのために HTTP **ヘッダー** で送信される。

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
