package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/shouni/go-notifier/pkg/notifier"
	"github.com/spf13/cobra"
)

// Backlog 固有の設定フラグ変数
var (
	projectIDStr string
	issueTypeID  int
	priorityID   int
	issueID      string
)

// 実行前に rootCmd の PersistentPreRun で sharedClient が初期化されている必要があります。
func getBacklogNotifier() (*notifier.BacklogNotifier, error) {
	backlogSpaceURL := os.Getenv("BACKLOG_SPACE_URL")
	backlogAPIKey := os.Getenv("BACKLOG_API_KEY")
	if backlogSpaceURL == "" || backlogAPIKey == "" {
		return nil, fmt.Errorf("BACKLOG_SPACE_URL または BACKLOG_API_KEY 環境変数が設定されていません")
	}

	return notifier.NewBacklogNotifier(*sharedClient, backlogSpaceURL, backlogAPIKey)
}

// --- サブコマンド: backlog (課題登録) ---

// backlogCmd は Cobra の Backlog 課題登録用サブコマンドです
var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Backlogへの課題登録またはコメント投稿を管理します",
	Long:  `環境変数 BACKLOG_SPACE_URL と BACKLOG_API_KEY が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		backlogNotifier, err := getBacklogNotifier()
		if err != nil {
			// 環境変数エラーもここでハンドリングされる
			log.Fatalf("🚨 Backlog Notifierの初期化に失敗しました: %v", err)
		}

		// プロジェクトIDの取得とチェック
		projectID, err := backlogNotifier.GetProjectID(context.Background(), projectIDStr)
		if err != nil || projectID <= 0 {
			log.Fatalf("🚨 致命的なエラー: --project-id の値が不正です: %v", err)
		}

		// 1. サマリーと説明への分割
		lines := strings.SplitN(inputMessage, "\n", 2)
		summary := strings.TrimSpace(lines[0])
		description := ""
		if len(lines) > 1 {
			description = strings.TrimSpace(lines[1])
		}

		if summary == "" {
			log.Fatal("🚨 致命的なエラー: 課題のサマリーとなるテキストがありません。")
		}

		// TODO::APIから取得できればいいがデフォルト指定
		issueTypeID = 1
		priorityID = 1

		// 2. 投稿実行（SendIssueを使用）
		if err := backlogNotifier.SendIssue(
			context.Background(),
			summary,
			description,
			projectID,
			issueTypeID,
			priorityID,
		); err != nil {
			log.Fatalf("🚨 Backlogへの投稿に失敗しました: %v", err)
		}

		log.Println("✅ Backlogへの課題登録が完了しました。")
	},
}

// --- サブコマンド: comment (backlogの子) ---

// commentCmd は Backlog 既存課題へのコメント投稿用サブコマンドです
var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "既存の課題にコメントを追記します",
	Run: func(cmd *cobra.Command, args []string) {
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		if issueID == "" {
			log.Fatal("🚨 致命的なエラー: --issue-id フラグでコメント対象の課題キーを指定してください。")
		}

		if !strings.Contains(issueID, "-") {
			log.Fatalf("🚨 致命的なエラー: --issue-id の値が不正な形式です。例: PROJECT-123 (含まれているハイフンがありません)")
		}

		backlogNotifier, err := getBacklogNotifier()
		if err != nil {
			// 環境変数エラーもここでハンドリングされる
			log.Fatalf("🚨 Backlog Notifierの初期化に失敗しました: %v", err)
		}

		// 投稿実行（SendCommentを使用 - 課題キーとメッセージを渡す）
		if err := backlogNotifier.PostComment(
			context.Background(),
			issueID,
			inputMessage,
		); err != nil {
			log.Fatalf("🚨 Backlogへのコメント投稿に失敗しました: %v", err)
		}

		log.Printf("✅ Backlog課題 (%s) へのコメント投稿が完了しました。", issueID)
	},
}

func init() {
	projectIDStr = os.Getenv("BACKLOG_PROJECT_ID")
	backlogCmd.Flags().StringVarP(&projectIDStr, "project-id", "p", projectIDStr, "【必須】課題を登録する Backlog のプロジェクトID (ENV: BACKLOG_PROJECT_ID)")
	backlogCmd.Flags().IntVarP(&issueTypeID, "issue-type-id", "t", 101, "課題の種別ID（例: 101 for タスク）")
	backlogCmd.Flags().IntVarP(&priorityID, "priority-id", "r", 3, "課題の優先度ID（例: 3 for 中）")
	commentCmd.Flags().StringVarP(&issueID, "issue-id", "i", "", "【必須】コメントを投稿する Backlog 課題 ID (例: PROJECT-123)")
	backlogCmd.AddCommand(commentCmd)
}
