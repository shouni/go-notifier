package cmd

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"go_notifier/pkg/notifier"

	"github.com/spf13/cobra"
)

// Backlog 固有の設定フラグ変数
var (
	projectIDStr string
	issueTypeID  int
	priorityID   int
)

// backlogCmd は Cobra の Backlog 課題登録用サブコマンドです
var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Backlogに課題として投稿します（Notifier側で絵文字除去）",
	Long:  `環境変数 BACKLOG_BASE_URL と BACKLOG_API_KEY が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		// 環境変数のチェック
		backlogBase := os.Getenv("BACKLOG_BASE_URL")
		backlogKey := os.Getenv("BACKLOG_API_KEY")
		if backlogBase == "" || backlogKey == "" {
			log.Fatal("🚨 致命的なエラー: BACKLOG_BASE_URL または BACKLOG_API_KEY 環境変数が設定されていません。")
		}

		// プロジェクトIDの取得とチェック
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil || projectID <= 0 {
			log.Fatalf("🚨 致命的なエラー: --project-id の値が不正です: %v", err)
		}

		// 1. サマリーと説明への分割（絵文字除去は Notifier 側に任せる）
		lines := strings.SplitN(inputMessage, "\n", 2)
		summary := strings.TrimSpace(lines[0])
		description := ""
		if len(lines) > 1 {
			description = strings.TrimSpace(lines[1])
		}

		if summary == "" {
			log.Fatal("🚨 致命的なエラー: 課題のサマリーとなるテキストがありません。")
		}

		// Notifier の初期化
		backlogNotifier := notifier.NewBacklogNotifier(
			sharedClient,
			backlogBase,
			backlogKey,
			issueTypeID,
			priorityID,
			projectID,
		)

		// 2. 投稿実行（SendIssueを使用）
		// Notifierが内部で絵文字除去を行う
		if err := backlogNotifier.SendIssue(context.Background(), summary, description, projectID); err != nil {
			log.Fatalf("🚨 Backlogへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Backlogへの課題登録が完了しました。")
	},
}

func init() {
	// Backlog コマンド固有の必須フラグとオプションフラグを定義
	projectIDStr = os.Getenv("BACKLOG_PROJECT_ID")
	backlogCmd.Flags().StringVar(&projectIDStr, "project-id", projectIDStr, "【必須】課題を登録する Backlog のプロジェクトID (ENV: BACKLOG_PROJECT_ID)")
	backlogCmd.Flags().IntVar(&issueTypeID, "issue-type-id", 101, "課題の種別ID（例: 101 for タスク）")
	backlogCmd.Flags().IntVar(&priorityID, "priority-id", 3, "課題の優先度ID（例: 3 for 中）")
}
