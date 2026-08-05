package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CreateIssueWithAssignee creates an issue with an optional assignee.
// Uses exact HTTP 201 status and no retry.
func CreateIssueWithAssignee(client *Client, owner, repo, title, body, assignee string) (*Issue, error) {
	requestBody := map[string]string{
		"title": title,
		"body":  body,
	}
	if assignee != "" {
		requestBody["assignee"] = assignee
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal issue create body: %w", err)
	}

	path := fmt.Sprintf("/repos/%s/%s/issues", url.PathEscape(owner), url.PathEscape(repo))

	var issue Issue
	err = client.doJSONRequest(http.MethodPost, path, bytes.NewReader(jsonBody),
		"application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusCreated}, CanRetry: false},
		&issue)
	if err != nil {
		return nil, fmt.Errorf("create issue with assignee: %w", err)
	}
	return &issue, nil
}

// EditIssueAssignee edits an issue's assignee via multipart PATCH.
// When clearAssignee is true, assignee is set to empty string to clear the assignee.
// Uses the existing PatchForm which enforces exact HTTP 200.
func EditIssueAssignee(client *Client, owner, repo, number, title string, body *string, assignee string, clearAssignee bool) error {
	path := fmt.Sprintf("/repos/%s/issues/%s", url.PathEscape(owner), url.PathEscape(number))

	fields := map[string]string{
		"repo":  repo,
		"title": title,
	}
	if body != nil {
		fields["body"] = *body
	}
	if clearAssignee {
		fields["assignee"] = ""
	} else if assignee != "" {
		fields["assignee"] = assignee
	}

	if err := client.PatchForm(path, fields, nil); err != nil {
		return fmt.Errorf("edit issue assignee: %w", err)
	}
	return nil
}

// ListIssueLinkedPullRequests returns pull requests linked to an issue.
// Uses exact HTTP 200.
func ListIssueLinkedPullRequests(client *Client, owner, repo, number string) ([]IssueLinkedPullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%s/pull_requests",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(number))

	var prs []IssueLinkedPullRequest
	err := client.doJSONRequest(http.MethodGet, path, nil,
		"", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&prs)
	if err != nil {
		return nil, fmt.Errorf("list issue linked pull requests: %w", err)
	}
	if prs == nil {
		prs = []IssueLinkedPullRequest{}
	}
	return prs, nil
}

// ListIssueRelatedBranches returns the related branches for an issue as a string array.
// Uses exact HTTP 200.
func ListIssueRelatedBranches(client *Client, owner, repo, number string) ([]string, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%s/related_branches",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(number))

	var branches []string
	err := client.doJSONRequest(http.MethodGet, path, nil,
		"", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		&branches)
	if err != nil {
		return nil, fmt.Errorf("list issue related branches: %w", err)
	}
	if branches == nil {
		branches = []string{}
	}
	return branches, nil
}

// UpdateIssueRelatedBranches replaces the related branch set for an issue.
// Uses exact HTTP 200 and no retry (whole-list replacement).
func UpdateIssueRelatedBranches(client *Client, owner, repo, number string, branches []string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%s/related_branches",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(number))

	if branches == nil {
		branches = []string{}
	}
	reqBody := RelatedBranchesRequest{BranchNames: branches}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal related branches body: %w", err)
	}

	err = client.doJSONRequest(http.MethodPut, path, bytes.NewReader(jsonBody),
		"application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false},
		nil)
	if err != nil {
		return fmt.Errorf("update issue related branches: the remote state may have changed; re-list and retry: %w", err)
	}
	return nil
}
