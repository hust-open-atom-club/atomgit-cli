package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// PullRequestDiscussionState identifies a top-level review discussion. A
// pointer distinguishes an explicit unresolved state from missing metadata.
type PullRequestDiscussionState struct {
	DiscussionID string `json:"discussion_id"`
	CommentType  string `json:"comment_type"`
	Resolved     *bool  `json:"resolved"`
}

// ValidatePullRequestDiscussionID accepts opaque, safe path identifiers, not
// comment/note numbers inferred from another field. Membership is checked by
// GetPullRequestDiscussionState before a command writes.
func ValidatePullRequestDiscussionID(id string) error {
	if id == "" || id == "." || id == ".." {
		return fmt.Errorf("invalid discussion ID: use the discussion_id shown by pr comment view")
	}
	for _, c := range id {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~' {
			continue
		}
		return fmt.Errorf("invalid discussion ID: expected letters, digits, '-', '_', '.', or '~'")
	}
	return nil
}

func validatePRDiscussion(number int, id string) error {
	if number <= 0 {
		return fmt.Errorf("invalid PR number: must be positive")
	}
	return ValidatePullRequestDiscussionID(id)
}

// GetPullRequestDiscussionState verifies PR identity and searches its review
// discussions, continuing through short pages until an empty page. Numeric
// note IDs and nested replies are never used as discussion identifiers.
func GetPullRequestDiscussionState(client *Client, owner, repo string, number int, id string) (PullRequestDiscussionState, error) {
	if err := validatePRDiscussion(number, id); err != nil {
		return PullRequestDiscussionState{}, err
	}
	path := RepositoryPath(owner, repo, "pulls", strconv.Itoa(number))
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}
	var pr PullRequest
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json", policy, &pr); err != nil {
		return PullRequestDiscussionState{}, fmt.Errorf("verify pull request #%d: %w", number, err)
	}
	if pr.GetNumber() != strconv.Itoa(number) {
		return PullRequestDiscussionState{}, fmt.Errorf("%w: missing or mismatched pull request identity", ErrInvalidResponse)
	}
	for page := 1; page <= maxUntilEmptyRequestPages; page++ {
		var discussions []PullRequestDiscussionState
		pagePath := commentPagePath(path+"/comments?view=all", page)
		if err := client.doJSONRequest(http.MethodGet, pagePath, nil, "", "application/json", policy, &discussions); err != nil {
			return PullRequestDiscussionState{}, fmt.Errorf("read discussions on PR #%d: %w", number, err)
		}
		if discussions == nil {
			return PullRequestDiscussionState{}, fmt.Errorf("%w: expected a PR discussion array", ErrInvalidResponse)
		}
		for _, discussion := range discussions {
			if discussion.DiscussionID == id {
				if discussion.CommentType != "diff_comment" || discussion.Resolved == nil {
					return PullRequestDiscussionState{}, fmt.Errorf("%w: discussion is not a review discussion with an explicit resolved state", ErrInvalidResponse)
				}
				return discussion, nil
			}
		}
		if len(discussions) == 0 {
			return PullRequestDiscussionState{}, fmt.Errorf("discussion %q does not belong to PR #%d", id, number)
		}
	}
	return PullRequestDiscussionState{}, fmt.Errorf("read PR discussions: pagination reached the maximum of %d pages", maxUntilEmptyRequestPages)
}

// SetPullRequestDiscussionResolved sends one PUT with an explicit boolean.
// The documented HTTP 200 response has no body. Writes are not retried or
// redirected; callers must read the discussion back to confirm its state.
func SetPullRequestDiscussionResolved(client *Client, owner, repo string, number int, id string, resolved bool) error {
	if err := validatePRDiscussion(number, id); err != nil {
		return err
	}
	request := struct {
		Resolved bool `json:"resolved"`
	}{Resolved: resolved}
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode discussion state: %w: %w", ErrRequestPreparation, err)
	}
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	path := RepositoryPath(owner, repo, "pulls", strconv.Itoa(number), "comments", id)
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPut, path,
		bytes.NewReader(body), "application/json", "application/json", policy, nil); err != nil {
		return fmt.Errorf("set resolved=%t for discussion %q on PR #%d: %w", resolved, id, number, err)
	}
	return nil
}
