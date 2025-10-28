# 🔔 Go Notifier

Go Notifier は、Web コンテンツを自動で抽出・整形し、複数のチャネル（Slack, Backlog）に堅牢に通知・投稿するための Go 言語製アプリケーションです。

本プロジェクトは、指数バックオフによるリトライ処理と堅牢な HTTP クライアント（`go-web-exact`）を基盤として利用しています。

---

## 🚀 セットアップと実行

### 1. 依存関係の解決

プロジェクトルートで以下のコマンドを実行し、Go Modules を初期化し、外部依存パッケージをダウンロードします。

```bash
# プロジェクトをGo Moduleとして初期化
go mod init go_notifier 

# 外部依存パッケージのダウンロード
go go get github.com/shouni/go-web-exact/...
````

### 2\. 環境変数の設定

`cmd/main.go` の実行には、以下の環境変数の設定が必要です。

| 環境変数名 | 役割 | 必須/任意 | 例 |
| :--- | :--- | :--- | :--- |
| **SLACK\_WEBHOOK\_URL** | Slack への通知用 Webhook URL | 任意 | `https://hooks.slack.com/services/TXXXX/...` |
| **BACKLOG\_BASE\_URL** | Backlog API のベース URL | 任意 | `https://[space_id].backlog.com/api/v2` |
| **BACKLOG\_API\_KEY** | Backlog への投稿に使用する API キー | 任意 | `xxxxxxxxxxxxxxxxxxxxxxxx` |

**💡 注意:** Slack/Backlog のいずれか、または両方の環境変数が設定されていない場合、その通知先への投稿は自動的にスキップされます。両方とも設定されていない場合は実行時にエラーとなります。

### 3\. 実行

以下のコマンドでプログラムを実行します。

```bash
go run cmd/main.go
```

-----

## ⚙️ 設定（Backlog 固有のパラメーター）

Backlog への課題投稿には、環境に依存する ID が必要です。これらの値は、`cmd/main.go` 内の定数としてハードコードされています。環境に合わせて変更してください。

| 定数名 | 役割 | `cmd/main.go` の該当行 |
| :--- | :--- | :--- |
| **`targetProjectID`** | **必須:** 記事を課題として登録する **プロジェクト ID**。 | `const targetProjectID = 10` |
| **`defaultIssueTypeID`** | 新規課題のデフォルトの **課題種別 ID**。 | `const defaultIssueTypeID = 101` |
| **`defaultPriorityID`** | 新規課題のデフォルトの **優先度 ID**。 | `const defaultPriorityID = 3` |

-----

## 📐 プロジェクト構成

コアとなる通知ロジックは、責務の分離のため `pkg/notifier` パッケージ内に配置されています。

```
go_notifier/
├── cmd/
│   └── main.go       # プログラムのエントリーポイント
└── pkg/
    └── notifier/     # コア通知ロジック (Notifier インターフェース実装)
        ├── slack.go  # Slack 通知クライアント
        ├── backlog.go # Backlog 投稿クライアント
        └── client.go # Notifier インターフェースおよび Web 抽出統合ロジック
```

### 外部依存パッケージ

本プロジェクトは、以下のリポジトリのパッケージを基盤として利用しています。

* **`github.com/shouni/go-web-exact`**: 堅牢な HTTP 通信（リトライ/エラー判定）および Web コンテンツ抽出機能を提供。

-----

## 📚 処理フロー

1.  `cmd/main.go` が実行され、`ContentNotifier` を初期化。
2.  `ContentNotifier` が、`targetURL` の Web ページを **`web` パッケージ** を使って取得・解析。
3.  取得した HTML からメインコンテンツを抽出し、Markdown 風のテキストに整形。
4.  整形されたテキストを、登録されているすべての **`Notifier`**（Slack/Backlog）に送信。
5.  通信は **`httpclient`** を通じて行われ、ネットワークエラーや 5xx エラーが発生した場合は自動的に **指数バックオフ** でリトライされます。
