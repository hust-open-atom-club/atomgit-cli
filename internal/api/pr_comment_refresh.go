package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// PullRequestCommentRefreshRequest requires an explicit scope. All is a local
// guard: only an explicit all request may serialize to the API's empty object.
type PullRequestCommentRefreshRequest struct {
	Paths  []string `json:"refresh_by_paths,omitempty"`
	NoteID string   `json:"refresh_by_note_id,omitempty"`
	All    bool     `json:"-"`
}

func (r PullRequestCommentRefreshRequest) Validate() error {
	scopes := 0
	if len(r.Paths) > 0 {
		scopes++
		for _, path := range r.Paths {
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("file paths must not be empty")
			}
		}
	}
	if r.NoteID != "" {
		scopes++
		if strings.TrimSpace(r.NoteID) == "" {
			return fmt.Errorf("comment ID must not be empty")
		}
	}
	if r.All {
		scopes++
	}
	if scopes != 1 {
		return fmt.Errorf("specify exactly one refresh scope: file paths, comment ID, or all")
	}
	return nil
}

// PullRequestCommentRefreshResult contains the IDs acknowledged by the API.
type PullRequestCommentRefreshResult struct {
	NoteIDs []int64 `json:"note_ids"`
}

// RefreshPullRequestCommentPositions never retries or follows redirects, even
// when the caller's HTTP client normally follows redirects.
func RefreshPullRequestCommentPositions(client *Client, owner, repo string, number int, request PullRequestCommentRefreshRequest) (PullRequestCommentRefreshResult, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || number <= 0 {
		return PullRequestCommentRefreshResult{}, fmt.Errorf("repository and positive PR number are required")
	}
	if err := request.Validate(); err != nil {
		return PullRequestCommentRefreshResult{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return PullRequestCommentRefreshResult{}, fmt.Errorf("encode comment refresh request: %w", err)
	}
	hc := *metadataHTTPClient(client.httpClient)
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var response struct {
		NoteIDs *[]*int64 `json:"note_ids"`
	}
	path := RepositoryPath(owner, repo, "pulls", strconv.Itoa(number), "refresh_position")
	err = client.doJSONRequestContext(client.requestContext(), &hc, http.MethodPost, path, bytes.NewReader(body), "application/json", "application/json", RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &response)
	if err != nil {
		return PullRequestCommentRefreshResult{}, fmt.Errorf("refresh PR comment positions: %w", err)
	}
	if response.NoteIDs == nil {
		return PullRequestCommentRefreshResult{}, fmt.Errorf("refresh PR comment positions: %w: missing note_ids array", ErrInvalidResponse)
	}
	ids := make([]int64, len(*response.NoteIDs))
	for i, id := range *response.NoteIDs {
		if id == nil {
			return PullRequestCommentRefreshResult{}, fmt.Errorf("refresh PR comment positions: %w: null note ID", ErrInvalidResponse)
		}
		ids[i] = *id
	}
	return PullRequestCommentRefreshResult{NoteIDs: ids}, nil
}
