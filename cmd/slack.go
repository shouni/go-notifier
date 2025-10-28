package cmd

import (
	"context"
	"log"
	"os"

	"go_notifier/pkg/notifier"

	"github.com/spf13/cobra"
)

// Slack 固有の設定フラグ変数
var (
	slackUsername  string
	slackIconEmoji string
	slackChannel   string
)

// Slack Block Kit に合わせた投稿メッセージの制限
const slackTextLimit = 3000

// slackCmd は Cobra の Slack 投稿用サブコマンドです
// 注: inputMessage, sharedClient は同じ cmd パッケージ内の root.go (または共有ファイル) で定義されている前提です
var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slackにメッセージを投稿します（Block Kit形式、文字数制限あり）",
	Long:  `環境変数 SLACK_WEBHOOK_URL が設定されている必要があります。投稿テキストは Block Kit 形式に変換され、文字数制限が適用されます。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
		if slackWebhook == "" {
			log.Fatal("🚨 致命的なエラー: SLACK_WEBHOOK_URL 環境変数が設定されていません。")
		}

		// Notifier の初期化
		slackNotifier := notifier.NewSlackNotifier(
			sharedClient,
			slackWebhook,
			slackUsername,
			slackIconEmoji,
			slackChannel,
		)

		// 投稿メッセージの整形と制限
		messageToSend := inputMessage
		runes := []rune(messageToSend)
		if len(runes) > slackTextLimit {
			// 文字数（rune）で切り詰め
			messageToSend = string(runes[:slackTextLimit]) + "..."
			log.Printf("⚠️ 警告: メッセージが %d 文字を超えたため、%d 文字に切り詰められました。", len(runes), slackTextLimit)
		}

		// 投稿実行
		if err := slackNotifier.SendText(context.Background(), messageToSend); err != nil {
			log.Fatalf("🚨 Slackへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Slackへの投稿が完了しました。")
	},
}

func init() {
	// Slack コマンド固有のフラグを定義
	slackCmd.Flags().StringVar(&slackUsername, "username", os.Getenv("SLACK_USERNAME"), "Slack投稿時のユーザー名 (ENV: SLACK_USERNAME)")
	slackCmd.Flags().StringVar(&slackIconEmoji, "icon-emoji", os.Getenv("SLACK_ICON_EMOJI"), "Slack投稿時の絵文字アイコン (ENV: SLACK_ICON_EMOJI)")
	slackCmd.Flags().StringVar(&slackChannel, "channel", os.Getenv("SLACK_CHANNEL"), "Slack投稿先のチャンネル（例: #general）(ENV: SLACK_CHANNEL)")
}
