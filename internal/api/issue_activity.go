package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// IssueActor is the public identity returned by Issue audit endpoints.
// Only fields shared by the operation-log and history/reaction schemas are used.
type IssueActor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// IssueActivityRef describes the PR branch snapshot attached to an audit event.
type IssueActivityRef struct {
	Ref      string                   `json:"ref"`
	SHA      string                   `json:"sha"`
	Repo     *IssueActivityRepository `json:"repo"`
	Assigner *IssueActor              `json:"assigner"`
}

type IssueActivityRepository struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type IssueActivity struct {
	ID         int64             `json:"id"`
	User       IssueActor        `json:"user"`
	Content    string            `json:"content"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"update_at"`
	ActionType string            `json:"action_type"`
	IssueID    string            `json:"issue_id"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Head       *IssueActivityRef `json:"head"`
	Base       *IssueActivityRef `json:"base"`
}

type IssueHistory struct {
	ID          string     `json:"id"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	Deleted     bool       `json:"deleted"`
	Created     bool       `json:"created"`
	Content     string     `json:"content"`
	User        IssueActor `json:"user"`
	UpdatedUser IssueActor `json:"updated_user"`
}

type IssueReaction struct {
	ID        string     `json:"id"`
	Emoji     string     `json:"emoji"`
	EmojiName string     `json:"emoji_name"`
	User      IssueActor `json:"user"`
}

// Validate each record while decoding, before pagination or --limit can discard
// elements. In particular, JSON null must not become a zero-valued audit event.
func (item *IssueActivity) UnmarshalJSON(data []byte) error {
	type wire IssueActivity
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.ID <= 0 {
		return fmt.Errorf("invalid issue activity record: id must be positive")
	}
	*item = IssueActivity(decoded)
	return nil
}

func (item *IssueHistory) UnmarshalJSON(data []byte) error {
	type wire IssueHistory
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return fmt.Errorf("invalid issue history record: id must be non-empty")
	}
	*item = IssueHistory(decoded)
	return nil
}

func (item *IssueReaction) UnmarshalJSON(data []byte) error {
	type wire IssueReaction
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return fmt.Errorf("invalid issue reaction record: id must be non-empty")
	}
	*item = IssueReaction(decoded)
	return nil
}

// ListIssueActivity uses the owner-scoped endpoint with the required repo query.
// The documented activity and history endpoints have no pagination parameters;
// their limits are applied locally without inventing page requests.
func ListIssueActivity(client *Client, owner, repo, number string, limit int) ([]IssueActivity, error) {
	path := fmt.Sprintf("/repos/%s/issues/%s/operate_logs?%s", url.PathEscape(owner), url.PathEscape(number), url.Values{"repo": {repo}}.Encode())
	return issueAuditList[IssueActivity](client, path, limit)
}

func ListIssueHistory(client *Client, owner, repo, number string, limit int) ([]IssueHistory, error) {
	return issueAuditList[IssueHistory](client, issueAuditPath(owner, repo, number, "modify_history"), limit)
}

func ListIssueReactions(client *Client, owner, repo, number string, limit int) ([]IssueReaction, error) {
	path := issueAuditPath(owner, repo, number, "user_reactions")
	return GetPaginated[IssueReaction](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", path, page, perPage)
	})
}

func issueAuditPath(owner, repo, number, endpoint string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(number), endpoint)
}

func issueAuditList[T any](client *Client, path string, limit int) ([]T, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	items := make([]T, 0)
	if err := client.Get(path, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]T, 0)
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
