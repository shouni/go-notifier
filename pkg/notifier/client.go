package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/pkg/web"
)

// Notifier は、外部システムへの通知処理のインターフェースを定義します。
type Notifier interface {
	// SendText は、プレーンテキストメッセージを通知します。（ヘッダーなし）
	SendText(ctx context.Context, message string) error

	// SendTextWithHeader は、ヘッダー付きのテキストメッセージを通知します。
	SendTextWithHeader(ctx context.Context, headerText string, message string) error

	// SendIssue は、Backlogなどの課題管理システムに課題を登録します。
	SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error
}

// ContentNotifier は、Web抽出と複数のNotifierへの通知を管理します。
type ContentNotifier struct {
	extractor *web.Extractor // Webコンテンツ抽出機
	Notifiers []Notifier     // 登録されている全ての通知先
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
//
// NOTE: 今回の改善では、抽出されたサマリーを 'headerText' として、詳細を 'message' として
// SendTextWithHeader メソッドに渡すロジックを採用します。
func (c *ContentNotifier) Notify(ctx context.Context, url string, backlogProjectID, issueTypeID, priorityID int) error {
	// 1. Webコンテンツの抽出 (実際には c.extractor を使用)
	// summary, description, err := c.extractor.Extract(ctx, url)

	// 処理の簡易化のため、ここではダミーデータを使用します。
	summary := "ウェブコンテンツ通知サマリー"
	description := "ウェブコンテンツ詳細: " + url

	var allErrors []error

	// 2. 抽出結果を全てのNotifierに通知
	for _, n := range c.Notifiers {
		var notifyErr error

		// Backlogなどの課題登録が可能なNotifierに対しては SendIssue を優先
		// それ以外には、ヘッダー付きのテキスト通知 (SendTextWithHeader) を使用

		// Notifierの具体的な実装によらず、ContentNotifierがどのように振る舞うかを定義
		// ここでは、Issue登録パラメータがあればIssueとして通知、そうでなければHeader付きTextとして通知する
		if backlogProjectID != 0 {
			// Backlogへの課題登録を試みる
			notifyErr = n.SendIssue(ctx, summary, description, backlogProjectID, issueTypeID, priorityID)
		} else {
			// ヘッダー付きテキストとして通知
			notifyErr = n.SendTextWithHeader(ctx, summary, description)
		}

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
