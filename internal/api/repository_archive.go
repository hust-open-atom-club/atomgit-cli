package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	OrganizationRepositoryActive   = 0
	OrganizationRepositoryArchived = 2
)

// repositoryArchiveRequestError keeps potentially credential-bearing response
// text and transport messages out of Error(), preserving causes for errors.As.
type repositoryArchiveRequestError struct{ cause error }

func (e *repositoryArchiveRequestError) Error() string {
	return "organization repository archive request failed; response details omitted, check AtomGit before retrying"
}
func (e *repositoryArchiveRequestError) Unwrap() error { return e.cause }

// SetOrganizationRepositoryArchiveStatus performs one non-replayed write to
// the organization-only archive endpoint. Success requires HTTP 200 and an
// explicitly present integer business code of zero; response messages are not
// exposed because they may echo passwords or authentication data.
func SetOrganizationRepositoryArchiveStatus(client *Client, organization, repository string, status int, password *string) error {
	if status != OrganizationRepositoryActive && status != OrganizationRepositoryArchived {
		return fmt.Errorf("invalid organization repository status: expected 0 or 2")
	}
	request := struct {
		Status   int     `json:"status"`
		Password *string `json:"password,omitempty"`
	}{Status: status, Password: password}
	//nolint:gosec // G117: The documented archive API accepts this optional password in its JSON body; it is never logged or persisted.
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode repository archive request: %w: %w", ErrRequestPreparation, err)
	}
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	path := fmt.Sprintf("/org/%s/repo/%s/status", url.PathEscape(organization), url.PathEscape(repository))
	var response struct {
		Code    *int    `json:"code"`
		Message *string `json:"msg"`
	}
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPut, path, bytes.NewReader(body), "application/json", "application/json", policy, &response); err != nil {
		return &repositoryArchiveRequestError{cause: err}
	}
	if response.Code == nil {
		return fmt.Errorf("organization repository archive response: %w: missing business code", ErrInvalidResponse)
	}
	if response.Message == nil {
		return fmt.Errorf("organization repository archive response: %w: missing response message", ErrInvalidResponse)
	}
	if *response.Code != 0 {
		return fmt.Errorf("organization repository archive request rejected with business code %d", *response.Code)
	}
	return nil
}
