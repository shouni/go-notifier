package cmd

import (
	"context"
	"log"
	"os"
	"strconv"
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

// --- サブコマンド: backlog ---

// backlogCmd は Cobra の Backlog 課題登録用サブコマンドです（ルートコマンドとしても機能）
var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Backlogへの課題登録またはコメント投稿を管理します",
	Long:  `環境変数 BACKLOG_SPACE_URL と BACKLOG_API_KEY が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 引数がない場合、デフォルトで課題登録機能を実行
		// 課題登録ロジック
		if inputMessage == "" {
			log.Fatal("🚨 致命的なエラー: 投稿メッセージがありません。-m フラグでメッセージを指定してください。")
		}

		// 環境変数のチェックと定義
		backlogSpaceURL := os.Getenv("BACKLOG_SPACE_URL")
		backlogAPIKey := os.Getenv("BACKLOG_API_KEY")
		if backlogSpaceURL == "" || backlogAPIKey == "" {
			log.Fatal("🚨 致命的なエラー: BACKLOG_SPACE_URL または BACKLOG_API_KEY 環境変数が設定されていません。")
		}

		// プロジェクトIDの取得とチェック
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil || projectID <= 0 {
			log.Fatalf("🚨 致命的なエラー: --project-id の値が不正です: %v", err)
		}

		// 1. サマリーと説明への分割 (課題登録用)
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
		backlogNotifier, err := notifier.NewBacklogNotifier(sharedClient, backlogSpaceURL, backlogAPIKey)
		if err != nil {
			log.Fatalf("🚨 Backlog Notifierの初期化に失敗しました: %v", err)
		}

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
			log.Fatal("🚨 致命的なエラー: --issue-key フラグでコメント対象の課題キーを指定してください。")
		}

		// 環境変数のチェックと定義 (backlogCmd と共通)
		backlogSpaceURL := os.Getenv("BACKLOG_SPACE_URL")
		backlogAPIKey := os.Getenv("BACKLOG_API_KEY")
		if backlogSpaceURL == "" || backlogAPIKey == "" {
			log.Fatal("🚨 致命的なエラー: BACKLOG_SPACE_URL または BACKLOG_API_KEY 環境変数が設定されていません。")
		}

		// Notifier の初期化
		backlogNotifier, err := notifier.NewBacklogNotifier(sharedClient, backlogSpaceURL, backlogAPIKey)
		if err != nil {
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
	commentCmd.Flags().StringVarP(&issueID, "issue-id", "i", "", "【必須】コメントを投稿する課題のキー（例: PROJECT-123）")
	rootCmd.AddCommand(backlogCmd)
	backlogCmd.AddCommand(commentCmd)
}
