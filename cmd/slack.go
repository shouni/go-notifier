package cmd

import (
	"context"
	"log"
	"os"

	"github.com/shouni/go-notifier/pkg/notifier"
	"github.com/spf13/cobra"
)

// Slack 固有の設定フラグ変数
var (
	slackUsername  string
	slackIconEmoji string
	slackChannel   string
)

// 💡 修正: Long の説明を復元
var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slackにプレーンテキストを投稿します",
	Long:  `環境変数 SLACK_WEBHOOK_URL が設定されている必要があります。投稿テキストは Block Kit 形式に変換され、文字数制限が適用されます。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
		if slackWebhookURL == "" {
			log.Fatal("🚨 致命的なエラー: SLACK_WEBHOOK_URL 環境変数が設定されていません。")
		}

		// Notifierの初期化
		slackNotifier := notifier.NewSlackNotifier(
			*sharedClient,
			slackWebhookURL,
			slackUsername,
			slackIconEmoji,
			slackChannel,
		)

		// 投稿実行
		if err := slackNotifier.SendTextWithHeader(context.Background(), "📝 テスト結果", inputMessage); err != nil {
			log.Fatalf("🚨 Slackへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Slackへの投稿が完了しました。")
	},
}

func init() {
	slackCmd.Flags().StringVarP(&slackUsername, "username", "u", os.Getenv("SLACK_USERNAME"), "Slack投稿時のユーザー名 (ENV: SLACK_USERNAME)")
	slackCmd.Flags().StringVarP(&slackIconEmoji, "icon-emoji", "e", os.Getenv("SLACK_ICON_EMOJI"), "Slack投稿時の絵文字アイコン (ENV: SLACK_ICON_EMOJI)")
	slackCmd.Flags().StringVarP(&slackChannel, "channel", "c", os.Getenv("SLACK_CHANNEL"), "Slack投稿先のチャンネル（例: #general）(ENV: SLACK_CHANNEL)")
}
