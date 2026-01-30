package api

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
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	HTMLURL   string `json:"html_url"`
	User      User   `json:"user"`
	Head      Branch `json:"head"`
	Base      Branch `json:"base"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Merged    bool   `json:"merged"`
	Mergeable bool   `json:"mergeable"`
}

// Issue represents an AtomGit issue
type Issue struct {
	ID        int64   `json:"id"`
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	State     string  `json:"state"`
	HTMLURL   string  `json:"html_url"`
	User      User    `json:"user"`
	Labels    []Label `json:"labels"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// User represents an AtomGit user
type User struct {
	ID      int64  `json:"id"`
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
