package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func CompareCommits(ctx context.Context, client *Client, owner, repo, base, head string) (*CommitComparison, error) {
	if ctx == nil {
		return nil, fmt.Errorf("compare context is nil")
	}
	if client == nil {
		return nil, fmt.Errorf("API client is nil")
	}
	path := fmt.Sprintf("/repos/%s/%s/compare/%s...%s",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(base), url.PathEscape(head))

	var comparison CommitComparison
	err := client.doJSONRequestContext(
		ctx,
		client.httpClient,
		http.MethodGet,
		path,
		nil,
		"",
		"application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&comparison,
	)
	if err != nil {
		return nil, fmt.Errorf("compare commits: %w", err)
	}
	if comparison.Commits == nil {
		comparison.Commits = []CompareCommit{}
	}
	if comparison.Files == nil {
		comparison.Files = []CompareFile{}
	}
	return &comparison, nil
}

func GetCommitText(ctx context.Context, client *Client, owner, repo, sha, format string) (io.ReadCloser, error) {
	if ctx == nil {
		return nil, fmt.Errorf("commit text context is nil")
	}
	if client == nil {
		return nil, fmt.Errorf("API client is nil")
	}
	if format != "diff" && format != "patch" {
		return nil, fmt.Errorf("unsupported commit text format %q", format)
	}

	path := fmt.Sprintf("/repos/%s/%s/commit/%s/%s",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha), format)
	resp, err := client.doRequestWithPolicyContext(ctx, streamingHTTPClient(client), http.MethodGet, path, nil, "", "text/plain, */*", true)
	if err != nil {
		return nil, fmt.Errorf("get commit %s: API request GET %s: %w", format, path, err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("get commit %s: %w", format, newAPIError(resp))
	}
	return resp.Body, nil
}
