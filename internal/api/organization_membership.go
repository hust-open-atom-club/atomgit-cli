package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OrganizationInvitationRequest grants an organization role, not repository
// collaborator access. RoleID is only used with customized permission.
type OrganizationInvitationRequest struct {
	Permission string `json:"permission"`
	RoleID     string `json:"role_id,omitempty"`
}

// InviteOrganizationMember submits an invitation. HTTP success does not prove
// that the user accepted it. No response fields are required for this operation.
func InviteOrganizationMember(client *Client, organization, username string, request OrganizationInvitationRequest) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode organization invitation: %w: %w", ErrRequestPreparation, err)
	}
	return writeOrganizationMembership(client, http.MethodPost, organization, username, bytes.NewReader(body))
}

// RemoveOrganizationMember accepts the documented HTTP 200, including an empty
// response body. It does not remove repository collaborators or enterprise members.
func RemoveOrganizationMember(client *Client, organization, username string) error {
	return writeOrganizationMembership(client, http.MethodDelete, organization, username, nil)
}

func writeOrganizationMembership(client *Client, method, organization, username string, body io.Reader) error {
	// Redirects can replay writes independently of the API retry policy. Copy
	// the HTTP client so the caller's redirect behavior remains unchanged.
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	return client.doJSONRequestContext(client.requestContext(), &httpClient, method,
		OrganizationPath(organization, "memberships", username), body, contentType, "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil)
}
