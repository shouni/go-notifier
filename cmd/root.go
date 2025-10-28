package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shouni/go-web-exact/pkg/httpclient"
	"github.com/spf13/cobra"
)

// 設定フラグのグローバル変数 (すべてのサブコマンドで参照可能)
var (
	inputMessage string // -m フラグで受け取る投稿メッセージ
	// targetURL string  // Web抽出を省略したため不要だが、必要ならここで定義
)

// 共通クライアントと定数
const (
	defaultTimeout = 60 * time.Second
)

// sharedClient はすべてのサブコマンドで共有される HTTP クライアント
var sharedClient *httpclient.Client

// rootCmd はアプリケーションのベースとなるコマンド
var rootCmd = &cobra.Command{
	Use:   "go_notifier",
	Short: "SlackとBacklogへの通知を管理するCLIツール",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// すべてのサブコマンド実行前に共有クライアントを初期化
		sharedClient = httpclient.New(defaultTimeout)
		log.Println("HTTPクライアントを初期化しました。")
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute はルートコマンドを実行するエントリーポイントです
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// グローバルなフラグ（すべてのサブコマンドで利用可能）を定義
	rootCmd.PersistentFlags().StringVarP(&inputMessage, "message", "m", "", "投稿するメッセージ（テキスト）")

	// サブコマンドの追加 (slackCmd と backlogCmd はそれぞれ cmd/slack.go と cmd/backlog.go で定義されている)
	rootCmd.AddCommand(slackCmd)
	rootCmd.AddCommand(backlogCmd)
}
