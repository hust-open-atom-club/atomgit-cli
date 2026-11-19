package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// OrganizationEditRequest preserves the difference between omitted fields and
// an explicit empty string. Name is a display name, never the organization path.
type OrganizationEditRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Email       *string `json:"email,omitempty"`
	Location    *string `json:"location,omitempty"`
	Website     *string `json:"html_url,omitempty"`
}

func (r OrganizationEditRequest) Validate() error {
	if r.Name == nil && r.Description == nil && r.Email == nil && r.Location == nil && r.Website == nil {
		return fmt.Errorf("at least one organization profile field must be provided")
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return fmt.Errorf("organization name must not be empty")
	}
	return nil
}

// EditedOrganization is the identity acknowledged by PATCH /orgs/{org}.
// The submitted values are not a complete or independently verified profile.
type EditedOrganization struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
}

// OrganizationEditUncertainError retains an unsuccessful result after a write
// that may have changed the organization profile.
type OrganizationEditUncertainError struct{ Err error }

func (e *OrganizationEditUncertainError) Error() string {
	return fmt.Sprintf("edit organization: %v; the profile may have changed, check AtomGit before retrying", e.Err)
}
func (e *OrganizationEditUncertainError) Unwrap() error { return e.Err }

// EditOrganization sends one PATCH without retries or redirect replay. Only
// HTTP 200 with a matching organization identity establishes an acknowledged update.
// Leading and trailing whitespace is removed from the organization path.
func EditOrganization(client *Client, organization string, request OrganizationEditRequest) (EditedOrganization, error) {
	organization = strings.TrimSpace(organization)
	if organization == "" {
		return EditedOrganization{}, fmt.Errorf("organization must not be empty")
	}
	if err := request.Validate(); err != nil {
		return EditedOrganization{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return EditedOrganization{}, fmt.Errorf("encode organization profile: %w: %w", ErrRequestPreparation, err)
	}
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var result EditedOrganization
	err = client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPatch, "/orgs/"+url.PathEscape(organization), bytes.NewReader(body), "application/json", "application/json", RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &result)
	if err != nil {
		var status *HTTPError
		isHTTP := errors.As(err, &status)
		uncertainHTTP := isHTTP && ((status.StatusCode >= 200 && status.StatusCode < 400) || (status.StatusCode >= 500 && status.StatusCode < 600))
		if !errors.Is(err, ErrRequestPreparation) && (!isHTTP || uncertainHTTP) {
			return EditedOrganization{}, &OrganizationEditUncertainError{Err: err}
		}
		return EditedOrganization{}, fmt.Errorf("edit organization: %w", err)
	}
	if result.ID <= 0 || !strings.EqualFold(result.Path, organization) {
		return EditedOrganization{}, &OrganizationEditUncertainError{Err: fmt.Errorf("%w: missing or mismatched organization identity", ErrInvalidResponse)}
	}
	return result, nil
}
