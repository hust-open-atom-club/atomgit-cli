package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RepositoryContentsResponse preserves the complete JSON returned by the
// repository contents endpoint while also exposing its file or directory
// representation for command-specific rendering.
type RepositoryContentsResponse struct {
	Raw     json.RawMessage
	File    *RepositoryContent
	Entries []RepositoryContent
}

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

func repositoryContentsContext(owner, repo, path, ref string) string {
	refDescription := "default branch"
	if ref != "" {
		refDescription = fmt.Sprintf("ref %q", ref)
	}
	return fmt.Sprintf("repository %s/%s path %q at %s", owner, repo, path, refDescription)
}

func getRepositoryContents(client *Client, owner, repo, path, ref string, allowRoot bool) (*RepositoryContentsResponse, error) {
	context := repositoryContentsContext(owner, repo, path, ref)
	contentPath, err := buildContentsPath(owner, repo, path, allowRoot)
	if err != nil {
		return nil, fmt.Errorf("get contents for %s: %w", context, err)
	}
	if ref != "" {
		contentPath += "?ref=" + url.QueryEscape(ref)
	}

	var raw json.RawMessage
	err = client.doJSONRequest(
		http.MethodGet, contentPath, nil,
		"application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&raw,
	)
	if err != nil {
		return nil, fmt.Errorf("get contents for %s: %w", context, err)
	}

	trimmed := bytes.TrimSpace(raw)
	response := &RepositoryContentsResponse{Raw: raw}
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("get contents for %s: API returned an empty JSON response", context)
	}

	switch trimmed[0] {
	case '{':
		var file RepositoryContent
		if err := json.Unmarshal(trimmed, &file); err != nil {
			return nil, fmt.Errorf("decode file contents for %s: %w", context, err)
		}
		response.File = &file
	case '[':
		if err := json.Unmarshal(trimmed, &response.Entries); err != nil {
			return nil, fmt.Errorf("decode directory contents for %s: %w", context, err)
		}
		if response.Entries == nil {
			response.Entries = []RepositoryContent{}
		}
	default:
		return nil, fmt.Errorf("get contents for %s: expected a file object or directory array", context)
	}

	return response, nil
}

// GetRepositoryContents fetches a file or directory response from the
// repository contents endpoint. Use "." as path for the repository root.
func GetRepositoryContents(client *Client, owner, repo, path, ref string) (*RepositoryContentsResponse, error) {
	return getRepositoryContents(client, owner, repo, path, ref, true)
}

// GetRepositoryContent fetches a single file from a repository.
// path is a repository-relative content path.
// ref, when non-empty, selects a branch, tag, or commit.
func GetRepositoryContent(client *Client, owner, repo, path, ref string) (*RepositoryContent, error) {
	response, err := getRepositoryContents(client, owner, repo, path, ref, false)
	if err != nil {
		return nil, err
	}
	if response.File == nil {
		return nil, fmt.Errorf("repository content path %q is a directory, not a file", path)
	}
	return response.File, nil
}

// ListRepositoryContent fetches directory contents from a repository.
// path is a repository-relative content path; use "." for root.
// ref, when non-empty, selects a branch, tag, or commit.
func ListRepositoryContent(client *Client, owner, repo, path, ref string) ([]RepositoryContent, error) {
	response, err := getRepositoryContents(client, owner, repo, path, ref, true)
	if err != nil {
		return nil, err
	}
	if response.File != nil {
		return nil, fmt.Errorf("repository content path %q is a file, not a directory", path)
	}
	return response.Entries, nil
}
