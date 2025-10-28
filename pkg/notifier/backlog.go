package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forPelevin/gomoji"
	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// BacklogClient is the API client for Backlog.
// NOTE: BacklogClient から BacklogNotifier に名称を変更
type BacklogNotifier struct {
	client  httpclient.HTTPClient // 汎用クライアント (リトライ機能込み)
	baseURL string                // 例: https://your-space.backlog.jp/api/v2
	apiKey  string
}

// BacklogIssuePayload は課題登録API (/issues) に必要なペイロードです。
type BacklogIssuePayload struct {
	ProjectID   int    `json:"projectId"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	// Backlogの課題登録に必要なその他のフィールド (IssueTypeID, PriorityIDなど) は、
	// CLI側で環境変数やフラグから受け取り、ここに含める必要があります。
	// 今回は簡易化のため省略しますが、実際の運用では必須です。
}

// BacklogErrorResponse はBacklog APIが返す一般的なエラー構造体です。
type BacklogErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errors"`
}

// BacklogError はBacklog APIから返されるエラーを表すカスタムエラーです。
type BacklogError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *BacklogError) Error() string {
	return fmt.Sprintf("Backlog API error (status %d, code %d): %s", e.StatusCode, e.Code, e.Message)
}

// cleanStringFromEmojis は、文字列から絵文字を削除します。
func cleanStringFromEmojis(s string) string {
	return gomoji.RemoveEmojis(s)
}

// NewBacklogNotifier はBacklogNotifierを初期化します。
// 💡 HTTPClient を受け取るように変更
func NewBacklogNotifier(client httpclient.HTTPClient, spaceURL string, apiKey string) (*BacklogNotifier, error) {
	if spaceURL == "" || apiKey == "" {
		return nil, errors.New("BACKLOG_SPACE_URL および BACKLOG_API_KEY の設定が必要です")
	}

	trimmedURL := strings.TrimRight(spaceURL, "/")
	// /api/v2 は APIキーがクエリパラメータで渡されるため、ベースURLには含めない設計も可能ですが、
	// 流用元の設計に合わせてベースURLに含めます。
	apiURL := trimmedURL + "/api/v2"

	return &BacklogNotifier{
		client:  client,
		baseURL: apiURL,
		apiKey:  apiKey,
	}, nil
}

// --- Notifier インターフェース実装 ---

// SendIssue は、Backlogに新しい課題を登録します。
func (c *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	// 1. 絵文字のサニタイズ
	sanitizedSummary := cleanStringFromEmojis(summary)
	sanitizedDescription := cleanStringFromEmojis(description)

	// 2. ペイロードの構築
	issueData := BacklogIssuePayload{
		ProjectID:   projectID,
		Summary:     sanitizedSummary,
		Description: sanitizedDescription,
	}

	jsonBody, err := json.Marshal(issueData)
	if err != nil {
		return fmt.Errorf("failed to marshal issue data: %w", err)
	}

	// 3. APIリクエストの実行
	// 💡 postCommentAttempt の代わりに汎用的な postRequest を利用
	err = c.postRequest(ctx, "/issues", jsonBody)
	if err != nil {
		return fmt.Errorf("failed to create issue in Backlog: %w", err)
	}

	fmt.Printf("✅ Backlog issue successfully created (ProjectID: %d).\n", projectID)
	return nil
}

// SendText は Backlog では課題登録を推奨するため、SendIssue にフォールバックさせます。
// 課題IDが必須であるため、このメソッドはここでは実装しません。
// CLIでテキスト投稿が必要な場合は、SendIssue を呼び出すようにします。
func (c *BacklogNotifier) SendText(ctx context.Context, message string) error {
	// Notifier インターフェースを満たすため、メッセージのみを渡された場合はエラーとする。
	// Backlogの性質上、課題IDがないと投稿できないため。
	return errors.New("BacklogNotifier cannot send plain text; use SendIssue with a project ID instead")
}

// postRequest は、指定されたエンドポイントへリクエストを送信する内部ヘルパーメソッドです。
// リトライは c.client (httpclient.Client) に委譲されます。
func (c *BacklogNotifier) postRequest(ctx context.Context, endpoint string, jsonBody []byte) error {
	// apiKey をクエリパラメータに追加
	fullURL := fmt.Sprintf("%s%s?apiKey=%s", c.baseURL, endpoint, c.apiKey)

	// 💡 context.Context を使用してリクエストを作成
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create POST request for Backlog: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 汎用クライアント c.client (リトライ機能込み) を使用して実行
	resp, err := c.client.Do(req)
	if err != nil {
		// ネットワークエラーなどは c.client がリトライした後で返ってくる
		return fmt.Errorf("failed to send POST request to Backlog (after retries): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// エラーレスポンスの処理
	// 💡 httpclient.HandleLimitedResponse を利用してボディを安全に読み込み
	body, _ := httpclient.HandleLimitedResponse(resp, 4096) // 4KBまで読み込み

	var errorResp BacklogErrorResponse
	if json.Unmarshal(body, &errorResp) == nil && len(errorResp.Errors) > 0 {
		firstError := errorResp.Errors[0]

		return &BacklogError{
			StatusCode: resp.StatusCode,
			Code:       firstError.Code,
			Message:    firstError.Message,
		}
	}

	return &BacklogError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
	}
}
