package api

import "fmt"

const maxPerPage = 100

// GetPaginated retrieves at most limit items from a page-based API endpoint.
// pagePath must build the endpoint path for the requested page and page size.
func GetPaginated[T any](client *Client, limit int, pagePath func(page, perPage int) string) ([]T, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	items := make([]T, 0, min(limit, maxPerPage))
	for page := 1; len(items) < limit; page++ {
		var pageItems []T
		if err := client.Get(pagePath(page, maxPerPage), &pageItems); err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			break
		}

		items = append(items, pageItems...)
		if len(pageItems) < maxPerPage {
			break
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
