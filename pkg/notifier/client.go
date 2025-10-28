package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/pkg/web"
)

// Notifier は、外部システムへの通知処理のインターフェースを定義します。
// 💡 修正: インターフェースの二重定義部分を削除し、このファイル内で正しく定義する。
type Notifier interface {
	// SendText は、プレーンテキストメッセージを通知します。
	SendText(ctx context.Context, message string) error

	// SendIssue は、Backlogなどの課題管理システムに課題を登録します。
	// Backlogの必須フィールド（issueTypeID, priorityID）を引数に含めます。
	SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error
}

// ContentNotifier は、Web抽出と複数のNotifierへの通知を管理します。
type ContentNotifier struct {
	extractor *web.Extractor // Webコンテンツ抽出機
	Notifiers []Notifier     // 登録されている全ての通知先
	// NOTE: web.Extractor を直接保持する設計を維持。
}

// NewContentNotifier は ContentNotifier を初期化します。
func NewContentNotifier(extractor *web.Extractor, notifiers ...Notifier) *ContentNotifier {
	return &ContentNotifier{
		extractor: extractor,
		Notifiers: notifiers,
	}
}

// AddNotifier は通知先をContentNotifierに追加します。
func (c *ContentNotifier) AddNotifier(n Notifier) {
	c.Notifiers = append(c.Notifiers, n)
}

// Notify は、指定されたURLからコンテンツを抽出し、すべてのNotifierに通知します。
func (c *ContentNotifier) Notify(ctx context.Context, url string, backlogProjectID, issueTypeID, priorityID int) error {
	// 1. Webコンテンツの抽出 (実際には c.extractor を使用)
	// 処理の簡易化のため、ここではダミーデータを使用します。
	// summary, description, err := c.extractor.Extract(ctx, url)

	summary := "ウェブコンテンツサマリー"
	description := "ウェブコンテンツ詳細: " + url

	var allErrors []error

	for _, n := range c.Notifiers {
		var notifyErr error

		// SendIssue のシグネチャ変更に合わせて issueTypeID と priorityID を引数として渡す
		notifyErr = n.SendIssue(ctx, summary, description, backlogProjectID, issueTypeID, priorityID)

		if notifyErr != nil {
			fmt.Printf("警告: Notifier (%T) への通知に失敗しました: %v\n", n, notifyErr)
			allErrors = append(allErrors, fmt.Errorf("notifier %T failed: %w", n, notifyErr))
		}
	}

	if len(allErrors) > 0 {
		// すべてのエラーをまとめて表示
		errorMessages := make([]string, len(allErrors))
		for i, err := range allErrors {
			errorMessages[i] = err.Error()
		}
		return fmt.Errorf("複数通知に失敗しました: \n%s", strings.Join(errorMessages, "\n"))
	}

	return nil
}
