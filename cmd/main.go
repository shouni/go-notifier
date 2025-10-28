package main

import (
	"context"
	"log"
	"os"
	"time"

	// internal/notifier パッケージを参照
	"go_notifier/pkg/notifier"
	// 外部パッケージとして参照
	"github.com/shouni/go-web-exact/pkg/httpclient"
)

const (
	// 環境変数のキー
	envSlackWebhook = "SLACK_WEBHOOK_URL"
	envBacklogBase  = "BACKLOG_BASE_URL"
	envBacklogKey   = "BACKLOG_API_KEY"

	// Backlog 環境依存のデフォルト設定 (実際の環境に合わせて変更が必要です)
	defaultIssueTypeID = 101 // 例: タスクの課題種別ID
	defaultPriorityID  = 3   // 例: 中の優先度ID
	targetProjectID    = 10  // 記事を投稿するBacklogのプロジェクトID

	// HTTPクライアントの設定 (すべての通信に適用される)
	defaultTimeout = 60 * time.Second
)

func main() {
	// 1. 環境設定と初期化
	ctx := context.Background()
	log.Println("Go Notifier 処理を開始します...")

	// 2. ターゲットURL（例として固定のURLを使用）
	targetURL := "https://github.com/shouni/go-web-exact/blob/main/README.md" // 例: 解析可能なURLを設定

	// 3. 共通の httpclient インスタンスの作成 (DIの基盤)
	// リトライとタイムアウト設定はここで一元管理される
	sharedClient := httpclient.New(defaultTimeout)

	// 4. Notifier インスタンスの作成

	// Slack Notifier の設定
	slackURL := os.Getenv(envSlackWebhook)
	if slackURL == "" {
		log.Println("警告: SLACK_WEBHOOK_URL が設定されていません。Slack通知はスキップされます。")
	}
	slackNotifier := notifier.NewSlackNotifier(sharedClient, slackURL) // 共有クライアントを注入

	// Backlog Notifier の設定
	backlogBase := os.Getenv(envBacklogBase)
	backlogKey := os.Getenv(envBacklogKey)
	if backlogBase == "" || backlogKey == "" {
		log.Println("警告: Backlog設定 (URL/KEY) が不足しています。Backlog投稿はスキップされます。")
	}
	backlogNotifier := notifier.NewBacklogNotifier(
		sharedClient, // 共有クライアントを注入
		backlogBase,
		backlogKey,
		defaultIssueTypeID,
		defaultPriorityID,
	)

	// 5. 登録する Notifier のリストアップ
	var notifiers []notifier.Notifier
	if slackURL != "" {
		notifiers = append(notifiers, slackNotifier)
	}
	if backlogBase != "" && backlogKey != "" {
		notifiers = append(notifiers, backlogNotifier)
	}

	if len(notifiers) == 0 {
		log.Fatal("🚨 致命的なエラー: 有効な通知先が一つも設定されていません。")
	}

	// 6. ContentNotifier の初期化 (Web抽出ロジックの統合)
	// ContentNotifier の内部で sharedClient が web.Extractor に渡される
	contentNotifier := notifier.NewContentNotifier(defaultTimeout, notifiers...)

	// 7. 通知処理の実行
	log.Printf("URL: %s のコンテンツを抽出・通知します。\n", targetURL)
	log.Printf("Backlog プロジェクトID: %d に課題として投稿されます。\n", targetProjectID)

	// ContentNotifier を使って、Webコンテンツを抽出し、登録されたすべての Notifier に通知を送信
	if err := contentNotifier.NotifyFromURL(ctx, targetURL, targetProjectID); err != nil {
		// ContentNotifier は内部でエラーをログに出力しているため、ここでは致命的なエラーとして扱う
		log.Fatalf("🚨 致命的なエラー: 通知処理全体が失敗しました: %v", err)
	}

	log.Println("✅ Go Notifier 処理が正常に完了しました。")
}
