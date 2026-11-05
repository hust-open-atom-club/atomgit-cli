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

// OrganizationCreationUncertainError means the request may have created the
// organization. Err retains the response or execution failure for inspection.
type OrganizationCreationUncertainError struct {
	Err error
}

func (e *OrganizationCreationUncertainError) Error() string {
	return fmt.Sprintf("create organization: %v; creation may have succeeded, check AtomGit before retrying", e.Err)
}

func (e *OrganizationCreationUncertainError) Unwrap() error { return e.Err }

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
		isHTTPError := errors.As(err, &httpErr)
		// Unexpected success, redirection, or server failure does not prove
		// that the organization was not created. Never replay the request.
		uncertainHTTPError := isHTTPError && ((httpErr.StatusCode >= 200 && httpErr.StatusCode < 400) ||
			(httpErr.StatusCode >= 500 && httpErr.StatusCode < 600))
		if !errors.Is(err, ErrRequestPreparation) && (!isHTTPError || uncertainHTTPError) {
			return CreatedOrganization{}, &OrganizationCreationUncertainError{Err: err}
		}
		return CreatedOrganization{}, fmt.Errorf("create organization: %w", err)
	}
	if result.ID <= 0 || result.Path != request.Path {
		return CreatedOrganization{}, &OrganizationCreationUncertainError{
			Err: fmt.Errorf("%w: missing or mismatched organization identity", ErrInvalidResponse),
		}
	}
	return result, nil
}
