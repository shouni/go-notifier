package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/shouni/go-cli-base"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/spf13/cobra"
)

const (
	appName           = "notifier"
	defaultTimeoutSec = 10 // 秒
)

// GlobalFlags はこのアプリケーション固有の永続フラグを保持
// clibase.Flags は clibase 共通フラグ（Verbose, ConfigFile）を保持
type AppFlags struct {
	Title      string // -H 投稿タイトル
	Message    string // -m 投稿メッセージ
	TimeoutSec int    // --timeout タイムアウト
}

var Flags AppFlags // アプリケーション固有フラグにアクセスするためのグローバル変数

// sharedClient はすべてのサブコマンドで共有される HTTP クライアント
var sharedClient *httpkit.Client

// --- アプリケーション固有のカスタム関数 ---

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().StringVarP(&Flags.Title, "title", "t", "", "投稿タイトル")
	rootCmd.PersistentFlags().StringVarP(&Flags.Message, "message", "m", "", "投稿メッセージ")
	rootCmd.PersistentFlags().IntVar(&Flags.TimeoutSec, "timeout", defaultTimeoutSec, "HTTPリクエストのタイムアウト時間（秒）")
}

// initAppPreRunE は、clibase共通処理の後に実行される、アプリケーション固有のPersistentPreRunEです。
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	// clibase共通処理（Verboseなど）は clibase 側で既に実行されている

	// HTTPクライアントの初期化ロジック
	timeout := time.Duration(Flags.TimeoutSec) * time.Second
	// request.New() が *request.Client を返す前提
	sharedClient = httpkit.New(timeout)

	// clibaseのVerboseフラグと連携したロギング
	if clibase.Flags.Verbose {
		log.Printf("HTTPクライアントを初期化しました (Timeout: %s)。", timeout)
	}

	// タイムアウト設定が有効かチェックするなどのエラーチェックもここに追加可能
	if Flags.TimeoutSec <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

// --- エントリポイント ---

// Execute は、rootCmd を実行するメイン関数です。
func Execute() {
	// ここで clibase.Execute を使用して、ルートコマンドの構築と実行を委譲します。
	// Execute(アプリ名, カスタムフラグ追加関数, PersistentPreRunE関数, サブコマンド...)
	clibase.Execute(
		appName,
		addAppPersistentFlags,
		initAppPreRunE,
		slackCmd,   // 既存のサブコマンド
		backlogCmd, // 既存のサブコマンド
	)
}
