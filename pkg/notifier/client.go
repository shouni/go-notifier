package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/v2/pkg/extract"
)

// Notifier は、外部システムへの通知処理のインターフェースを定義します。
type Notifier interface {
	// SendText は、プレーンテキストメッセージを通知します。（ヘッダーなし）
	SendText(ctx context.Context, message string) error

	// SendTextWithHeader は、ヘッダー付きのテキストメッセージを通知します。
	SendTextWithHeader(ctx context.Context, headerText string, message string) error

	// SendIssue は、Backlogなどの課題管理システムに課題を登録します。
	// summary, description に加え、Backlogの必須フィールドである projectID, issueTypeID, priorityID を引数に含めます。
	SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error
}

// ContentNotifier は、Web抽出と複数のNotifierへの通知を管理します。
type ContentNotifier struct {
	extractor *extract.Extractor // Webコンテンツ抽出機
	Notifiers []Notifier         // 登録されている全ての通知先
}

// NewContentNotifier は ContentNotifier を初期化します。
func NewContentNotifier(extractor *extract.Extractor, notifiers ...Notifier) *ContentNotifier {
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
// NOTE: NotifierはSendText/SendTextWithHeaderをサポートしない場合エラーを返すことがあり、
// その場合、エラーは収集され呼び出し元に返されます。BacklogNotifierが登録されており、
// backlogProjectIDが0の場合、BacklogNotifierはテキスト通知をサポートしないため通知に失敗します。
func (c *ContentNotifier) Notify(ctx context.Context, url string, backlogProjectID, issueTypeID, priorityID int) error {
	// 1. Webコンテンツの抽出 (c.extractor を使用)
	// hasBodyFound は現在未使用のため、アンダースコア (_) で無視します。
	text, _, err := c.extractor.FetchAndExtractText(url, ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch and extract content from URL %s: %w", url, err)
	}

	// 抽出されたテキストをサマリーと詳細に分割
	var summary string
	var description string

	// 最初の改行で分割 (修正: "\n\n" ではなく "\n" で分割し、より柔軟に対応)
	lines := strings.SplitN(text, "\n", 2)
	summary = lines[0]
	if len(lines) > 1 {
		description = lines[1]
	} else {
		description = summary // 本文がない場合はサマリーを本文として使用
	}

	var allErrors []error

	// 2. 抽出結果を全てのNotifierに通知
	for _, n := range c.Notifiers {
		var notifyErr error

		// Backlogなどの課題登録が可能なNotifierに対しては SendIssue を優先
		if backlogProjectID != 0 {
			// BacklogNotifierの場合のみ、issueTypeIDとpriorityIDのバリデーションを行う
			if _, ok := n.(*BacklogNotifier); ok {
				if issueTypeID == 0 || priorityID == 0 { // Backlog APIの仕様上、これらのIDは必須
					allErrors = append(allErrors, fmt.Errorf("Notifier (%T): Backlogへの課題登録には issueTypeID (%d) と priorityID (%d) が非ゼロである必要があります", n, issueTypeID, priorityID))
					continue // このNotifierへの通知をスキップ
				}
			}

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
