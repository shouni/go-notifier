package cmd

import (
	"log"
	"os"
	"time"

	request "github.com/shouni/go-web-exact/v2/pkg/client"
	"github.com/spf13/cobra"
)

var (
	inputHeader  string // -h フラグで受け取る投稿ヘッダー、BacklogのサマリーやSlackのヘッダーに使用
	inputMessage string // -m フラグで受け取る投稿メッセージ、Backlogの課題説明やSlackの本文に使用
	timeoutSec   int    // HTTPリクエストのタイムアウト時間（秒）
)

const (
	defaultTimeoutSec = 60 // 秒
)

// sharedClient はすべてのサブコマンドで共有される HTTP クライアント
// 記憶したパッケージの Client 型に変更
var sharedClient *request.Client

// rootCmd はアプリケーションのベースとなるコマンド
var rootCmd = &cobra.Command{
	Use:   "notifier",
	Short: "SlackとBacklogへの通知を管理するCLIツール",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		timeout := time.Duration(timeoutSec) * time.Second
		sharedClient = request.New(timeout)
		log.Printf("HTTPクライアントを初期化しました (Timeout: %s)。", timeout)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute は、rootCmd を実行するメイン関数です。
// main.go から呼び出されます。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&inputHeader, "header", "H", "", "ヘッダー（テキスト）")
	rootCmd.PersistentFlags().StringVarP(&inputMessage, "message", "m", "", "投稿するメッセージ（テキスト）")
	rootCmd.PersistentFlags().IntVar(&timeoutSec, "timeout", defaultTimeoutSec, "HTTPリクエストのタイムアウト時間（秒）")

	rootCmd.AddCommand(slackCmd)
	rootCmd.AddCommand(backlogCmd)
}
