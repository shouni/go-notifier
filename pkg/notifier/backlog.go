package notifier

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	// 外部ライブラリとして参照
	"github.com/shouni/go-web-exact/pkg/httpclient"
)

// BacklogIssuePayload はBacklog APIに課題を登録するためのペイロード構造です。
// APIのリクエストパラメータに合わせてフィールドを定義します。
type BacklogIssuePayload struct {
	ProjectId   int    `json:"projectId"`
	Summary     string `json:"summary"`
	IssueTypeId int    `json:"issueTypeId"` // Backlog環境に合わせて適切なIDを設定する必要があります
	PriorityId  int    `json:"priorityId"`  // Backlog環境に合わせて適切なIDを設定する必要があります
	Description string `json:"description,omitempty"`
	// AssigneeId, StartDate, DueDateなど、必要に応じてフィールドを追加
}

// BacklogNotifier はBacklogへの投稿を管理するためのクライアントです。
// Notifier インターフェースを実装します。
type BacklogNotifier struct {
	client  *httpclient.Client
	baseURL string // 例: https://[space_id].backlog.com/api/v2
	apiKey  string // 認証用のAPIキー

	// 環境依存のデフォルト設定
	defaultIssueTypeID int
	defaultPriorityID  int
}

// NewBacklogNotifier は新しい BacklogNotifier のインスタンスを初期化します。
func NewBacklogNotifier(
	httpClient *httpclient.Client,
	baseURL string,
	apiKey string,
	issueTypeID int, // 例: 課題/タスクのデフォルトID
	priorityID int, // 例: 標準/中のデフォルトID
) *BacklogNotifier {
	return &BacklogNotifier{
		client:             httpClient,
		baseURL:            baseURL,
		apiKey:             apiKey,
		defaultIssueTypeID: issueTypeID,
		defaultPriorityID:  priorityID,
	}
}

// postInternal は、ペイロードを作成し、httpclientを使ってBacklog APIにPOSTリクエストを送る内部ヘルパー関数です。
func (b *BacklogNotifier) postInternal(ctx context.Context, payload BacklogIssuePayload) error {
	if b.baseURL == "" || b.apiKey == "" {
		return fmt.Errorf("BacklogのベースURLまたはAPIキーが設定されていません")
	}

	// 課題登録のエンドポイントURLを作成し、APIキーをクエリパラメータとして付与
	apiURL := fmt.Sprintf("%s/issues?apiKey=%s", b.baseURL, url.QueryEscape(b.apiKey))

	// httpclientのPostJSONAndFetchBytesを利用してJSON POSTを実行
	_, err := b.client.PostJSONAndFetchBytes(apiURL, payload, ctx)

	if err != nil {
		// httpclientのエラー判定を活用し、非リトライ対象エラー（4xx）を特定
		if httpclient.IsNonRetryableError(err) {
			// 例: HTTP 404 (エンドポイント不正)、400 (リクエストパラメータ不正) など
			return fmt.Errorf("Backlogへの課題登録に失敗しました (APIエラー): %w", err)
		}
		// その他 (5xx やネットワークエラー) はリトライ上限到達のエラー
		return fmt.Errorf("Backlogへの課題登録に失敗しました: %w", err)
	}

	return nil
}

// SendText は Notifier インターフェースを実装しますが、
// Backlogは課題トラッカーであるため、テキストのみの送信は課題として扱います。
// Backlogは主に構造化されたデータ（課題）を扱うため、このメソッドは簡易的な課題登録として実装します。
func (b *BacklogNotifier) SendText(ctx context.Context, message string) error {
	// メッセージをサマリーと説明に分割して課題として登録
	//summary := message
	//description := ""

	// 簡略化のため、最初の1行をサマリー、残りを説明とする（改行がない場合は全体をサマリーとする）
	lines := strings.SplitN(message, "\n", 2)
	if len(lines) == 2 {
		summary = strings.TrimSpace(lines[0])
		description = strings.TrimSpace(lines[1])
	}

	// 課題ペイロードを作成（projectIdは設定がないため0、APIでエラーとなる可能性あり）
	//payload := BacklogIssuePayload{
	//	ProjectId:   b.defaultIssueTypeID, // このIDは実際にはprojectIdではないため、注意が必要。通常はSendIssueを使う。
	//	Summary:     summary,
	//	IssueTypeId: b.defaultIssueTypeID,
	//	PriorityId:  b.defaultPriorityID,
	//	Description: description,
	//}

	// 課題登録は本来プロジェクトIDが必須のため、このメソッドは推奨されないが、インターフェースを満たすために実装
	return fmt.Errorf("BacklogNotifier.SendText は、プロジェクトIDがないため使用できません。SendIssueを使用してください。")
}

// SendIssue は Notifier インターフェースを実装し、課題情報を受け取ってBacklogに課題として投稿します。
func (b *BacklogNotifier) SendIssue(ctx context.Context, summary, description string, projectID int) error {
	if projectID <= 0 {
		return fmt.Errorf("Backlogへの課題登録には有効なプロジェクトIDが必要です")
	}

	// 課題ペイロードを作成
	payload := BacklogIssuePayload{
		ProjectId:   projectID,
		Summary:     summary,
		IssueTypeId: b.defaultIssueTypeID, // 初期化時に設定されたデフォルト値を使用
		PriorityId:  b.defaultPriorityID,  // 初期化時に設定されたデフォルト値を使用
		Description: description,
	}

	return b.postInternal(ctx, payload)
}
