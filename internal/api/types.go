package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Repository represents an AtomGit repository
type Repository struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Path             string `json:"path"`
	FullName         string `json:"full_name"`
	Description      string `json:"description"`
	HTMLURL          string `json:"web_url"`
	AlternateHTMLURL string `json:"html_url"`
	Private          bool   `json:"private"`
	Internal         bool   `json:"internal"`
	DefaultBranch    string `json:"default_branch"`
	Language         string `json:"language"`
	License          string `json:"license"`
	Fork             bool   `json:"fork"`
	ParentFullName   string `json:"parentfull_name"`
	UpdatedAt        string `json:"updated_at"`
	StarsCount       int    `json:"stargazers_count"`
	ForksCount       int    `json:"forks_count"`
	WatchersCount    int    `json:"watchers_count"`
	OpenIssuesCount  int    `json:"open_issues_count"`
	Owner            struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
}

// RepositorySyncRequest selects the fork branch to synchronize. Force permits
// the server to overwrite commits that cannot be fast-forwarded.
type RepositorySyncRequest struct {
	Branch string `json:"branch"`
	Force  bool   `json:"force,omitempty"`
}

// RepositorySyncResponse is returned by the fork synchronization endpoint.
type RepositorySyncResponse struct {
	Result bool `json:"repo_sync_result"`
}

// PullRequest represents an AtomGit pull request
type PullRequest struct {
	ID                int64       `json:"id"`
	Number            interface{} `json:"number"`
	Title             string      `json:"title"`
	Body              string      `json:"body"`
	State             string      `json:"state"`
	HTMLURL           string      `json:"html_url"`
	User              User        `json:"user"`
	Head              Branch      `json:"head"`
	Base              Branch      `json:"base"`
	Assignees         []User      `json:"assignees"`
	ApprovalReviewers []User      `json:"approval_reviewers"`
	Testers           []User      `json:"testers"`
	Labels            []Label     `json:"labels"`
	Milestone         *Milestone  `json:"milestone"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
	Merged            bool        `json:"merged"`
	Mergeable         bool        `json:"mergeable"`
}

// PullRequestWriteResponse represents the compact response returned by pull
// request write endpoints. AtomGit uses web_url in create responses while
// other endpoints may use html_url or return an empty body.
type PullRequestWriteResponse struct {
	ID      interface{} `json:"id"`
	Number  interface{} `json:"number"`
	IID     interface{} `json:"iid"`
	HTMLURL string      `json:"html_url"`
	WebURL  string      `json:"web_url"`
}

// GetNumber returns the PR number from either supported response field.
func (pr *PullRequestWriteResponse) GetNumber() string {
	if number := formatIdentifier(pr.Number); number != "" {
		return number
	}
	return formatIdentifier(pr.IID)
}

// GetURL returns the browser URL from either supported response field.
func (pr *PullRequestWriteResponse) GetURL() string {
	if url := strings.TrimSpace(pr.WebURL); url != "" {
		return url
	}
	return strings.TrimSpace(pr.HTMLURL)
}

// PullRequestReviewRequest represents AtomGit's formal review request.
// Force only takes effect for repository administrators.
type PullRequestReviewRequest struct {
	Force bool `json:"force"`
}

// GetNumber returns the PR number as a string
func (pr *PullRequest) GetNumber() string {
	return formatIdentifier(pr.Number)
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
	return formatIdentifier(i.Number)
}

func formatIdentifier(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
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

// Collaborator represents a repository member and the provenance of their
// effective permission.
type Collaborator struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Username    string                  `json:"username"`
	Login       string                  `json:"login"`
	WebURL      string                  `json:"web_url"`
	AccessLevel int                     `json:"access_level"`
	Type        string                  `json:"type"`
	JoinWay     string                  `json:"join_way"`
	SourceName  string                  `json:"source_name"`
	RoleName    string                  `json:"role_name"`
	RoleNameCN  string                  `json:"role_name_cn"`
	Permission  string                  `json:"permission"`
	Permissions CollaboratorPermissions `json:"permissions"`
}

// CollaboratorPermissions contains AtomGit's built-in repository permissions.
type CollaboratorPermissions struct {
	Pull  FlexibleBool `json:"pull"`
	Push  FlexibleBool `json:"push"`
	Admin FlexibleBool `json:"admin"`
}

// Webhook represents a repository webhook. The API's password field is
// intentionally omitted so commands cannot accidentally render stored secrets.
type Webhook struct {
	ID                  int64        `json:"id"`
	URL                 string       `json:"url"`
	Result              string       `json:"result"`
	ProjectID           int64        `json:"project_id"`
	ResultCode          int          `json:"result_code"`
	PushEvents          FlexibleBool `json:"push_events"`
	TagPushEvents       FlexibleBool `json:"tag_push_events"`
	IssuesEvents        FlexibleBool `json:"issues_events"`
	NoteEvents          FlexibleBool `json:"note_events"`
	MergeRequestsEvents FlexibleBool `json:"merge_requests_events"`
	CreatedAt           string       `json:"created_at"`
	Active              FlexibleBool `json:"active"`
}

// Organization represents an AtomGit organization visible to the authenticated user.
type Organization struct {
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
}

// SSHKey represents a public SSH key registered with an AtomGit account.
type SSHKey struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	URL         string `json:"url"`
	CreatedAt   string `json:"created_at"`
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
	User               User         `json:"user"`
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

// ProtectedBranchUser identifies a user explicitly allowed by a protected
// branch rule. AtomGit responses use either username or login.
type ProtectedBranchUser struct {
	Username string `json:"username"`
	Login    string `json:"login"`
	Name     string `json:"name"`
}

// ProtectedBranchRule is the rule returned by the protect_branches endpoint.
type ProtectedBranchRule struct {
	Name               string                `json:"name"`
	UpdatedAt          string                `json:"updated_at"`
	PushUsers          []ProtectedBranchUser `json:"push_users"`
	MergeUsers         []ProtectedBranchUser `json:"merge_users"`
	Merged             FlexibleBool          `json:"merged"`
	DevelopersCanPush  FlexibleBool          `json:"developers_can_push"`
	DevelopersCanMerge FlexibleBool          `json:"developers_can_merge"`
	CommitterCanPush   FlexibleBool          `json:"committer_can_push"`
	CommitterCanMerge  FlexibleBool          `json:"committer_can_merge"`
	MasterCanPush      FlexibleBool          `json:"master_can_push"`
	MasterCanMerge     FlexibleBool          `json:"master_can_merge"`
	MaintainerCanPush  FlexibleBool          `json:"maintainer_can_push"`
	MaintainerCanMerge FlexibleBool          `json:"maintainer_can_merge"`
	OwnerCanPush       FlexibleBool          `json:"owner_can_push"`
	OwnerCanMerge      FlexibleBool          `json:"owner_can_merge"`
	NoOneCanPush       FlexibleBool          `json:"no_one_can_push"`
	NoOneCanMerge      FlexibleBool          `json:"no_one_can_merge"`
}

// ProtectedBranchRequest is accepted by both protected branch create and
// update endpoints. The API requires both permission strings on every write.
type ProtectedBranchRequest struct {
	Wildcard string `json:"wildcard,omitempty"`
	Pusher   string `json:"pusher"`
	Merger   string `json:"merger"`
}

// Label represents an issue/PR label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// Milestone represents an AtomGit repository milestone.
type Milestone struct {
	Number       interface{} `json:"number"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	State        string      `json:"state"`
	DueOn        string      `json:"due_on"`
	OpenIssues   int         `json:"open_issues"`
	ClosedIssues int         `json:"closed_issues"`
	RepositoryID int64       `json:"repository_id"`
	URL          string      `json:"url"`
	HTMLURL      string      `json:"html_url"`
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
}

// GetNumber returns the milestone number as a string.
func (m *Milestone) GetNumber() string {
	return formatIdentifier(m.Number)
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
	ID        interface{} `json:"id"`
	NoteID    interface{} `json:"note_id"`
	Body      string      `json:"body"`
	User      User        `json:"user"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	HTMLURL   string      `json:"html_url"`
	WebURL    string      `json:"web_url"`
}

// GetID returns the created comment identifier from either response shape.
func (c *CreateCommentResponse) GetID() string {
	if id := formatIdentifier(c.ID); id != "" {
		return id
	}
	return formatIdentifier(c.NoteID)
}

// GetURL returns the comment browser URL when AtomGit supplies one.
func (c *CreateCommentResponse) GetURL() string {
	if url := strings.TrimSpace(c.WebURL); url != "" {
		return url
	}
	return strings.TrimSpace(c.HTMLURL)
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

// MergePRRequest represents the request body for merging a pull request
type MergePRRequest struct {
	MergeMethod         string `json:"merge_method"`
	Title               string `json:"title,omitempty"`
	Description         string `json:"description,omitempty"`
	ForceMerge          bool   `json:"force_merge,omitempty"`
	Squash              bool   `json:"squash,omitempty"`
	SquashCommitMessage string `json:"squash_commit_message,omitempty"`
}

// MergePRResponse represents the response from merging a pull request
type MergePRResponse struct {
	SHA     string `json:"sha"`
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

// ReleaseStatusPre and ReleaseStatusLatest are the two supported values of
// the release_status field in the create/update release request body.
const (
	ReleaseStatusPre    = "pre"
	ReleaseStatusLatest = "latest"
)

// ReleaseAuthor is the author embedded in a Release response.
type ReleaseAuthor struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	URL       string `json:"url"`
}

// ReleaseAsset is one entry of the assets array on a Release. The id and
// type fields distinguish deletable uploaded attachments (type="attach",
// id>0) from auto-generated source archives that cannot be removed.
type ReleaseAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Release is the response shape returned by the AtomGit release endpoints.
// release_status is the server-side status field; it is reported on every
// release response (e.g. "latest" or "pre").
type Release struct {
	TagName         string         `json:"tag_name"`
	TargetCommitish string         `json:"target_commitish"`
	Draft           bool           `json:"draft"`
	Prerelease      bool           `json:"prerelease"`
	Name            string         `json:"name"`
	Body            string         `json:"body"`
	ReleaseStatus   string         `json:"release_status"`
	CreatedAt       string         `json:"created_at"`
	Author          ReleaseAuthor  `json:"author"`
	Assets          []ReleaseAsset `json:"assets"`
}

// CreateReleaseRequest is the body for POST /repos/{owner}/{repo}/releases.
// tag_name, name and body are required; target_commitish and release_status
// are optional. release_status, when sent, must be ReleaseStatusPre or
// ReleaseStatusLatest; it is omitted from the wire format when empty so the
// server keeps its default.
type CreateReleaseRequest struct {
	TagName         string `json:"tag_name"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	ReleaseStatus   string `json:"release_status,omitempty"`
}

// UpdateReleaseRequest is the body for PATCH /repos/{owner}/{repo}/releases/{tag}.
// name and body are required; release_status is optional and follows the same
// rules as CreateReleaseRequest. The AtomGit release API does not support
// changing target_commitish after creation.
type UpdateReleaseRequest struct {
	Name          string `json:"name"`
	Body          string `json:"body"`
	ReleaseStatus string `json:"release_status,omitempty"`
}

// ReleaseUploadURL is the exact response of
// GET /repos/{owner}/{repo}/releases/{tag}/upload_url?file_name=...
// The URL points at an external object-store; only the headers returned here
// may be sent on the subsequent PUT. The AtomGit Bearer token must not be
// forwarded to the external host.
type ReleaseUploadURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Discussion represents a repository discussion returned by AtomGit API v5.
type Discussion struct {
	ID           string             `json:"id"`
	Number       int                `json:"number"`
	Title        string             `json:"title"`
	Author       DiscussionAuthor   `json:"author"`
	Category     DiscussionCategory `json:"category"`
	IsClosed     FlexibleBool       `json:"is_closed"`
	IsAnswered   FlexibleBool       `json:"is_answered"`
	IsLocked     FlexibleBool       `json:"is_lock"`
	IsPinned     FlexibleBool       `json:"is_pin"`
	CommentTotal int                `json:"comment_total"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
}

// DiscussionAuthor is author of discussion
type DiscussionAuthor struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type DiscussionCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Type        int    `json:"type"`
}
