package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// SlackMessage はSlackのIncoming Webhookペイロードの構造を定義します。
type SlackMessage struct {
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// SlackNotifier はSlackへの通知を管理するためのクライアントです。
type SlackNotifier struct {
	client     *httpclient.Client
	webhookURL string
	username   string
	iconEmoji  string
	channel    string
}

// NewSlackNotifier は新しい SlackNotifier のインスタンスを初期化します。
func NewSlackNotifier(httpClient *httpclient.Client, webhookURL, username, iconEmoji, channel string) *SlackNotifier {
	return &SlackNotifier{
		client:     httpClient,
		webhookURL: webhookURL,
		username:   username,
		iconEmoji:  iconEmoji,
		channel:    channel,
	}
}

// postInternal は、ペイロードを作成し、httpclientを使ってSlackにPOSTリクエストを送る内部ヘルパー関数です。
func (s *SlackNotifier) postInternal(ctx context.Context, message SlackMessage) error {
	if s.webhookURL == "" {
		// Webhook URLがない場合、エラーではなくスキップとして扱う
		return nil
	}

	// postInternalに入る前に payload にデフォルト値を設定する

	// httpclientのPostJSONAndFetchBytesを利用してJSON POSTを実行
	_, err := s.client.PostJSONAndFetchBytes(s.webhookURL, message, ctx)

	if err != nil {
		if httpclient.IsNonRetryableError(err) {
			return fmt.Errorf("Slackへのメッセージ投稿に失敗しました (Webhook設定エラーかペイロード不正): %w", err)
		}
		return fmt.Errorf("Slackへのメッセージ投稿に失敗しました: %w", err)
	}

	return nil
}

// SendText は Notifier インターフェースを実装し、指定されたテキストメッセージをSlackに投稿します。
func (s *SlackNotifier) SendText(ctx context.Context, message string) error {
	payload := SlackMessage{
		Text:      message,
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Channel:   s.channel,
	}
	return s.postInternal(ctx, payload)
}

// SendIssue は Notifier インターフェースを実装し、課題情報をSlackのテキスト形式に変換して投稿します。
func (s *SlackNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(":pushpin: *【課題通知】* %s\n", summary))
	sb.WriteString(fmt.Sprintf(":clipboard: プロジェクトID: `%d`\n", projectID))
	sb.WriteString("-------------------------------------------\n")
	sb.WriteString(description)

	payload := SlackMessage{
		Text:      sb.String(),
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Channel:   s.channel,
	}
	return s.postInternal(ctx, payload)
}
