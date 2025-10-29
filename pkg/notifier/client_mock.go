package notifier

import (
	"context"
	"log"
	"strings"
)

// MockNotifier は Notifier インターフェースを実装し、実際のAPIコールを行わずログに出力します。
type MockNotifier struct {
	Name string
}

// NewMockNotifier は MockNotifier のインスタンスを作成します。
func NewMockNotifier(name string) *MockNotifier {
	return &MockNotifier{Name: name}
}

// truncateAndClean はメッセージを指定された長さに切り詰め、不要な改行やスペースを除去します。
func truncateAndClean(message string, maxLen int) string {
	end := len(message)
	if end > maxLen {
		end = maxLen
	}

	// 不必要な改行やスペースを除去し、整形された最初の部分を出力
	preview := strings.ReplaceAll(message[:end], "\n", " ")
	preview = strings.TrimSpace(preview)
	return preview
}

// --- Notifier インターフェース実装 ---

// SendText は実際の投稿の代わりにログを出力します。（ヘッダーなし）
func (m *MockNotifier) SendText(ctx context.Context, message string) error {
	const maxLen = 50
	preview := truncateAndClean(message, maxLen)

	log.Printf("🤖 MockNotifier (%s): SendText 実行 -> テキスト: %s... (最初の%d文字)",
		m.Name, preview, len(preview))
	return nil // 成功を返す
}

// SendTextWithHeader は実際の投稿の代わりにログを出力します。（ヘッダーあり）
func (m *MockNotifier) SendTextWithHeader(ctx context.Context, headerText string, message string) error {
	const maxLen = 50
	preview := truncateAndClean(message, maxLen)

	log.Printf("🤖 MockNotifier (%s): SendTextWithHeader 実行 -> ヘッダー: %s, 本文: %s... (最初の%d文字)",
		m.Name, headerText, preview, len(preview))
	return nil // 成功を返す
}

// SendIssue は実際の課題登録の代わりにログを出力します。
func (m *MockNotifier) SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error {

	log.Printf("🤖 MockNotifier (%s): SendIssue 実行 -> サマリー: %s, 本文の長さ: %d, ProjectID: %d, IssueTypeID: %d, PriorityID: %d",
		m.Name, summary, len(description), projectID, issueTypeID, priorityID)

	// 必要に応じて、特定のテストケースでエラーを返すことも可能
	// if m.Name == "ErrorTest" {
	//    return errors.New("モック通知エラーをシミュレーション")
	// }
	return nil // 成功を返す
}
