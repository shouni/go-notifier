package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/shouni/go-web-exact/pkg/httpclient"

	"go_notifier/pkg/notifier"

	"github.com/spf13/cobra"
)

// slackCmd は Cobra の Slack 投稿用サブコマンドです
var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slackにプレーンテキストを投稿します",
	Long:  `環境変数 SLACK_WEBHOOK_URL が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		// 💡 修正点 1: slackWebhookURL を Run 関数内で定義・取得
		slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
		if slackWebhookURL == "" {
			log.Fatal("🚨 致命的なエラー: SLACK_WEBHOOK_URL 環境変数が設定されていません。")
		}

		// 💡 修正点 2: httpClient を Run 関数内で初期化
		httpClient := httpclient.New(time.Duration(timeoutSec) * time.Second)

		// Notifierの初期化
		// (httpclient.HTTPClient, string) の新しいシグネチャに適合
		slackNotifier := notifier.NewSlackNotifier(httpClient, slackWebhookURL)

		// 投稿実行
		if err := slackNotifier.SendText(context.Background(), inputMessage); err != nil {
			log.Fatalf("🚨 Slackへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Slackへの投稿が完了しました。")
	},
}

// ⚠️ 注意:
// このコードは、他のファイル（例: cmd/root.go）で
// `slackCmd` がルートコマンドに追加され、
// `inputMessage` および `timeoutSec` がフラグとして
// 定義・パースされていることを前提としています。
