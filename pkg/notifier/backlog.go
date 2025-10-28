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

// BacklogNotifier は Backlog 課題登録用の API クライアントです。
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

// cleanStringFromEmojis は、文字列から絵文字を削除します。
func cleanStringFromEmojis(s string) string {
	return gomoji.RemoveEmojis(s)
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
	sanitizedSummary := cleanStringFromEmojis(summary)
	sanitizedDescription := cleanStringFromEmojis(description)

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

// SendText は Backlog では課題登録を推奨するため、SendIssue にフォールバックさせます。
func (c *BacklogNotifier) SendText(ctx context.Context, message string) error {
	return errors.New("BacklogNotifier cannot send plain text; use SendIssue with a project ID and issue details instead")
}

// postRequest は、指定されたエンドポイントへリクエストを送信する内部ヘルパーメソッドです。
func (c *BacklogNotifier) postRequest(ctx context.Context, endpoint string, jsonBody []byte) error {
	// 💡 修正: apiKey をURLから削除し、ヘッダーに設定 (セキュリティ向上)
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
