package notifier

import (
	"context"
	"fmt"
	"time"

	// 外部ライブラリとして参照
	"github.com/shouni/go-web-exact/pkg/httpclient"
	"github.com/shouni/go-web-exact/pkg/web"
)

// Notifier は、任意の形式で通知を送信するための共通インターフェースです。
// SlackやBacklogといった具体的な送信先に依存しません。
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
	// 通知先に依存しないNotifierのリスト
	Notifiers []Notifier

	// Webコンテンツ抽出機能
	extractor *web.Extractor
}

// NewContentNotifier は、ContentNotifierを初期化します。
func NewContentNotifier(timeout time.Duration, notifiers ...Notifier) *ContentNotifier {
	// Notifier全体で利用する基盤となるhttpclientを作成
	// ここでhttpclientのタイムアウトやリトライ設定を一元管理できます。
	httpClient := httpclient.New(timeout)

	// webパッケージのExtractorをhttpclientをFetcherとして使用して初期化
	// (httpclient.Client は web.Fetcher インターフェースを満たすため)
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

	// 抽出されたテキストの最初の行をサマリーとして使用（記事タイトルなど）
	summary := extractedText
	if len(extractedText) > 100 {
		summary = extractedText[:100] + "..."
	}

	// 2. すべてのNotifierに対してループ処理で通知を送信
	// Slackにはテキスト、Backlogには課題として送信するなど、通知の形式を分ける
	for _, n := range c.Notifiers {
		var notifyErr error

		// Notifierの具体的な型に応じて、適切なメソッドを呼び出す
		switch n.(type) {
		case *SlackNotifier:
			// Slackには整形済みテキスト全体を送信
			notifyErr = n.SendText(ctx, fmt.Sprintf("【新着記事】 %s\n\n出典: %s\n\n%s", summary, url, extractedText))
		case *BacklogNotifier:
			// Backlogには課題として送信
			// 抽出されたテキスト全体をDescriptionとする
			notifyErr = n.SendIssue(ctx, summary, fmt.Sprintf("出典URL: %s\n\n%s", url, extractedText), backlogProjectID)
		default:
			// 未知のNotifier型の場合はスキップまたはエラー
			continue
		}

		if notifyErr != nil {
			// いずれかの通知が失敗しても処理を継続し、エラーを記録・集約する
			fmt.Printf("警告: Notifier (%T) への通知に失敗しました: %v\n", n, notifyErr)
			// TODO: エラーを集約して最後に返すロジックを追加
		}
	}

	return nil
}
