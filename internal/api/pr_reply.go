package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// PullRequestReplyUncertainError marks failures after a reply may have been
// created. Preserve this operation boundary in user-facing diagnostics.
type PullRequestReplyUncertainError struct{ Err error }

func (e *PullRequestReplyUncertainError) Error() string {
	return fmt.Sprintf("PR reply could not be confirmed: %v; the reply may have been created, check AtomGit before retrying", e.Err)
}
func (e *PullRequestReplyUncertainError) Unwrap() error { return e.Err }

// CreatePullRequestReply sends one POST without retries or redirects. The
// discussion id identifies the thread, never the newly created comment.
func CreatePullRequestReply(client *Client, owner, repo, number, discussionID, body string) (ReplyResponse, error) {
	payload, err := json.Marshal(CommentRequest{Body: body})
	if err != nil {
		return ReplyResponse{}, fmt.Errorf("encode PR reply: %w: %w", ErrRequestPreparation, err)
	}
	hc := *metadataHTTPClient(client.httpClient)
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var response *ReplyResponse
	err = client.doJSONRequestContext(client.requestContext(), &hc, http.MethodPost,
		RepositoryPath(owner, repo, "pulls", number, "discussions", discussionID, "comments"),
		bytes.NewReader(payload), "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &response)
	if err != nil {
		status, isHTTP := errors.AsType[*HTTPError](err)
		if errors.Is(err, ErrRequestPreparation) || (isHTTP && status.StatusCode >= 400 && status.StatusCode < 500) {
			return ReplyResponse{}, fmt.Errorf("create PR reply: %w", err)
		}
		return ReplyResponse{}, &PullRequestReplyUncertainError{Err: err}
	}
	if response == nil {
		return ReplyResponse{}, &PullRequestReplyUncertainError{Err: fmt.Errorf("%w: missing PR reply object", ErrInvalidResponse)}
	}
	if _, valid := response.ReplyID(); !valid || response.DiscussionID != discussionID {
		return ReplyResponse{}, &PullRequestReplyUncertainError{Err: fmt.Errorf("%w: missing or invalid PR reply identity", ErrInvalidResponse)}
	}
	return *response, nil
}
