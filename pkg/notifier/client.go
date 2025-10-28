package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/pkg/httpclient"
	"github.com/shouni/go-web-exact/pkg/web"
)

// MultiError は複数のエラーを保持するためのカスタムエラー型です。
// Notifierの処理を継続しつつ、最終的にすべてを報告するために使用されます。
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
	// SlackNotifierなどの課題機能がない実装は、この情報を使ってテキストを整形し、SendTextにフォールバックします。
	SendIssue(ctx context.Context, summary, description string, projectID int) error
}

// ----------------------------------------------------------------------
// ContentNotifier (Web抽出と通知を統合するクライアント)
// ----------------------------------------------------------------------

// ContentNotifier は、Webコンテンツの抽出と、その結果を各Notifierに渡す役割を担います。
type ContentNotifier struct {
	Notifiers []Notifier
	// 依存性注入の一貫性のために、web.Extractorではなく、その生成に必要なhttpclient.Clientを受け取る設計を維持
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

	// Notifierに渡すための詳細な本文情報を作成
	description := fmt.Sprintf("出典URL: %s\n\n%s", url, extractedText)

	// 2. すべてのNotifierに対してループ処理で通知を送信
	var allErrors MultiError // 複数のエラーを収集するためのスライス

	for _, n := range c.Notifiers {
		var notifyErr error

		// 💡 修正箇所: 具体的な型判定（switch n.(type)）を削除し、インターフェースメソッドを直接呼び出す
		// Notifierの実装側が、渡されたprojectIDや自身の通知先に合わせた処理（課題作成 or テキスト整形）を実行する責任を負う。

		// SlackNotifierはこのメソッド内でテキストを整形し、SendTextを呼び出す
		// BacklogNotifierはこのメソッド内で課題作成APIを呼び出す
		notifyErr = n.SendIssue(ctx, summary, description, backlogProjectID)

		if notifyErr != nil {
			// Notifierがエラーを返した場合、処理を中断せず、エラーを記録して次のNotifierへ進む
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
