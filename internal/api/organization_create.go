package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// OrganizationCreateRequest preserves omitted options so the API can apply its
// documented name=path and visibility=public defaults.
type OrganizationCreateRequest struct {
	Path        string  `json:"path"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
}

// CreatedOrganization is the POST /orgs response, whose URL and visibility
// fields differ from the organization discovery endpoints.
type CreatedOrganization struct {
	ID          int64  `json:"id"`
	WebURL      string `json:"web_url"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	FullName    string `json:"full_name"`
	FullPath    string `json:"full_path"`
}

// CreateOrganization sends a single write without transport retries or redirect
// replay. A failed response read does not establish whether creation occurred.
func CreateOrganization(client *Client, request OrganizationCreateRequest) (CreatedOrganization, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return CreatedOrganization{}, fmt.Errorf("encode organization creation: %w: %w", ErrRequestPreparation, err)
	}
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var result CreatedOrganization
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPost,
		"/orgs", bytes.NewReader(body), "application/json", "application/json", policy, &result); err != nil {
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) && !errors.Is(err, ErrRequestPreparation) {
			return CreatedOrganization{}, fmt.Errorf("create organization: %w; creation may have succeeded, check AtomGit before retrying", err)
		}
		return CreatedOrganization{}, fmt.Errorf("create organization: %w", err)
	}
	if result.ID <= 0 || result.Path != request.Path {
		return CreatedOrganization{}, fmt.Errorf("create organization: %w: missing or mismatched organization identity; creation may have succeeded, check AtomGit before retrying", ErrInvalidResponse)
	}
	return result, nil
}
