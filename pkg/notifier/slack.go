package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// SlackNotifier は Slack Webhook API と連携するためのクライアントです。
// Notifier インターフェースを満たします。
type SlackNotifier struct {
	// WebhookURL: 必須の通知先URL
	WebhookURL string
	// httpClient: 汎用クライアント (リトライロジックを含む)
	client    httpclient.HTTPClient
	Username  string
	IconEmoji string
	Channel   string
}

// NewSlackNotifier は SlackNotifier の新しいインスタンスを作成します。
func NewSlackNotifier(client httpclient.HTTPClient, webhookURL, username, iconEmoji, channel string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		client:     client,
		Username:   username,
		IconEmoji:  iconEmoji,
		Channel:    channel,
	}
}

// --- Notifier インターフェース実装 ---

// SendText は、プレーンなMarkdownテキストを解析し、SlackのBlock Kit形式で投稿します。
// message は、抽出された本文全体（Markdownとして解釈可能）を想定します。
func (s *SlackNotifier) SendText(ctx context.Context, message string) error {
	// --- 1. Block Kitの構築ロジック（流用元のロジックを汎用化） ---

	// ヘッダーブロックは汎用的なものに変更
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "📢 Web Content Notification", true, false),
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

	// 抽出テキストをセクション（ここでは行）で分割
	// Web抽出後のテキストは通常、セクション区切り（\n---\n）がないため、全体を一つのセクションとして扱います
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

	footerBlock := slack.NewContextBlock(
		"notification-context",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("送信時刻: %s",
			time.Now().Format("2006-01-02 15:04:05")), false, false),
	)
	blocks = append(blocks, footerBlock)

	// --- 2. Webhookメッセージの作成とペイロード準備 ---
	msg := slack.WebhookMessage{
		Text: fmt.Sprintf("Webコンテンツ通知: %s...", strings.Split(message, "\n")[0]), // 最初の行をプレーンテキストの代替として使用
		Blocks: &slack.Blocks{
			BlockSet: blocks,
		},
	}

	jsonPayload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// --- 3. Webhookメッセージの送信（httpclientを利用） ---

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("Slackリクエスト作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 汎用クライアント (リトライロジックを持つ) を利用してリクエスト実行
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Slack WebhookへのHTTPリクエストに失敗しました (ネットワーク/接続エラー): %w", err)
	}
	defer resp.Body.Close()

	// レスポンスステータスのチェック（httpclientのhandleResponseは利用できないため、ここで手動チェック）
	if resp.StatusCode != http.StatusOK {
		// ボディを読み込み、エラーメッセージとして含める（最大1024バイトまで）
		body, _ := httpclient.HandleLimitedResponse(resp, 1024)

		// 5xx やその他のエラーは httpclient.Client のリトライ層で処理されているはずだが、
		// ここでは最終的なエラーとして報告
		return fmt.Errorf("Slack API returned non-OK status code: %d %s, Body: %s",
			resp.StatusCode, resp.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

// SendIssue は Slack では課題登録機能が標準ではないため、SendTextにフォールバックします。
func (s *SlackNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	fullMessage := fmt.Sprintf("*【課題】%s*\n\n%s", summary, description)
	return s.SendText(ctx, fullMessage)
}
