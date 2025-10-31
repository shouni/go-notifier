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
	issueID      string
)

// getBacklogNotifier は環境変数チェックを行い、Backlog Notifierを生成します。
// sharedClient は PersistentPreRunE で初期化済みのため、そのまま使用します。
func getBacklogNotifier() (*notifier.BacklogNotifier, error) {
	backlogSpaceURL := os.Getenv("BACKLOG_SPACE_URL")
	backlogAPIKey := os.Getenv("BACKLOG_API_KEY")
	if backlogSpaceURL == "" || backlogAPIKey == "" {
		return nil, fmt.Errorf("BACKLOG_SPACE_URL または BACKLOG_API_KEY 環境変数が設定されていません")
	}

	// Notifierの初期化に sharedClient を使用
	return notifier.NewBacklogNotifier(*sharedClient, backlogSpaceURL, backlogAPIKey)
}

// --- サブコマンド: backlog (課題登録) ---

// backlogCmd は Cobra の Backlog 課題登録用サブコマンドです
var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Backlogへの課題登録またはコメント投稿を管理します",
	Long:  `環境変数 BACKLOG_SPACE_URL と BACKLOG_API_KEY が設定されている必要があります。`,
	Run: func(cmd *cobra.Command, args []string) {
		backlogNotifier, err := getBacklogNotifier()
		if err != nil {
			log.Fatalf("🚨 Backlog Notifierの初期化に失敗しました: %v", err)
		}

		// プロジェクトIDの取得とチェック
		projectID, err := backlogNotifier.GetProjectID(context.Background(), projectIDStr)
		if err != nil {
			log.Fatalf("🚨 致命的なエラー: プロジェクトIDの取得に失敗しました: %v", err)
		}

		// 🚨 修正点1: 課題サマリーのチェックで Flags.Header を使用
		if Flags.Title == "" {
			log.Fatal("🚨 致命的なエラー: 課題のタイトルがありません。-t フラグでタイトルを指定してください。")
		}

		if Flags.Message == "" {
			log.Fatal("🚨 致命的なエラー: 課題のメッセージがありません。-m フラグでメッセージを指定してください。")
		}

		// 2. 投稿実行（SendIssueを使用）
		if err := backlogNotifier.SendIssue(
			context.Background(),
			Flags.Title,   // Backlogの課題サマリーとして使用
			Flags.Message, // Backlogの課題説明として使用
			projectID,
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
		// 🚨 修正点2: 投稿メッセージのチェックで Flags.Message を使用
		if Flags.Message == "" {
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
			log.Fatalf("🚨 Backlog Notifierの初期化に失敗しました: %v", err)
		}

		// 投稿実行（SendCommentを使用 - 課題キーとメッセージを渡す）
		// 🚨 修正点3: 投稿メッセージに Flags.Message を使用
		if err := backlogNotifier.PostComment(
			context.Background(),
			issueID,
			Flags.Message,
		); err != nil {
			log.Fatalf("🚨 Backlogへのコメント投稿に失敗しました: %v", err)
		}

		log.Printf("✅ Backlog課題 (%s) へのコメント投稿が完了しました。", issueID)
	},
}

func init() {
	// init() 内での projectIDStr の環境変数からの初期設定はフラグ定義に統合する

	// backlogCmd のフラグ定義
	projectIDEnv := os.Getenv("BACKLOG_PROJECT_ID")
	backlogCmd.Flags().StringVarP(&projectIDStr, "project-id", "p", projectIDEnv, "【必須】課題を登録する Backlog のプロジェクトID (ENV: BACKLOG_PROJECT_ID)")

	// commentCmd のフラグ定義
	commentCmd.Flags().StringVarP(&issueID, "issue-id", "i", "", "【必須】コメントを投稿する Backlog 課題 ID (例: PROJECT-123)")
}
