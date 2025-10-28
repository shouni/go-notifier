package cmd

import (
	"context"
	"log"
	"os"

	"go_notifier/pkg/notifier"

	"github.com/spf13/cobra"
)

// 💡 修正: inputMessage と timeoutSec は cmd/root.go で定義されるため、パッケージレベルの変数は削除

// slackCmd は Cobra の Slack 投稿用サブコマンドです
var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slackにプレーンテキストを投稿します",
	Long:  `環境変数 SLACK_WEBHOOK_URL が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		// 環境変数から Webhook URL を取得し、定義
		slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
		if slackWebhookURL == "" {
			log.Fatal("🚨 致命的なエラー: SLACK_WEBHOOK_URL 環境変数が設定されていません。")
		}

		// Notifierの初期化
		// 💡 修正: sharedClient を使用 (ローカルの httpclient.New の呼び出しを削除)
		slackNotifier := notifier.NewSlackNotifier(sharedClient, slackWebhookURL)

		// 投稿実行
		if err := slackNotifier.SendText(context.Background(), inputMessage); err != nil {
			log.Fatalf("🚨 Slackへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Slackへの投稿が完了しました。")
	},
}
