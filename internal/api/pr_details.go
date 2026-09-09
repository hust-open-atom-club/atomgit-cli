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

func ListPullRequestReactions(client *Client, owner, repo, number string) ([]PullRequestReaction, error) {
	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	escapedNumber := url.PathEscape(number)

	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/user_reactions", escapedOwner, escapedRepo, escapedNumber)

	var reactions []PullRequestReaction
	err := client.doJSONRequest(http.MethodGet, path, nil, "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &reactions)
	if err != nil {
		return nil, err
	}
	if reactions == nil {
		reactions = make([]PullRequestReaction, 0)
	}
	return reactions, nil
}
