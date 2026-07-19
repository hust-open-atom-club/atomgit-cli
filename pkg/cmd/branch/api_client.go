package branch

import (
	"fmt"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func newAPIClient(f *cmdutil.Factory, token string) (*api.Client, error) {
	if f.HttpClient == nil {
		return api.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), nil
}

func parseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repository)
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repository)
	}
	return owner, repo, nil
}

func escapePathSegment(value string) string {
	return url.PathEscape(value)
}
