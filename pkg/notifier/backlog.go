package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/shouni/go-utils/text"
	request "github.com/shouni/go-web-exact/v2/pkg/client"
)

// BacklogNotifier は Backlog 課題登録用の API クライアントです。
// Notifier インターフェースを満たしますが、SendText および SendTextWithHeader は Backlog の利用方針（課題登録推奨）に基づきエラーを返します。
type BacklogNotifier struct {
	client  request.Client // 汎用クライアント (リトライ機能込み)
	baseURL string
	apiKey  string
}

// BacklogProjectResponse はプロジェクトキーまたはIDで取得した際のレスポンスを扱います。
type BacklogProjectResponse struct {
	ID   int    `json:"id"`
	Key  string `json:"projectKey"`
	Name string `json:"name"`
}

// BacklogIssueTypeResponse は課題種別の最小限の構造体です。
type BacklogIssueTypeResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BacklogPriorityResponse は優先度の最小限の構造体です。
type BacklogPriorityResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
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
func NewBacklogNotifier(client request.Client, spaceURL string, apiKey string) (*BacklogNotifier, error) {
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

// GetProjectID は、プロジェクトキー（文字列）を受け取り、プロジェクトID（整数）を取得します。
func (c *BacklogNotifier) GetProjectID(ctx context.Context, projectKey string) (int, error) {
	if projectKey == "" {
		return 0, errors.New("プロジェクトIDまたはキーは空にできません")
	}
	endpoint := fmt.Sprintf("/projects/%s", projectKey)
	fullURL := fmt.Sprintf("%s%s?apiKey=%s", c.baseURL, endpoint, c.apiKey)

	data, err := c.client.FetchBytes(fullURL, ctx)
	if err != nil {
		// FetchBytes がすでにリトライ済みのため、そのままエラーを返す
		return 0, fmt.Errorf("Backlog APIへのプロジェクト情報取得リクエストに失敗: %w", err)
	}

	// 3. JSONのパース
	var projectResp BacklogProjectResponse
	if err := json.Unmarshal(data, &projectResp); err != nil {
		return 0, fmt.Errorf("プロジェクト情報レスポンスのパースに失敗しました (データ: %s): %w", string(data), err)
	}

	// 4. IDのチェック
	if projectResp.ID == 0 {
		// APIが200 OKを返したがIDがない場合（通常は発生しないが安全のため）
		return 0, fmt.Errorf("BacklogからプロジェクトIDを取得できませんでした (キー: %s)", projectKey)
	}

	return projectResp.ID, nil
}

// getFirstIssueAttributes は、指定されたプロジェクトの最初の有効な IssueTypeID と PriorityID を取得します。
func (c *BacklogNotifier) getFirstIssueAttributes(ctx context.Context, projectID int) (issueTypeID int, priorityID int, err error) {
	// 1. 課題種別 (Issue Types) の取得
	// エンドポイント: /projects/{projectId}/issueTypes
	issueTypeURL := fmt.Sprintf("%s/projects/%d/issueTypes?apiKey=%s", c.baseURL, projectID, c.apiKey)
	issueTypeData, fetchErr := c.client.FetchBytes(issueTypeURL, ctx)
	if fetchErr != nil {
		return 0, 0, fmt.Errorf("課題種別リストの取得に失敗: %w", fetchErr)
	}

	var issueTypes []BacklogIssueTypeResponse
	if err := json.Unmarshal(issueTypeData, &issueTypes); err != nil {
		return 0, 0, fmt.Errorf("課題種別リストのパースに失敗しました (ProjectID: %d): %w", projectID, err)
	}

	// 💡 修正ロジック: "タスク" を優先して探す
	foundIssueTypeID := 0
	for _, it := range issueTypes {
		if it.Name == "タスク" { // あるいは設定可能なデフォルト値
			foundIssueTypeID = it.ID
			break
		}
	}
	if foundIssueTypeID == 0 && len(issueTypes) > 0 {
		foundIssueTypeID = issueTypes[0].ID // 見つからなければ最初のものをデフォルトとする
	}
	if foundIssueTypeID == 0 {
		return 0, 0, fmt.Errorf("プロジェクトの課題種別が見つかりませんでした (ProjectID: %d)", projectID)
	}
	issueTypeID = foundIssueTypeID // 採用

	// 2. 優先度 (Priorities) の取得
	// エンドポイント: /priorities (優先度はプロジェクト共通だが、念のため取得)
	priorityURL := fmt.Sprintf("%s/priorities?apiKey=%s", c.baseURL, c.apiKey)
	priorityData, fetchErr := c.client.FetchBytes(priorityURL, ctx)
	if fetchErr != nil {
		return 0, 0, fmt.Errorf("優先度リストの取得に失敗: %w", fetchErr)
	}

	var priorities []BacklogPriorityResponse
	if err := json.Unmarshal(priorityData, &priorities); err != nil {
		return 0, 0, fmt.Errorf("優先度リストのパースに失敗しました: %w", err)
	}

	// 💡 修正ロジック: "中" を優先して探す
	foundPriorityID := 0
	for _, p := range priorities {
		if p.Name == "中" { // あるいは設定可能なデフォルト値
			foundPriorityID = p.ID
			break
		}
	}
	if foundPriorityID == 0 && len(priorities) > 0 {
		foundPriorityID = priorities[0].ID // 見つからなければ最初のものをデフォルトとする
	}
	if foundPriorityID == 0 {
		return 0, 0, fmt.Errorf("優先度が見つかりませんでした")
	}
	priorityID = foundPriorityID // 採用

	return issueTypeID, priorityID, nil
}

// SendIssue は、Backlogに新しい課題を登録します。
// func (c *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID, issueTypeID, priorityID int) error {
func (c *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {

	// 1. 絵文字のサニタイズ
	sanitizedSummary := text.CleanStringFromEmojis(summary)
	sanitizedDescription := text.CleanStringFromEmojis(description)

	// 有効な ID を取得
	validIssueTypeID, validPriorityID, err := c.getFirstIssueAttributes(ctx, projectID)
	if err != nil {
		return fmt.Errorf("プロジェクトの有効な課題属性の取得に失敗: %w", err)
	}

	// 2. ペイロードの構築
	issueData := BacklogIssuePayload{
		ProjectID:   projectID,
		Summary:     sanitizedSummary,
		Description: sanitizedDescription,
		IssueTypeID: validIssueTypeID,
		PriorityID:  validPriorityID,
	}

	jsonBody, err := json.Marshal(issueData)
	if err != nil {
		return fmt.Errorf("failed to marshal issue data: %w", err)
	}

	// 3. APIリクエストの実行
	err = c.postRequest(ctx, "/issues", jsonBody)
	if err != nil {
		// エラーを呼び出し元に返す
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
	sanitizedContent := text.CleanStringFromEmojis(content)

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
	fullURL := fmt.Sprintf("%s%s?apiKey=%s", c.baseURL, endpoint, c.apiKey)

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
	body, _ := request.HandleLimitedResponse(resp, 4096) // 4KBまで読み込み

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
