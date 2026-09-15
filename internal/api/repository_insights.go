package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// RepositoryContributorStatistics describes contribution totals and daily
// activity returned by the repository contributor statistics endpoint.
type RepositoryContributorStatistics struct {
	Name          string                         `json:"name"`
	Email         string                         `json:"email"`
	Contributions []RepositoryContributionDay    `json:"contributions"`
	Overview      RepositoryContributionOverview `json:"overview"`
}

type RepositoryContributionDay struct {
	Date         string `json:"date"`
	Additions    int64  `json:"additions"`
	Deletions    int64  `json:"deletions"`
	TotalChanges int64  `json:"total_changes"`
	CommitCount  int64  `json:"commit_count"`
}

type RepositoryContributionOverview struct {
	Additions    int64 `json:"additions"`
	Deletions    int64 `json:"deletions"`
	TotalChanges int64 `json:"total_changes"`
	CommitCount  int64 `json:"commit_count"`
}

type RepositoryEventAuthor struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	WebURL   string `json:"web_url"`
}

type RepositoryEvent struct {
	Action           int                   `json:"action"`
	ActionName       string                `json:"action_name"`
	Author           RepositoryEventAuthor `json:"author"`
	AuthorID         int64                 `json:"author_id"`
	AuthorUsername   string                `json:"author_username"`
	CreatedAt        string                `json:"created_at"`
	ProjectID        int64                 `json:"project_id"`
	Title            string                `json:"title"`
	FilterSensitive  FlexibleBool          `json:"filter_sensitive"`
	TargetID         int64                 `json:"target_id"`
	TargetIID        int64                 `json:"target_iid"`
	TargetTitle      string                `json:"target_title"`
	TargetType       string                `json:"target_type"`
	TargetTypeFormat string                `json:"target_type_format"`
}

type RepositoryEventsPage struct {
	Events      []RepositoryEvent `json:"events"`
	HasNextPage FlexibleBool      `json:"has_next_page"`
}

// RepositoryInsightUser is the endpoint-specific user shape returned by the
// repository watchers and stargazers endpoints.
type RepositoryInsightUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	ObjectID  string `json:"object_id"`
	WatchedAt string `json:"watch_at"`
	StarredAt string `json:"starred_at"`
}

type RepositoryDownloadDetail struct {
	Date           string `json:"pdate"`
	RepositoryID   string `json:"repo_id"`
	TodayDownloads int64  `json:"today_dl_cnt"`
	TotalDownloads int64  `json:"total_dl_cnt"`
}

type RepositoryDownloadStatistics struct {
	Details      []RepositoryDownloadDetail `json:"download_statistics_detail"`
	PeriodTotal  int64                      `json:"download_statistics_total"`
	HistoryTotal int64                      `json:"download_statistics_history_total"`
}

func repositoryInsightsPath(owner, repo, suffix string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + suffix
}

func GetRepositoryLanguages(client *Client, owner, repo string) (map[string]float64, error) {
	languages := make(map[string]float64)
	if err := client.Get(repositoryInsightsPath(owner, repo, "/languages"), &languages); err != nil {
		return nil, err
	}
	return languages, nil
}

func GetRepositoryContributorStatistics(client *Client, owner, repo string) ([]RepositoryContributorStatistics, error) {
	contributors := make([]RepositoryContributorStatistics, 0)
	if err := client.Get(repositoryInsightsPath(owner, repo, "/contributors/statistic"), &contributors); err != nil {
		return nil, err
	}
	for i := range contributors {
		if contributors[i].Contributions == nil {
			contributors[i].Contributions = make([]RepositoryContributionDay, 0)
		}
	}
	return contributors, nil
}

func GetRepositoryEventsPage(client *Client, owner, repo string, page, perPage int) (RepositoryEventsPage, error) {
	if page <= 0 || perPage <= 0 {
		return RepositoryEventsPage{}, fmt.Errorf("page and per-page values must be positive")
	}
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	var result RepositoryEventsPage
	err := client.Get(repositoryInsightsPath(owner, repo, "/events")+"?"+query.Encode(), &result)
	if result.Events == nil {
		result.Events = make([]RepositoryEvent, 0)
	}
	return result, err
}

func ListRepositoryWatchers(client *Client, owner, repo string, limit int) ([]RepositoryInsightUser, error) {
	return GetPaginated[RepositoryInsightUser](client, limit, func(page, perPage int) string {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(perPage))
		return repositoryInsightsPath(owner, repo, "/subscribers") + "?" + query.Encode()
	})
}

func ListRepositoryStargazers(client *Client, owner, repo string, limit int) ([]RepositoryInsightUser, error) {
	return GetPaginated[RepositoryInsightUser](client, limit, func(page, perPage int) string {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(perPage))
		return repositoryInsightsPath(owner, repo, "/stargazers") + "?" + query.Encode()
	})
}

func GetRepositoryDownloadStatistics(client *Client, owner, repo string) (RepositoryDownloadStatistics, error) {
	var statistics RepositoryDownloadStatistics
	err := client.Get(repositoryInsightsPath(owner, repo, "/download_statistics"), &statistics)
	if statistics.Details == nil {
		statistics.Details = make([]RepositoryDownloadDetail, 0)
	}
	return statistics, err
}
