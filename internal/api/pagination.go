package api

import (
	"fmt"
	"net/http"
)

const (
	defaultMaxPerPage         = 100
	maxUntilEmptyRequestPages = 100
)

// GetPaginated retrieves at most limit items from a page-based API endpoint.
// pagePath must build the endpoint path for the requested page and page size.
func GetPaginated[T any](client *Client, limit int, pagePath func(page, perPage int) string) ([]T, error) {
	return GetPaginatedWithPageSize[T](client, limit, defaultMaxPerPage, pagePath)
}

// GetPaginatedWithPageSize retrieves at most limit items using the specified
// maximum page size. Use this for endpoints whose per_page limit differs from
// the API-wide default.
func GetPaginatedWithPageSize[T any](client *Client, limit, maxPerPage int, pagePath func(page, perPage int) string) ([]T, error) {
	return getPaginated[T](limit, maxPerPage, true, nil, fetchJSONPage[T](client, pagePath))
}

// GetPaginatedUntilEmptyWithPageSize retrieves at most limit unique items and
// only treats an empty response page as the end. It preserves the first
// occurrence of each key and keeps requesting pages until limit unique items
// have been collected or an empty page is returned. Use this for search
// endpoints that can return short non-terminal or overlapping pages.
func GetPaginatedUntilEmptyWithPageSize[T any](client *Client, limit, maxPerPage int, key func(T) string, pagePath func(page, perPage int) string) ([]T, error) {
	if key == nil {
		return nil, fmt.Errorf("item key function must not be nil")
	}
	return getPaginated[T](limit, maxPerPage, false, key, fetchJSONPage[T](client, pagePath))
}

func pageQuery(path string, page, perPage int) string {
	return fmt.Sprintf("%s?page=%d&per_page=%d", path, page, perPage)
}

func fetchJSONPage[T any](client *Client, pagePath func(page, perPage int) string) func(page, perPage int) ([]T, error) {
	return func(page, perPage int) ([]T, error) {
		var pageItems []T
		if err := client.Get(pagePath(page, perPage), &pageItems); err != nil {
			return nil, err
		}
		return pageItems, nil
	}
}

func getPaginatedWithPolicy[T any](client *Client, limit int, policy RequestPolicy, pagePath func(page, perPage int) string) ([]T, error) {
	return getPaginated[T](limit, defaultMaxPerPage, true, nil, fetchJSONPageWithPolicy[T](client, policy, pagePath))
}

func fetchJSONPageWithPolicy[T any](client *Client, policy RequestPolicy, pagePath func(page, perPage int) string) func(page, perPage int) ([]T, error) {
	return func(page, perPage int) ([]T, error) {
		var pageItems []T
		if err := client.doJSONRequest(http.MethodGet, pagePath(page, perPage), nil, "", "application/json", policy, &pageItems); err != nil {
			return nil, err
		}
		if pageItems == nil {
			pageItems = []T{}
		}
		return pageItems, nil
	}
}

func getPaginated[T any](limit, maxPerPage int, stopOnShortPage bool, key func(T) string, fetch func(page, perPage int) ([]T, error)) ([]T, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	if maxPerPage <= 0 {
		return nil, fmt.Errorf("invalid maximum page size: %d (must be positive)", maxPerPage)
	}

	items := make([]T, 0, min(limit, maxPerPage))
	var seen map[string]struct{}
	if key != nil {
		seen = make(map[string]struct{}, min(limit, maxPerPage))
	}
	for page := 1; len(items) < limit; page++ {
		pageItems, err := fetch(page, maxPerPage)
		if err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			break
		}

		itemCountBeforePage := len(items)
		if key == nil {
			items = append(items, pageItems...)
		} else {
			for _, item := range pageItems {
				itemKey := key(item)
				if _, exists := seen[itemKey]; exists {
					continue
				}
				seen[itemKey] = struct{}{}
				items = append(items, item)
				if len(items) == limit {
					break
				}
			}
		}
		if stopOnShortPage && len(pageItems) < maxPerPage {
			break
		}
		if key != nil && len(items) == itemCountBeforePage {
			return nil, fmt.Errorf("pagination made no progress on page %d: received %d items but found no new unique items", page, len(pageItems))
		}
		if key != nil && page == maxUntilEmptyRequestPages && len(items) < limit {
			return nil, fmt.Errorf("pagination reached the maximum of %d pages with %d unique items, fewer than requested limit %d", maxUntilEmptyRequestPages, len(items), limit)
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
