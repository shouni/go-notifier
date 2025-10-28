package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/forPelevin/gomoji"
	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// BacklogNotifier は Backlog API へ課題登録を行うクライアントです。
type BacklogNotifier struct {
	client           *httpclient.Client
	baseURL          string
	apiKey           string
	issueTypeID      int
	priorityID       int
	defaultProjectID int
}

// NewBacklogNotifier は BacklogNotifier のインスタンスを作成します。
func NewBacklogNotifier(client *httpclient.Client, baseURL, apiKey string, issueTypeID, priorityID, defaultProjectID int) *BacklogNotifier {
	return &BacklogNotifier{
		client:           client,
		baseURL:          baseURL,
		apiKey:           apiKey,
		issueTypeID:      issueTypeID,
		priorityID:       priorityID,
		defaultProjectID: defaultProjectID,
	}
}

// SendText は Backlog にテキストを投稿します (SendIssue に委譲)。
func (b *BacklogNotifier) SendText(ctx context.Context, message string) error {
	log.Printf("⚠️ 警告: BacklogNotifier.SendText は BacklogNotifier.SendIssue に委譲されます。")

	lines := strings.SplitN(message, "\n", 2)
	summary := strings.TrimSpace(lines[0])
	description := ""
	if len(lines) > 1 {
		description = strings.TrimSpace(lines[1])
	}

	if summary == "" {
		return fmt.Errorf("投稿メッセージが空です")
	}

	return b.SendIssue(ctx, summary, description, b.defaultProjectID)
}

// SendIssue は Backlog に課題を登録します。
func (b *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	if b.baseURL == "" || b.apiKey == "" {
		log.Println("⚠️ 警告: Backlog設定が不足しているため、投稿処理をスキップします。")
		return nil
	}

	// 1. gomoji を使用した絵文字の除去処理
	cleanedSummary := gomoji.RemoveEmojis(summary)
	cleanedDescription := gomoji.RemoveEmojis(description)

	if strings.TrimSpace(cleanedSummary) == "" {
		return fmt.Errorf("絵文字除去後、課題のサマリーが空になりました")
	}

	// 2. Backlog API へのリクエストペイロード作成
	issueData := map[string]interface{}{
		"projectId":   projectID,
		"summary":     cleanedSummary,
		"description": cleanedDescription,
		"issueTypeId": b.issueTypeID,
		"priorityId":  b.priorityID,
	}

	return b.postInternal(ctx, issueData)
}

func (b *BacklogNotifier) postInternal(ctx context.Context, data map[string]interface{}) error {
	// Backlog API のエンドポイント
	url := fmt.Sprintf("%s/api/v2/issues?apiKey=%s", b.baseURL, b.apiKey)

	jsonBody, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("JSONのマーシャリングに失敗しました: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("APIコールの実行に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResponse struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		// エラーレスポンスのデコードを試みる
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			return fmt.Errorf("Backlog APIエラー (%d): %s", resp.StatusCode, errorResponse.Errors[0].Message)
		}
		return fmt.Errorf("Backlog APIが予期せぬステータスを返しました: %d", resp.StatusCode)
	}

	return nil
}
