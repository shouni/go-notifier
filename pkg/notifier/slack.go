package notifier

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/slack-go/slack"
)

// SlackNotifier は Slack Webhook API と連携するためのクライアントです。
// Notifier インターフェースを満たします。
type SlackNotifier struct {
	// WebhookURL: 必須の通知先URL
	WebhookURL string
	// httpClient: 汎用クライアント (リトライロジックを含む)
	client    httpkit.Client
	Username  string
	IconEmoji string
	Channel   string
}

// NewSlackNotifier は SlackNotifier の新しいインスタンスを作成します。
func NewSlackNotifier(client httpkit.Client, webhookURL, username, iconEmoji, channel string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		client:     client,
		Username:   username,
		IconEmoji:  iconEmoji,
		Channel:    channel,
	}
}

// --- Notifier インターフェース実装 ---

// SendTextWithHeader は、ヘッダー付きのテキストメッセージを解析し、SlackのBlock Kit形式で投稿します。
// headerText は、Slackメッセージのヘッダーとして表示されるテキストです。
// message は、抽出された本文全体（Markdownとして解釈可能）を想定します。
func (s *SlackNotifier) SendTextWithHeader(ctx context.Context, headerText string, message string) error {
	// --- 1. Block Kitの構築ロジック（流用元のロジックを汎用化） ---

	// 外部から指定されたheaderTextを使用してヘッダーブロックを作成
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", headerText, true, false),
		),
		slack.NewDividerBlock(),
	}

	// 流用元と同様の整形と文字数制限の定数
	const maxSectionLength = 2900
	const maxBlocks = 50
	const truncationSuffix = "\n\n... (メッセージが長すぎるため省略されました)"

	// Markdown整形用の正規表現（流用元からそのまま採用）
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*`)     // **text** -> *text*
	headerRegex := regexp.MustCompile(`(?m)^##\s*(.*)$`) // ## Title -> *Title*
	listItemRegex := regexp.MustCompile(`(?m)^\s*-\s+`)  // - item -> • item

	// 抽出テキストをセクションで分割 (Web抽出後のテキストは通常、全体を一つのセクションとして扱います)
	reviewSections := []string{message}

	for _, sectionText := range reviewSections {
		if len(blocks) >= maxBlocks-2 {
			log.Println("WARNING: Notification message is too long, truncating message.")
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", truncationSuffix, false, false), nil, nil))
			break
		}
		if strings.TrimSpace(sectionText) == "" {
			continue
		}

		// Markdown整形処理
		processedText := sectionText
		processedText = boldRegex.ReplaceAllString(processedText, "*$1*")
		processedText = headerRegex.ReplaceAllString(processedText, "*$1*")
		processedText = listItemRegex.ReplaceAllString(processedText, "• ")

		// 文字数制限の適用
		if len(processedText) > maxSectionLength {
			log.Printf("WARNING: The notification message is too long (%d chars), truncating.", len(processedText))
			processedText = processedText[:maxSectionLength-len(truncationSuffix)] + truncationSuffix
		}

		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", processedText, false, false), nil, nil),
			slack.NewDividerBlock(),
		)
	}

	if len(blocks) > 0 {
		blocks = blocks[:len(blocks)-1] // 最後の余分なDividerを削除
	}

	// フッターには送信時刻を含める
	footerBlock := slack.NewContextBlock(
		"notification-context",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("送信時刻: %s",
			time.Now().Format("2006-01-02 15:04:05")), false, false),
	)
	blocks = append(blocks, footerBlock)

	// --- 2. Webhookメッセージの作成とペイロード準備 ---
	msg := slack.WebhookMessage{
		// プレーンテキストの代替としてヘッダーを使用し、必要に応じてユーザー名とアイコンを上書き
		Text:      headerText,
		Username:  s.Username,
		IconEmoji: s.IconEmoji,
		Channel:   s.Channel,
		Blocks: &slack.Blocks{
			BlockSet: blocks,
		},
	}

	// --- 3. Webhookメッセージの送信（httpkit.PostJSONAndFetchBytesを利用） ---

	// PostJSONAndFetchBytes は、以下の処理を自動で行います。
	// 1. msg を JSON に Marshal する (Marshal失敗はここでエラーを返す)
	// 2. http.NewRequestWithContext で POST リクエストを作成
	// 3. Headerに Content-Type: application/json を設定
	// 4. c.DoRequest を通じてリトライ付きでリクエストを実行
	// 5. 5xx/ネットワークエラーの場合は自動でリトライ
	// 6. 4xx/2xx レスポンスを HandleResponse で処理し、適切なエラーを返すか nil を返す
	respBodyBytes, err := s.client.PostJSONAndFetchBytes(s.WebhookURL, msg, ctx)

	if err != nil {
		// PostJSONAndFetchBytes から返されるエラーは、リトライ後の最終エラーです。
		// エラーには、Marshal失敗、リクエスト作成失敗、リトライ上限到達、または 4xx HTTPエラーが含まれます。

		// SlackのWebhookは成功時に 200 OK を返します。
		// 4xxエラーが返された場合、そのエラーは httpkit.NonRetryableHTTPError にラップされています。

		// 戻り値のエラーをラップして、呼び出し元に Slack 送信のコンテキストを与える
		return fmt.Errorf("Slack Webhookメッセージの送信に失敗しました: %w", err)
	}

	// Slack Webhookは通常、成功時に空のボディ、または "ok" というテキストを返します。
	// respBodyBytes にはそのボディが格納されますが、ここでは利用しないため無視します。
	_ = respBodyBytes

	return nil
}

// SendText は、プレーンテキストメッセージを通知します。（ヘッダーなし）
// Notifier インターフェースを満たすため、SendTextWithHeader にデフォルトヘッダーを付けてフォールバックします。
func (s *SlackNotifier) SendText(ctx context.Context, message string) error {
	header := "📢 通知メッセージ" // デフォルトヘッダー
	if len(message) > 0 {
		firstLine := strings.SplitN(message, "\n", 2)[0]
		if firstLine != "" { // firstLineが空でなければ、それを使用
			if len(firstLine) > 50 { // ヘッダーが長くなりすぎないように調整
				firstLine = firstLine[:50] + "..."
			}
			header = fmt.Sprintf("📢 %s", firstLine)
		}
	}
	return s.SendTextWithHeader(ctx, header, message)
}

// SendIssue は Slack では課題登録機能が標準ではないため、SendTextWithHeaderにフォールバックします。
// 課題の概要をヘッダーとして使用し、課題の詳細をメッセージ本文として送信します。
func (s *SlackNotifier) SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error {
	// summary をヘッダーとして使用し、description を本文として渡す
	header := fmt.Sprintf("【課題】%s", summary)
	return s.SendTextWithHeader(ctx, header, description)
}
