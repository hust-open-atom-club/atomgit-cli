package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func ListPullRequestCommits(client *Client, owner, repo, number string, limit int) ([]PullRequestCommit, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	commits := make([]PullRequestCommit, 0, min(limit, defaultMaxPerPage))
	for page := 1; len(commits) < limit; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(defaultMaxPerPage))
		path := fmt.Sprintf("/repos/%s/%s/pulls/%s/commits?%s",
			escapedOwner, escapedRepo, escapedNumber, query.Encode())

		var pageCommits []PullRequestCommit
		err := client.doJSONRequest(
			http.MethodGet,
			path,
			nil,
			"application/json",
			"application/json",
			RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
			&pageCommits,
		)
		if err != nil {
			return nil, fmt.Errorf("list pull request commits: %w", err)
		}

		commits = append(commits, pageCommits...)
		if len(pageCommits) < defaultMaxPerPage {
			break
		}
	}

	if len(commits) > limit {
		commits = commits[:limit]
	}
	return commits, nil
}

func ListPullRequestFiles(client *Client, owner, repo, number string) ([]PullRequestFile, error) {
	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/files", escapedOwner, escapedRepo, escapedNumber)

	var files []PullRequestFile
	err := client.doJSONRequest(http.MethodGet, path, nil, "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &files)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = make([]PullRequestFile, 0)
	}
	return files, nil
}

// ListPullRequestReactions lists up to limit reactions on a pull request. The
// user_reactions endpoint supports page/per_page, so results are paginated.
func ListPullRequestReactions(client *Client, owner, repo, number string, limit int) ([]PullRequestReaction, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	return GetPaginated[PullRequestReaction](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/pulls/%s/user_reactions?page=%d&per_page=%d",
			escapedOwner, escapedRepo, escapedNumber, page, perPage)
	})
}

// ListPullRequestOperateLogs lists up to limit operation-log entries for a pull
// request. The operate_logs endpoint supports page/per_page, so results are
// paginated.
func ListPullRequestOperateLogs(client *Client, owner, repo, number string, limit int) ([]PullRequestOperateLog, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	return GetPaginated[PullRequestOperateLog](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/pulls/%s/operate_logs?page=%d&per_page=%d",
			escapedOwner, escapedRepo, escapedNumber, page, perPage)
	})
}

// ListPullRequestModifyHistory lists modification-history entries for a pull
// request. The modify_history endpoint does not support pagination, so the full
// response is fetched and then truncated to limit locally.
func ListPullRequestModifyHistory(client *Client, owner, repo, number string, limit int) ([]PullRequestModifyHistory, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/modify_history", escapedOwner, escapedRepo, escapedNumber)

	var history []PullRequestModifyHistory
	err := client.doJSONRequest(http.MethodGet, path, nil, "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &history)
	if err != nil {
		return nil, fmt.Errorf("list pull request modify history: %w", err)
	}
	if history == nil {
		history = make([]PullRequestModifyHistory, 0)
	}
	if len(history) > limit {
		history = history[:limit]
	}
	return history, nil
}
