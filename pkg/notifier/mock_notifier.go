package notifier

import (
	"context"
	"log"
)

// MockNotifier は Notifier インターフェースを実装し、実際のAPIコールを行わずログに出力します。
type MockNotifier struct {
	Name string
}

// NewMockNotifier は MockNotifier のインスタンスを作成します。
func NewMockNotifier(name string) *MockNotifier {
	return &MockNotifier{Name: name}
}

// SendText は実際の投稿の代わりにログを出力します。
func (m *MockNotifier) SendText(ctx context.Context, message string) error {
	log.Printf("🤖 MockNotifier (%s): SendText 実行 -> テキスト: %s (最初の50文字)",
		m.Name, message[:min(50, len(message))])
	return nil // 成功を返す
}

// SendIssue は実際の課題登録の代わりにログを出力します。
func (m *MockNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	log.Printf("🤖 MockNotifier (%s): SendIssue 実行 -> サマリー: %s, プロジェクトID: %d",
		m.Name, summary, projectID)
	// 投稿先がないため、テストとして意図的にエラーを返すことも可能
	// return fmt.Errorf("投稿先がないためモックエラー")
	return nil // 成功を返す
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
