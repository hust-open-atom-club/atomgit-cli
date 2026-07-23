package api

import (
	"fmt"
	"net/url"
)

// ListReleases fetches up to limit releases, most recent first.
func ListReleases(client *Client, owner, repo string, limit int) ([]Release, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	encodedOwner := url.PathEscape(owner)
	encodedRepo := url.PathEscape(repo)
	return GetPaginated[Release](client, limit, func(page, perPage int) string {
		return fmt.Sprintf(
			"/repos/%s/%s/releases?page=%d&per_page=%d&direction=desc",
			encodedOwner, encodedRepo, page, perPage,
		)
	})
}

// GetReleaseByTag fetches a single release by its tag name.
func GetReleaseByTag(client *Client, owner, repo, tag string) (Release, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/releases/tags/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(tag),
	)

	var release Release
	if err := client.Get(path, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

// CreateRelease creates a release and returns the server representation.
func CreateRelease(client *Client, owner, repo string, request CreateReleaseRequest) (Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases", url.PathEscape(owner), url.PathEscape(repo))

	var release Release
	if err := client.Post(path, request, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

// UpdateRelease updates a release identified by tag.
func UpdateRelease(client *Client, owner, repo, tag string, request UpdateReleaseRequest) (Release, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/releases/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(tag),
	)

	var release Release
	if err := client.Patch(path, request, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}
