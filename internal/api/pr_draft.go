package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

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
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPatch, path,
		bytes.NewReader(body), "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil); err != nil {
		return fmt.Errorf("set PR draft status: %w", err)
	}
	return nil
}
