package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"go_notifier/pkg/notifier"

	"github.com/shouni/go-web-exact/pkg/httpclient"
)

const (
	// 環境変数のキー
	envSlackWebhook    = "SLACK_WEBHOOK_URL"
	envBacklogBase     = "BACKLOG_BASE_URL"
	envBacklogKey      = "BACKLOG_API_KEY"
	envTargetURL       = "TARGET_URL"         // 新規追加
	envTargetProjectID = "BACKLOG_PROJECT_ID" // 新規追加

	// Slack固有設定 (環境変数から取得しない場合はここでデフォルト値を設定)
	envSlackUsername  = "SLACK_USERNAME"
	envSlackIconEmoji = "SLACK_ICON_EMOJI"
	envSlackChannel   = "SLACK_CHANNEL"

	// Backlog 環境依存のデフォルト設定 (実際の環境に合わせて変更が必要です)
	defaultIssueTypeID = 101 // 例: タスクの課題種別ID
	defaultPriorityID  = 3   // 例: 中の優先度ID

	// HTTPクライアントの設定 (すべての通信に適用される)
	defaultTimeout = 60 * time.Second
)

func main() {
	// 1. 環境設定と初期化
	ctx := context.Background()
	log.Println("Go Notifier 処理を開始します...")

	// 2. ターゲットURL（環境変数から取得）
	targetURL := os.Getenv(envTargetURL)
	if targetURL == "" {
		log.Fatal("🚨 致命的なエラー: TARGET_URL が設定されていません。")
	}

	// Backlog プロジェクトID（環境変数から取得）
	targetProjectIDStr := os.Getenv(envTargetProjectID)
	if targetProjectIDStr == "" {
		log.Fatal("🚨 致命的なエラー: BACKLOG_PROJECT_ID が設定されていません。")
	}
	targetProjectID, err := strconv.Atoi(targetProjectIDStr)
	if err != nil {
		log.Fatalf("🚨 致命的なエラー: BACKLOG_PROJECT_ID の値が不正です: %v", err)
	}

	// 3. 共通の httpclient インスタンスの作成 (DIの基盤)
	sharedClient := httpclient.New(defaultTimeout)

	// 4. Notifier インスタンスの作成

	// Slack Notifier の設定
	slackURL := os.Getenv(envSlackWebhook)
	slackNotifier := notifier.NewSlackNotifier(
		sharedClient,
		slackURL,
		os.Getenv(envSlackUsername),
		os.Getenv(envSlackIconEmoji),
		os.Getenv(envSlackChannel),
	)

	// Backlog Notifier の設定
	backlogBase := os.Getenv(envBacklogBase)
	backlogKey := os.Getenv(envBacklogKey)
	backlogNotifier := notifier.NewBacklogNotifier(
		sharedClient,
		backlogBase,
		backlogKey,
		defaultIssueTypeID,
		defaultPriorityID,
		targetProjectID, // SendText 用のデフォルトIDとして渡す
	)

	// 5. 登録する Notifier のリストアップ (設定がない場合は自動的にスキップ)
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

	// 6. ContentNotifier の初期化 (DIの一貫性を保つため sharedClient を渡す)
	contentNotifier := notifier.NewContentNotifier(sharedClient, notifiers...)

	// 7. 通知処理の実行
	log.Printf("URL: %s のコンテンツを抽出・通知します。\n", targetURL)
	log.Printf("Backlog プロジェクトID: %d に課題として投稿されます。\n", targetProjectID)

	if err := contentNotifier.NotifyFromURL(ctx, targetURL, targetProjectID); err != nil {
		// MultiError が返された場合、個々の通知エラーとして処理
		if multiErr, ok := err.(notifier.MultiError); ok {
			for _, e := range multiErr {
				log.Printf("⚠️ 警告: 個別の通知処理でエラーが発生しました: %v", e)
			}
			log.Println("✅ Go Notifier 処理は一部の通知でエラーがありましたが完了しました。")
		} else {
			// Web抽出処理の失敗など、致命的なエラーの場合
			log.Fatalf("🚨 致命的なエラー: 通知処理全体が失敗しました: %v", err)
		}
	} else {
		log.Println("✅ Go Notifier 処理が正常に完了しました。")
	}
}
