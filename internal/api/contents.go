package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func buildContentsPath(owner, repo, path string, allowRoot bool) (string, error) {
	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	if path == "." {
		if !allowRoot {
			return "", fmt.Errorf("repository content path %q is only valid for directory listings", path)
		}
		return fmt.Sprintf("/repos/%s/%s/contents", escapedOwner, escapedRepo), nil
	}
	if path == "" {
		return "", fmt.Errorf("repository content path cannot be empty")
	}
	if strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return "", fmt.Errorf("invalid repository content path %q", path)
	}
	segments := strings.Split(path, "/")
	escapedSegments := make([]string, len(segments))
	for i, seg := range segments {
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("repository content path must not contain %q segments", seg)
		}
		escapedSegments[i] = url.PathEscape(seg)
	}
	return fmt.Sprintf("/repos/%s/%s/contents/%s", escapedOwner, escapedRepo, strings.Join(escapedSegments, "/")), nil
}

// GetRepositoryContent fetches a single file from a repository.
// path is a repository-relative content path.
// ref, when non-empty, selects a branch, tag, or commit.
func GetRepositoryContent(client *Client, owner, repo, path, ref string) (*RepositoryContent, error) {
	contentPath, err := buildContentsPath(owner, repo, path, false)
	if err != nil {
		return nil, err
	}
	if ref != "" {
		contentPath += "?ref=" + url.QueryEscape(ref)
	}

	var content RepositoryContent
	err = client.doJSONRequest(
		http.MethodGet, contentPath, nil,
		"application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&content,
	)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// ListRepositoryContent fetches directory contents from a repository.
// path is a repository-relative content path; use "." for root.
// ref, when non-empty, selects a branch, tag, or commit.
func ListRepositoryContent(client *Client, owner, repo, path, ref string) ([]RepositoryContent, error) {
	contentPath, err := buildContentsPath(owner, repo, path, true)
	if err != nil {
		return nil, err
	}
	if ref != "" {
		contentPath += "?ref=" + url.QueryEscape(ref)
	}

	var entries []RepositoryContent
	err = client.doJSONRequest(
		http.MethodGet, contentPath, nil,
		"application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&entries,
	)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []RepositoryContent{}
	}
	return entries, nil
}
