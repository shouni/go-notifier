package notifier

import (
	"context"
	"fmt"
	"strings"

	// 外部ライブラリとして参照
	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// SlackMessage はSlackのIncoming Webhookペイロードの構造を定義します。
type SlackMessage struct {
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	Channel   string `json:"channel,omitempty"`
	// 必要に応じて、AttachmentsやBlocksなどのフィールドを追加できます。
}

// SlackNotifier はSlackへの通知を管理するためのクライアントです。
// Notifier インターフェースを実装します。
type SlackNotifier struct {
	client     *httpclient.Client
	webhookURL string
}

// NewSlackNotifier は新しい SlackNotifier のインスタンスを初期化します。
func NewSlackNotifier(httpClient *httpclient.Client, webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		client:     httpClient,
		webhookURL: webhookURL,
	}
}

// postInternal は、ペイロードを作成し、httpclientを使ってSlackにPOSTリクエストを送る内部ヘルパー関数です。
func (s *SlackNotifier) postInternal(ctx context.Context, message SlackMessage) error {
	if s.webhookURL == "" {
		return fmt.Errorf("Slack Webhook URLが設定されていません")
	}

	// httpclientのPostJSONAndFetchBytesを利用してJSON POSTを実行
	_, err := s.client.PostJSONAndFetchBytes(s.webhookURL, message, ctx)

	if err != nil {
		// httpclientのエラー判定を活用し、非リトライ対象エラー（4xx）を特定
		if httpclient.IsNonRetryableError(err) {
			return fmt.Errorf("Slackへのメッセージ投稿に失敗しました (Webhook設定エラーかペイロード不正): %w", err)
		}
		// その他 (5xx やネットワークエラー) はリトライ上限到達のエラー
		return fmt.Errorf("Slackへのメッセージ投稿に失敗しました: %w", err)
	}

	return nil
}

// SendText は Notifier インターフェースを実装し、指定されたテキストメッセージをSlackに投稿します。
func (s *SlackNotifier) SendText(ctx context.Context, message string) error {
	// 投稿するペイロードを作成
	payload := SlackMessage{
		Text: message,
	}
	return s.postInternal(ctx, payload)
}

// SendIssue は Notifier インターフェースを実装し、課題情報をSlackのテキスト形式に変換して投稿します。
// Slackは課題トラッカーではないため、課題情報をテキストに整形して処理します。
func (s *SlackNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	var sb strings.Builder

	// BacklogのProjectIDはSlackでは直接利用しないため、情報として記載
	sb.WriteString(fmt.Sprintf(":pushpin: *【課題通知】* %s\n", summary))
	sb.WriteString(fmt.Sprintf(":clipboard: プロジェクトID: `%d`\n", projectID))
	sb.WriteString("-------------------------------------------\n")
	sb.WriteString(description)

	// 投稿するペイロードを作成
	payload := SlackMessage{
		Text: sb.String(),
	}
	return s.postInternal(ctx, payload)
}
