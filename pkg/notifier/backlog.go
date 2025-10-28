package notifier

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// BacklogIssuePayload はBacklog APIに課題を登録するためのペイロード構造です。
type BacklogIssuePayload struct {
	ProjectId   int    `json:"projectId"`
	Summary     string `json:"summary"`
	IssueTypeId int    `json:"issueTypeId"`
	PriorityId  int    `json:"priorityId"`
	Description string `json:"description,omitempty"`
}

// BacklogNotifier はBacklogへの投稿を管理するためのクライアントです。
type BacklogNotifier struct {
	client  *httpclient.Client
	baseURL string
	apiKey  string

	defaultIssueTypeID int
	defaultPriorityID  int
	targetProjectID    int // SendTextで利用するデフォルトのプロジェクトIDを保持
}

// NewBacklogNotifier は新しい BacklogNotifier のインスタンスを初期化します。
// targetProjectID は SendText の実装を可能にするために必要です。
func NewBacklogNotifier(
	httpClient *httpclient.Client,
	baseURL string,
	apiKey string,
	issueTypeID int,
	priorityID int,
	targetProjectID int, // 新規追加
) *BacklogNotifier {
	return &BacklogNotifier{
		client:             httpClient,
		baseURL:            baseURL,
		apiKey:             apiKey,
		defaultIssueTypeID: issueTypeID,
		defaultPriorityID:  priorityID,
		targetProjectID:    targetProjectID,
	}
}

// postInternal は、httpclientを使ってBacklog APIにPOSTリクエストを送る内部ヘルパー関数です。
func (b *BacklogNotifier) postInternal(ctx context.Context, payload BacklogIssuePayload) error {
	if b.baseURL == "" || b.apiKey == "" {
		return nil // API設定がない場合、エラーではなくスキップとして扱う
	}

	apiURL := fmt.Sprintf("%s/issues?apiKey=%s", b.baseURL, url.QueryEscape(b.apiKey))

	_, err := b.client.PostJSONAndFetchBytes(apiURL, payload, ctx)

	if err != nil {
		if httpclient.IsNonRetryableError(err) {
			return fmt.Errorf("Backlogへの課題登録に失敗しました (APIエラー): %w", err)
		}
		return fmt.Errorf("Backlogへの課題登録に失敗しました: %w", err)
	}

	return nil
}

// SendText は Notifier インターフェースを実装し、簡易的な課題として登録します。
func (b *BacklogNotifier) SendText(ctx context.Context, message string) error {
	if b.targetProjectID <= 0 {
		return fmt.Errorf("BacklogNotifier.SendText は、有効なデフォルトプロジェクトIDが設定されていないため実行できません。")
	}

	// メッセージをサマリーと説明に分割して課題として登録
	summary := message
	description := ""

	// 最初の1行をサマリー、残りを説明とする
	lines := strings.SplitN(message, "\n", 2)
	if len(lines) == 2 {
		summary = strings.TrimSpace(lines[0])
		description = strings.TrimSpace(lines[1])
	}

	// SendIssueに委譲し、初期化時に設定されたデフォルトのプロジェクトIDを使用する
	return b.SendIssue(ctx, summary, description, b.targetProjectID)
}

// SendIssue は Notifier インターフェースを実装し、課題情報を受け取ってBacklogに課題として投稿します。
func (b *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	if projectID <= 0 {
		return fmt.Errorf("Backlogへの課題登録には有効なプロジェクトIDが必要です")
	}

	payload := BacklogIssuePayload{
		ProjectId:   projectID,
		Summary:     summary,
		IssueTypeId: b.defaultIssueTypeID,
		PriorityId:  b.defaultPriorityID,
		Description: description,
	}

	return b.postInternal(ctx, payload)
}
