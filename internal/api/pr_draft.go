package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// PullRequestDraftUpdateUncertainError retains a failure after a draft write
// which may have taken effect. The caller must check AtomGit before retrying.
type PullRequestDraftUpdateUncertainError struct{ Err error }

func (e *PullRequestDraftUpdateUncertainError) Error() string {
	return fmt.Sprintf("PR draft status update could not be confirmed: %v; check AtomGit before retrying", e.Err)
}
func (e *PullRequestDraftUpdateUncertainError) Unwrap() error { return e.Err }

// SetPullRequestDraft sends one PATCH. The caller reads the PR to confirm the new state.
func SetPullRequestDraft(client *Client, owner, repo, number string, draft bool) error {
	body, err := json.Marshal(struct {
		Draft bool `json:"draft"`
	}{Draft: draft})
	if err != nil {
		return fmt.Errorf("encode PR draft status: %w", err)
	}
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/pulls/" + url.PathEscape(number)
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	err = client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPatch, path,
		bytes.NewReader(body), "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil)
	if err == nil {
		return nil
	}
	var httpErr *HTTPError
	isHTTP := errors.As(err, &httpErr)
	// A 4xx response rejects the write. Every other failure leaves the draft
	// state unknown: the request may have reached the server.
	definiteFailure := isHTTP && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500
	if errors.Is(err, ErrRequestPreparation) || definiteFailure {
		return fmt.Errorf("set PR draft status: %w", err)
	}
	return &PullRequestDraftUpdateUncertainError{Err: err}
}
