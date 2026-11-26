package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Use the documented default for pp. The unified endpoint has its own
// pagination parameters and does not specify the legacy endpoints' limits.
const unifiedSearchPageSize = 10

const maxUnifiedSearchPages = 100

type SearchOptions struct {
	Query string
	Limit int
	Sort  string
	Order string
}

type SearchRepositoryType string

const (
	SearchCodeRepository    SearchRepositoryType = "0"
	SearchModelRepository   SearchRepositoryType = "1"
	SearchDatasetRepository SearchRepositoryType = "2"
)

type SearchModelFilters struct {
	Task     string
	Library  string
	ParamMin string
	ParamMax string
}

func (c *Client) SearchRepositories(opts SearchOptions, repoType SearchRepositoryType, model SearchModelFilters) ([]SearchRepository, error) {
	switch repoType {
	case "", SearchCodeRepository, SearchModelRepository, SearchDatasetRepository:
	default:
		return nil, fmt.Errorf("invalid repo_type: %q", repoType)
	}
	params := url.Values{}
	if repoType != "" {
		params.Set("repo_type", string(repoType))
	}
	for name, value := range map[string]string{
		"task":      model.Task,
		"library":   model.Library,
		"param.min": model.ParamMin,
		"param.max": model.ParamMax,
	} {
		if value != "" {
			params.Set(name, value)
		}
	}
	return searchByType[SearchRepository](c, "repo", opts, params)
}

func (c *Client) SearchUsers(opts SearchOptions) ([]SearchUser, error) {
	return searchByType[SearchUser](c, "user", opts, url.Values{})
}

func (c *Client) SearchOrganizations(opts SearchOptions) ([]SearchOrganization, error) {
	return searchByType[SearchOrganization](c, "group", opts, url.Values{})
}

func (c *Client) SearchPullRequests(opts SearchOptions) ([]SearchPullRequest, error) {
	return searchByType[SearchPullRequest](c, "pullrequest", opts, url.Values{})
}

func (c *Client) SearchIssues(opts SearchOptions) ([]SearchIssue, error) {
	return searchByType[SearchIssue](c, "issue", opts, url.Values{})
}

func searchByType[T interface{ searchKey() searchResultKey }](client *Client, kind string, opts SearchOptions, params url.Values) ([]T, error) {
	if opts.Limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}

	perPage := min(opts.Limit, unifiedSearchPageSize)
	params.Set("type", kind)
	params.Set("q", opts.Query)
	params.Set("pp", strconv.Itoa(perPage))
	if opts.Sort != "" {
		params.Set("s", opts.Sort)
	}
	if opts.Order != "" {
		params.Set("o", opts.Order)
	}

	results := make([]T, 0, perPage)
	seen := make(map[searchResultKey]struct{}, perPage)
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}
	for page := 1; page <= maxUnifiedSearchPages; page++ {
		params.Set("p", strconv.Itoa(page))
		// Some successful responses omit total or return null. Use page_count
		// to determine the end, including when the server returns a short page.
		var response struct {
			PageCount *int            `json:"page_count"`
			Content   json.RawMessage `json:"content"`
		}
		path := "/search?" + params.Encode()
		if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json", policy, &response); err != nil {
			return nil, fmt.Errorf("search %s page %d: %w", kind, page, err)
		}
		if response.PageCount == nil || *response.PageCount < 0 || response.Content == nil {
			return nil, fmt.Errorf("search %s page %d: %w: expected a non-negative page_count and content", kind, page, ErrInvalidResponse)
		}
		// Accept explicit null content for empty results.
		var content []*T
		if err := json.Unmarshal(response.Content, &content); err != nil {
			return nil, fmt.Errorf("search %s page %d: %w: decode content: %w", kind, page, ErrInvalidResponse, err)
		}
		for _, item := range content {
			if item == nil {
				return nil, fmt.Errorf("search %s page %d: %w: null content item", kind, page, ErrInvalidResponse)
			}
			if len(results) == opts.Limit {
				continue
			}
			key := (*item).searchKey()
			if key != (searchResultKey{}) {
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
			}
			results = append(results, *item)
		}
		if len(results) == opts.Limit || page >= *response.PageCount {
			return results, nil
		}
	}
	return nil, fmt.Errorf("search %s: pagination reached the maximum of %d pages after collecting %d of %d requested results", kind, maxUnifiedSearchPages, len(results), opts.Limit)
}

type searchResultKey struct {
	id         string
	repository string
	iid        int
}

// The unified endpoint returns its own result shapes. Nullable fields keep
// missing values distinct from zero counts, false flags, and empty strings.
type SearchResult struct {
	ID        *string `json:"id"`
	URL       *string `json:"url"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

func (result SearchResult) searchKey() searchResultKey {
	if result.ID != nil {
		return searchResultKey{id: *result.ID}
	}
	return searchResultKey{}
}

type SearchRepository struct {
	SearchResult
	Name                    *string        `json:"name"`
	Description             *string        `json:"description"`
	ProjectType             *string        `json:"project_type"`
	NameWithNamespace       *string        `json:"name_with_namespace"`
	PathWithNamespace       *string        `json:"path_with_namespace"`
	Language                *string        `json:"language"`
	ShowRecommendIcon       *bool          `json:"show_recommend_icon"`
	ImportStarCount         *int           `json:"import_star_count"`
	Namespace               *string        `json:"namespace"`
	Path                    *string        `json:"path"`
	WebURL                  *string        `json:"web_url"`
	StarCount               *int           `json:"star_count"`
	ForksCount              *int           `json:"forks_count"`
	DefaultBranches         *string        `json:"default_branches"`
	LastActivityAt          *string        `json:"last_activity_at"`
	MainLanguage            *string        `json:"main_language"`
	License                 *SearchLicense `json:"license"`
	LastRepositoryUpdatedAt *string        `json:"last_repository_updated_at"`
	RepoType                *string        `json:"repo_type"`
	Task                    []SearchTag    `json:"task"`
	Library                 []SearchTag    `json:"library"`
	ModelParamSize          *string        `json:"model_param_size"`
	Archived                *bool          `json:"archived"`
	AvatarURL               *string        `json:"avatar_url"`
	Visibility              *string        `json:"visibility"`
	Creator                 *SearchAuthor  `json:"creator"`
}

type SearchLicense struct {
	Key       *string `json:"key"`
	Name      *string `json:"name"`
	Nickname  *string `json:"nickname"`
	HTMLURL   *string `json:"html_url"`
	SourceURL *string `json:"source_url"`
}

type SearchTag struct {
	Code *string `json:"code"`
	Name *string `json:"name"`
}

type SearchUser struct {
	SearchResult
	Username    *string `json:"username"`
	Nickname    *string `json:"nickname"`
	AvatarURL   *string `json:"avatar_url"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
}

type SearchOrganization struct {
	SearchResult
	Name            *string `json:"name"`
	FullName        *string `json:"full_name"`
	FullPath        *string `json:"full_path"`
	Description     *string `json:"description"`
	Avatar          *string `json:"avatar"`
	VisibilityLevel *string `json:"visibility_level"`
	FollowNum       *int    `json:"follow_num"`
	ProjectNum      *int    `json:"project_num"`
	WatchNum        *int    `json:"watch_num"`
	ForkNum         *int    `json:"fork_num"`
	StarNum         *int    `json:"star_num"`
}

type SearchIssueOrPullRequest struct {
	SearchResult
	IID               *int          `json:"iid"`
	Title             *string       `json:"title"`
	Description       *string       `json:"description"`
	State             *string       `json:"state"`
	Author            *SearchAuthor `json:"author"`
	Labels            []string      `json:"labels"`
	PathWithNamespace *string       `json:"path_with_namespace"`
	ProjectURL        *string       `json:"project_url"`
	CommentNum        *int          `json:"comment_num"`
}

func (item SearchIssueOrPullRequest) searchKey() searchResultKey {
	key := item.SearchResult.searchKey()
	if key.id == "" && item.PathWithNamespace != nil && *item.PathWithNamespace != "" && item.IID != nil && *item.IID > 0 {
		return searchResultKey{repository: *item.PathWithNamespace, iid: *item.IID}
	}
	return key
}

type SearchIssue struct {
	SearchIssueOrPullRequest
}

type SearchPullRequest struct {
	SearchIssueOrPullRequest
	TargetBranch *string `json:"target_branch"`
	SourceBranch *string `json:"source_branch"`
}

type SearchAuthor struct {
	ID        *int64  `json:"id"`
	Name      *string `json:"name"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatar_url"`
	WebURL    *string `json:"web_url"`
	NickName  *string `json:"nick_name"`
}
