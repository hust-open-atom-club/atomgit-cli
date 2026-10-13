package api

import (
	"bytes"
	"encoding/base64"
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

const contentWriteMessage = "repository content writes are not retried; inspect the branch before retrying"

type contentWriteRequest struct {
	Content *string `json:"content,omitempty"`
	Message string  `json:"message"`
	SHA     string  `json:"sha,omitempty"`
	Branch  string  `json:"branch"`
}

// RepositoryContentChange is the documented result of creating, updating, or
// deleting one file. Create and delete primarily return commit; update may
// also return content. Either object can be absent.
type RepositoryContentChange struct {
	Commit  *RepositoryContentCommit `json:"commit"`
	Content *RepositoryContent       `json:"content"`
}

// RepositoryContentCommit is the commit object returned by a content write.
type RepositoryContentCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

// CreateRepositoryContent creates path on branch. It does not read the path
// first and never turns a conflict into an update.
func CreateRepositoryContent(client *Client, owner, repo, path, branch, message string, content []byte) (RepositoryContentChange, error) {
	encoded := base64.StdEncoding.EncodeToString(content)
	return writeRepositoryContent(client, http.MethodPost, owner, repo, path, contentWriteRequest{
		Content: &encoded,
		Message: message,
		Branch:  branch,
	}, "create")
}

// UpdateRepositoryContent replaces path using its current file Blob SHA.
// A conflicting SHA is returned to the caller and is not refreshed or retried.
func UpdateRepositoryContent(client *Client, owner, repo, path, branch, message, sha string, content []byte) (RepositoryContentChange, error) {
	encoded := base64.StdEncoding.EncodeToString(content)
	return writeRepositoryContent(client, http.MethodPut, owner, repo, path, contentWriteRequest{
		Content: &encoded,
		Message: message,
		SHA:     sha,
		Branch:  branch,
	}, "update")
}

// DeleteRepositoryContent deletes path using its current file Blob SHA.
func DeleteRepositoryContent(client *Client, owner, repo, path, branch, message, sha string) (RepositoryContentChange, error) {
	return writeRepositoryContent(client, http.MethodDelete, owner, repo, path, contentWriteRequest{
		Message: message,
		SHA:     sha,
		Branch:  branch,
	}, "delete")
}

func writeRepositoryContent(client *Client, method, owner, repo, path string, request contentWriteRequest, operation string) (RepositoryContentChange, error) {
	contentPath, err := buildContentsPath(owner, repo, path, false)
	if err != nil {
		return RepositoryContentChange{}, fmt.Errorf("%s contents for repository %s/%s path %q: %w", operation, owner, repo, path, err)
	}
	body, err := json.Marshal(request)
	if err != nil {
		return RepositoryContentChange{}, fmt.Errorf("%s contents for repository %s/%s path %q: %w", operation, owner, repo, path, err)
	}
	// Redirects can replay PUT and DELETE independently of the retry policy.
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	var change RepositoryContentChange
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, method, contentPath, bytes.NewReader(body), "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &change); err != nil {
		return RepositoryContentChange{}, fmt.Errorf("%s contents for repository %s/%s path %q on branch %q: %w; %s", operation, owner, repo, path, request.Branch, err, contentWriteMessage)
	}
	return change, nil
}
