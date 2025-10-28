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

// SendText は実際の投稿の代わりにログを出力します。
func (m *MockNotifier) SendText(ctx context.Context, message string) error {
	// メッセージの最初の50文字（またはそれ以下）を取得してログ出力
	const maxLen = 50
	end := len(message)
	if end > maxLen {
		end = maxLen
	}

	// 不必要な改行やスペースを除去し、整形された最初の部分を出力
	preview := strings.ReplaceAll(message[:end], "\n", " ")
	preview = strings.TrimSpace(preview)

	log.Printf("🤖 MockNotifier (%s): SendText 実行 -> テキスト: %s... (最初の%d文字)",
		m.Name, preview, end)
	return nil // 成功を返す
}

// SendIssue は実際の課題登録の代わりにログを出力します。
func (m *MockNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {

	// BacklogNotifier/SlackNotifier と同様に、DIのテストとして利用可能

	log.Printf("🤖 MockNotifier (%s): SendIssue 実行 -> サマリー: %s, 本文の長さ: %d, プロジェクトID: %d",
		m.Name, summary, len(description), projectID)

	// 必要に応じて、特定のテストケースでエラーを返すことも可能
	// if m.Name == "ErrorTest" {
	//    return errors.New("モック通知エラーをシミュレーション")
	// }
	return nil // 成功を返す
}
