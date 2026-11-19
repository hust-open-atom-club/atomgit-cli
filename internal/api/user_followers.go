package api

import (
	"fmt"
	"net/http"
	"net/url"
)

// UserFollower is the public identity returned by the user followers endpoint.
// IDs are opaque strings; optional or null string fields decode as empty values.
type UserFollower struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

// ListUserFollowers returns at most limit followers and discards partial results
// when any page fails. Only the documented HTTP 200 array response is accepted.
func ListUserFollowers(client *Client, username string, limit int) ([]UserFollower, error) {
	path := "/users/" + url.PathEscape(username) + "/followers"
	items, err := getPaginated[UserFollower](limit, defaultMaxPerPage, true, nil, func(page, perPage int) ([]UserFollower, error) {
		var response []*UserFollower
		if err := client.doJSONRequest(http.MethodGet, pageQuery(path, page, perPage), nil, "", "application/json",
			RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &response); err != nil {
			return nil, err
		}
		if response == nil {
			return nil, fmt.Errorf("%w: missing followers array", ErrInvalidResponse)
		}
		followers := make([]UserFollower, 0, len(response))
		for _, follower := range response {
			if follower == nil {
				return nil, fmt.Errorf("%w: null follower", ErrInvalidResponse)
			}
			followers = append(followers, *follower)
		}
		return followers, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list user followers: %w", err)
	}
	return items, nil
}
