package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/shouni/go-web-exact/pkg/httpclient"
	"go-notifier/pkg/util"
)

// BacklogNotifier は Backlog 課題登録用の API クライアントです。
// Notifier インターフェースを満たしますが、SendText および SendTextWithHeader は Backlog の利用方針（課題登録推奨）に基づきエラーを返します。
type BacklogNotifier struct {
	client  httpclient.HTTPClient // 汎用クライアント (リトライ機能込み)
	baseURL string
	apiKey  string
}

// BacklogIssuePayload は課題登録API (/issues) に必要なペイロードです。
type BacklogIssuePayload struct {
	ProjectID   int    `json:"projectId"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	IssueTypeID int    `json:"issueTypeId"` // 必須
	PriorityID  int    `json:"priorityId"`  // 必須
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

// NewBacklogNotifier はBacklogNotifierを初期化します。
func NewBacklogNotifier(client httpclient.HTTPClient, spaceURL string, apiKey string) (*BacklogNotifier, error) {
	if spaceURL == "" || apiKey == "" {
		return nil, errors.New("BACKLOG_SPACE_URL および BACKLOG_API_KEY の設定が必要です")
	}

	trimmedURL := strings.TrimRight(spaceURL, "/")
	// /api/v2 の二重化を防止
	trimmedURL = strings.TrimSuffix(trimmedURL, "/api/v2")
	apiURL := trimmedURL + "/api/v2"

	return &BacklogNotifier{
		client:  client,
		baseURL: apiURL,
		apiKey:  apiKey,
	}, nil
}

// --- Notifier インターフェース実装 ---

// SendIssue は、Backlogに新しい課題を登録します。
func (c *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error {
	// 1. 絵文字のサニタイズ
	sanitizedSummary := util.CleanStringFromEmojis(summary)         // 修正: 大文字始まりの関数を呼び出し
	sanitizedDescription := util.CleanStringFromEmojis(description) // 修正: 大文字始まりの関数を呼び出し

	// 2. ペイロードの構築
	issueData := BacklogIssuePayload{
		ProjectID:   projectID,
		Summary:     sanitizedSummary,
		Description: sanitizedDescription,
		IssueTypeID: issueTypeID,
		PriorityID:  priorityID,
	}

	jsonBody, err := json.Marshal(issueData)
	if err != nil {
		return fmt.Errorf("failed to marshal issue data: %w", err)
	}

	// 3. APIリクエストの実行
	err = c.postRequest(ctx, "/issues", jsonBody)
	if err != nil {
		return fmt.Errorf("failed to create issue in Backlog: %w", err)
	}

	fmt.Printf("✅ Backlog issue successfully created (ProjectID: %d).\n", projectID)
	return nil
}

// SendText は Backlog では課題登録を推奨するため、エラーを返します。
// Notifier インターフェース (ヘッダーなし) を満たすための実装です。
func (c *BacklogNotifier) SendText(ctx context.Context, message string) error {
	return errors.New("BacklogNotifier: Plain text notification is not supported; use SendIssue or PostComment")
}

// SendTextWithHeader は Backlog では課題登録を推奨するため、エラーを返します。
// Notifier インターフェース (ヘッダーあり) を満たすための実装です。
func (c *BacklogNotifier) SendTextWithHeader(ctx context.Context, headerText string, message string) error {
	return errors.New("BacklogNotifier: Plain text notification is not supported; use SendIssue or PostComment")
}

// --- コメント投稿機能の追加 ---

// PostComment は指定された課題IDにコメントを投稿します。
func (c *BacklogNotifier) PostComment(ctx context.Context, issueID string, content string) error {
	if issueID == "" {
		return errors.New("issueID cannot be empty for posting a comment")
	}

	// 1. 絵文字のサニタイズ (Backlogの制限対策)
	sanitizedContent := util.CleanStringFromEmojis(content) // 修正: 大文字始まりの関数を呼び出し

	// 2. ペイロードの構築
	commentData := map[string]string{
		"content": sanitizedContent,
	}
	jsonBody, err := json.Marshal(commentData)
	if err != nil {
		return fmt.Errorf("failed to marshal comment data: %w", err)
	}

	// 3. APIリクエストの実行
	// エンドポイント: /issues/{issueIdOrKey}/comments
	endpoint := fmt.Sprintf("/issues/%s/comments", issueID)

	// c.postRequest を使用してコメントを投稿
	err = c.postRequest(ctx, endpoint, jsonBody)
	if err != nil {
		return fmt.Errorf("failed to post comment to Backlog issue %s: %w", issueID, err)
	}

	fmt.Printf("✅ Backlog issue %s successfully commented.\n", issueID)
	return nil
}

// postRequest は、指定されたエンドポイントへリクエストを送信する内部ヘルパーメソッドです。
func (c *BacklogNotifier) postRequest(ctx context.Context, endpoint string, jsonBody []byte) error {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create POST request for Backlog: %w", err)
	}

	// APIキーをヘッダーに追加
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Backlog-Api-Key", c.apiKey)

	// 汎用クライアント c.client (リトライ機能込み) を使用して実行
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request to Backlog (after retries): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// エラーレスポンスの処理
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
