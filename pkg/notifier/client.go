package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/pkg/httpclient"
	"github.com/shouni/go-web-exact/pkg/web"
)

// MultiError は複数のエラーを保持するためのカスタムエラー型です。
type MultiError []error

func (m MultiError) Error() string {
	if len(m) == 0 {
		return "no errors"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:\n", len(m)))
	for i, err := range m {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Notifier は、任意の形式で通知を送信するための共通インターフェースです。
type Notifier interface {
	// SendText は、プレーンなテキストメッセージを送信します。
	SendText(ctx context.Context, message string) error

	// SendIssue は、特定の情報を構造化して課題として送信します（Backlogなどに利用）。
	SendIssue(ctx context.Context, summary, description string, projectID int) error
}

// ----------------------------------------------------------------------
// ContentNotifier (Web抽出と通知を統合するクライアント)
// ----------------------------------------------------------------------

// ContentNotifier は、Webコンテンツの抽出と、その結果を各Notifierに渡す役割を担います。
type ContentNotifier struct {
	Notifiers []Notifier
	extractor *web.Extractor
}

// NewContentNotifier は、ContentNotifierを初期化します。
// 引数として *httpclient.Client を受け取り、DIの一貫性を保ちます。
func NewContentNotifier(httpClient *httpclient.Client, notifiers ...Notifier) *ContentNotifier {
	// webパッケージのExtractorを引数で受け取ったhttpclientをFetcherとして使用して初期化
	extractor := web.NewExtractor(httpClient)

	return &ContentNotifier{
		Notifiers: notifiers,
		extractor: extractor,
	}
}

// NotifyFromURL は、指定されたURLからコンテンツを抽出（webパッケージ利用）し、
// すべての登録されたNotifierに通知を送信します。
func (c *ContentNotifier) NotifyFromURL(ctx context.Context, url string, backlogProjectID int) error {
	// 1. Webコンテンツの抽出 (pkg/web と pkg/httpclient の連携)
	extractedText, _, err := c.extractor.FetchAndExtractText(url, ctx)
	if err != nil {
		return fmt.Errorf("URL(%s)からのコンテンツ抽出に失敗しました: %w", url, err)
	}

	// 抽出されたテキストの最初の100文字（マルチバイト対応）をサマリーとして使用
	var summary string
	runes := []rune(extractedText)
	if len(runes) > 100 {
		summary = string(runes[:100]) + "..."
	} else {
		summary = extractedText
	}

	// 2. すべてのNotifierに対してループ処理で通知を送信
	var allErrors MultiError // 複数のエラーを収集するためのスライス

	for _, n := range c.Notifiers {
		var notifyErr error

		// Notifierの具体的な型に応じて、適切なメソッドを呼び出す
		switch n.(type) {
		case *SlackNotifier:
			// Slackには整形済みテキスト全体を送信
			notifyErr = n.SendText(ctx, fmt.Sprintf("【新着記事】 %s\n\n出典: %s\n\n%s", summary, url, extractedText))
		case *BacklogNotifier:
			// Backlogには課題として送信（抽出されたテキスト全体をDescriptionとする）
			notifyErr = n.SendIssue(ctx, summary, fmt.Sprintf("出典URL: %s\n\n%s", url, extractedText), backlogProjectID)
		default:
			// 未知のNotifier型の場合はエラーとして記録
			notifyErr = fmt.Errorf("unknown notifier type: %T", n)
		}

		if notifyErr != nil {
			fmt.Printf("警告: Notifier (%T) への通知に失敗しました: %v\n", n, notifyErr)
			allErrors = append(allErrors, fmt.Errorf("notifier %T failed: %w", n, notifyErr))
		}
	}

	if len(allErrors) > 0 {
		// 抽出は成功したが、通知に失敗したエラーを返す
		return allErrors
	}
	return nil
}
