package api

import "fmt"

// Repository represents an AtomGit repository
type Repository struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"html_url"`
	Private         bool   `json:"private"`
	DefaultBranch   string `json:"default_branch"`
	StarsCount      int    `json:"stars_count"`
	ForksCount      int    `json:"forks_count"`
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

// Branch represents a git branch
type Branch struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo Repository `json:"repo"`
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
	ParentID *int64 `json:"parent_id,omitempty"`
}

// CommentRequest represents the request body for creating/updating a comment
type CommentRequest struct {
	Body string `json:"body"`
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
