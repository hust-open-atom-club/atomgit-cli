package api

import (
	"fmt"
	"net/url"
)

// RepositoryTransferResponse is returned by
// POST /repos/{owner}/{repo}/transfer.
type RepositoryTransferResponse struct {
	NewOwner string `json:"new_owner"`
	NewName  string `json:"new_name"`
}

// OrganizationRepositoryTransferResponse is returned by
// POST /org/{org}/projects/{repo}/transfer.
type OrganizationRepositoryTransferResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TransferRepository transfers a repository through the general repository
// transfer endpoint.
func TransferRepository(client *Client, owner, repo, destination string) (RepositoryTransferResponse, error) {
	var response RepositoryTransferResponse
	path := fmt.Sprintf("/repos/%s/%s/transfer", url.PathEscape(owner), url.PathEscape(repo))
	err := client.Post(path, map[string]string{"new_owner": destination}, &response)
	return response, err
}

// TransferOrganizationRepository transfers a repository currently owned by
// an organization. AtomGit requires the authenticated user's password for this
// endpoint in addition to the destination namespace.
func TransferOrganizationRepository(client *Client, organization, repo, destination, password string) (OrganizationRepositoryTransferResponse, error) {
	var response OrganizationRepositoryTransferResponse
	path := fmt.Sprintf("/org/%s/projects/%s/transfer", url.PathEscape(organization), url.PathEscape(repo))
	err := client.Post(path, map[string]string{
		"transfer_to": destination,
		"password":    password,
	}, &response)
	return response, err
}
