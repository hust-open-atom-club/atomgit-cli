package api

import "net/http"

// ListOrganizationFollowers retrieves at most limit followers using the
// documented API v5 pagination and exact HTTP 200 success status.
func ListOrganizationFollowers(client *Client, organization string, limit int) ([]OrganizationFollower, error) {
	path := OrganizationPath(organization, "followers")
	return getPaginatedWithPolicy[OrganizationFollower](client, limit,
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true},
		func(page, perPage int) string { return pageQuery(path, page, perPage) })
}
