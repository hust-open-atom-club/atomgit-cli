package api

import (
	"fmt"
	"net/url"
)

// ListRepositoryForks fetches up to limit forks of a repository.
func ListRepositoryForks(client *Client, owner, repo string, limit int) ([]Repository, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	return GetPaginated[Repository](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/forks?page=%d&per_page=%d", escapedOwner, escapedRepo, page, perPage)
	})
}
