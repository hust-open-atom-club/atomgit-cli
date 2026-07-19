package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Repository represents an AtomGit repository
type Repository struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"web_url"`
	Private         bool   `json:"private"`
	Internal        bool   `json:"internal"`
	DefaultBranch   string `json:"default_branch"`
	Language        string `json:"language"`
	License         string `json:"license"`
	Fork            bool   `json:"fork"`
	ParentFullName  string `json:"parentfull_name"`
	UpdatedAt       string `json:"updated_at"`
	StarsCount      int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	WatchersCount   int    `json:"watchers_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
	Owner           struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
}

// PullRequest represents an AtomGit pull request
type PullRequest struct {
	ID        int64       `json:"id"`
	Number    interface{} `json:"number"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	State     string      `json:"state"`
	HTMLURL   string      `json:"html_url"`
	User      User        `json:"user"`
	Head      Branch      `json:"head"`
	Base      Branch      `json:"base"`
	Labels    []Label     `json:"labels"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Merged    bool        `json:"merged"`
	Mergeable bool        `json:"mergeable"`
}

// PullRequestReviewEvent is an action submitted as a formal pull request review.
type PullRequestReviewEvent string

const (
	PullRequestReviewApprove        PullRequestReviewEvent = "APPROVE"
	PullRequestReviewRequestChanges PullRequestReviewEvent = "REQUEST_CHANGES"
	PullRequestReviewComment        PullRequestReviewEvent = "COMMENT"
)

// PullRequestReviewRequest represents a formal pull request review submission.
type PullRequestReviewRequest struct {
	Body  string                 `json:"body,omitempty"`
	Event PullRequestReviewEvent `json:"event"`
}

// PullRequestReview represents the response from a formal review submission.
// AtomGit may return an empty success response, so all fields are optional.
type PullRequestReview struct {
	ID      string                 `json:"id,omitempty"`
	Body    string                 `json:"body,omitempty"`
	Event   PullRequestReviewEvent `json:"event,omitempty"`
	State   string                 `json:"state,omitempty"`
	HTMLURL string                 `json:"html_url,omitempty"`
}

// GetNumber returns the PR number as a string
func (pr *PullRequest) GetNumber() string {
	switch v := pr.Number.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Issue represents an AtomGit issue
type Issue struct {
	ID        int64       `json:"id"`
	Number    interface{} `json:"number"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	State     string      `json:"state"`
	HTMLURL   string      `json:"html_url"`
	User      User        `json:"user"`
	Labels    []Label     `json:"labels"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

// GetNumber returns the Issue number as a string
func (i *Issue) GetNumber() string {
	switch v := i.Number.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// User represents an AtomGit user
type User struct {
	ID      string `json:"id"`
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	HTMLURL string `json:"html_url"`
	Type    string `json:"type"`
}

// FlexibleBool decodes AtomGit boolean metadata that may be returned as
// JSON booleans, integers, or strings depending on the endpoint.
type FlexibleBool bool

func (b *FlexibleBool) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		*b = false
		return nil
	}

	var boolValue bool
	if err := json.Unmarshal(data, &boolValue); err == nil {
		*b = FlexibleBool(boolValue)
		return nil
	}

	var numberValue int
	if err := json.Unmarshal(data, &numberValue); err == nil {
		*b = FlexibleBool(numberValue != 0)
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		parsed, err := strconv.ParseBool(stringValue)
		if err == nil {
			*b = FlexibleBool(parsed)
			return nil
		}
		if numeric, err := strconv.Atoi(stringValue); err == nil {
			*b = FlexibleBool(numeric != 0)
			return nil
		}
	}

	return fmt.Errorf("invalid boolean value %q", string(data))
}

func (b FlexibleBool) Bool() bool {
	return bool(b)
}

// Branch represents a git branch
type Branch struct {
	Ref                string       `json:"ref"`
	SHA                string       `json:"sha"`
	Repo               Repository   `json:"repo"`
	Name               string       `json:"name"`
	Commit             BranchCommit `json:"commit"`
	Protected          FlexibleBool `json:"protected"`
	Default            FlexibleBool `json:"default"`
	Merged             FlexibleBool `json:"merged"`
	DevelopersCanPush  FlexibleBool `json:"developers_can_push"`
	DevelopersCanMerge FlexibleBool `json:"developers_can_merge"`
	CanPush            FlexibleBool `json:"can_push"`
	CreatedAt          string       `json:"created_at"`
	Creator            User         `json:"creator"`
}

// BranchCommit represents the latest commit summary attached to a branch.
type BranchCommit struct {
	ID                 string   `json:"id"`
	SHA                string   `json:"sha"`
	ShortID            string   `json:"short_id"`
	URL                string   `json:"url"`
	Message            string   `json:"message"`
	Title              string   `json:"title"`
	ParentIDs          []string `json:"parent_ids"`
	AuthoredDate       string   `json:"authored_date"`
	CommittedDate      string   `json:"committed_date"`
	CreatedAt          string   `json:"created_at"`
	AuthorName         string   `json:"author_name"`
	AuthorEmail        string   `json:"author_email"`
	AuthorAvatarURL    string   `json:"author_avatar_url"`
	CommitterName      string   `json:"committer_name"`
	CommitterEmail     string   `json:"committer_email"`
	CommitterAvatarURL string   `json:"committer_avatar_url"`
	Commit             struct {
		Author struct {
			Name  string `json:"name"`
			Date  string `json:"date"`
			Email string `json:"email"`
		} `json:"author"`
		Committer struct {
			Name  string `json:"name"`
			Date  string `json:"date"`
			Email string `json:"email"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
}

// BranchRequest represents the request body for creating a branch.
type BranchRequest struct {
	BranchName string `json:"branch_name"`
	Refs       string `json:"refs"`
}

// Label represents an issue/PR label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// Comment represents a comment on an issue or pull request
type Comment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	User      User   `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	HTMLURL   string `json:"html_url"`
	// ParentID is used for PR review comments to indicate the parent comment in a thread
	ParentID *string `json:"parent_id,omitempty"`

	// DiscussionID groups a review thread together.
	DiscussionID string `json:"discussion_id,omitempty"`
	// CommentType distinguishes review comments from plain comments:
	//   pr_comment   普通评论
	//   diff_comment 检视意见 (carries DiffPosition)
	CommentType string `json:"comment_type,omitempty"`
	// Resolved indicates whether a diff_comment thread has been resolved.
	Resolved bool `json:"resolved,omitempty"`
	// IsOutdated reports whether a diff_comment now points at outdated code.
	IsOutdated bool `json:"is_outdated,omitempty"`
	// DiffFile is the file path a diff_comment refers to.
	DiffFile string `json:"diff_file,omitempty"`
	// Path is the file a diff_comment refers to (alternate top-level field).
	Path string `json:"path,omitempty"`
	// DiffPosition locates a diff_comment within a file.
	DiffPosition *DiffPosition `json:"diff_position,omitempty"`
	// Reply holds nested replies to this comment (GitCode view=all shape).
	Reply []Comment `json:"reply,omitempty"`
}

// DiffPosition locates a review (diff) comment within a file's diff.
type DiffPosition struct {
	Path         string `json:"path,omitempty"`
	NewPath      string `json:"new_path,omitempty"`
	OldPath      string `json:"old_path,omitempty"`
	StartNewLine int    `json:"start_new_line,omitempty"`
	EndNewLine   int    `json:"end_new_line,omitempty"`
	StartOldLine int    `json:"start_old_line,omitempty"`
	EndOldLine   int    `json:"end_old_line,omitempty"`
	PositionType string `json:"position_type,omitempty"`
}

// CreateCommentResponse represents the response from creating a comment
type CreateCommentResponse struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	User      User   `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	HTMLURL   string `json:"html_url"`
}

// CommentRequest represents the request body for creating/updating a comment
type CommentRequest struct {
	Body string `json:"body"`
}

// ReplyResponse is the response from replying to a discussion thread.
// Here `id` is the discussion_id (the thread); the newly created reply's
// comment id is returned as `note_id`.
type ReplyResponse struct {
	DiscussionID string `json:"id"`
	Body         string `json:"body"`
	NoteID       int64  `json:"note_id"`
}

// Tag represents a git tag
type Tag struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Commit  struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	Tagger struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Date  string `json:"date"`
	} `json:"tagger"`
}

// TagRequest represents the request body for creating a tag
type TagRequest struct {
	TagName string `json:"tag_name"`
	Message string `json:"message"`
	Refs    string `json:"refs"`
}
