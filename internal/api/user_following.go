package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// FollowedAccount preserves the string identities and object type returned by
// the user following endpoint. Nullable or omitted strings remain empty; URLs
// are never synthesized (organizations may omit following_url).
type FollowedAccount struct {
	ID           string `json:"id"`
	ObjectID     string `json:"object_id"`
	Login        string `json:"login"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	AvatarURL    string `json:"avatar_url"`
	HTMLURL      string `json:"html_url"`
	URL          string `json:"url"`
	FollowersURL string `json:"followers_url"`
	FollowingURL string `json:"following_url"`
}

type followingPage []FollowedAccount

func (p *followingPage) UnmarshalJSON(data []byte) error {
	var items []FollowedAccount
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	if items == nil {
		return fmt.Errorf("expected an array of followed accounts")
	}
	for i, item := range items {
		if item.Type != "User" && item.Type != "Organization" {
			return fmt.Errorf("followed account at index %d has a missing or unsupported object type", i)
		}
	}
	*p = items
	return nil
}

// ListUserFollowing reads page-based public data and returns no partial result
// when any page fails. The page size follows the endpoint's 30 default and 100 cap.
func ListUserFollowing(client *Client, username string, limit int) ([]FollowedAccount, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username must not be empty")
	}
	endpoint := "/users/" + url.PathEscape(username) + "/following"
	return getPaginated[FollowedAccount](limit, min(limit, 100), true, nil, func(page, perPage int) ([]FollowedAccount, error) {
		var items followingPage
		err := client.doJSONRequest(http.MethodGet, pageQuery(endpoint, page, perPage), nil, "", "application/json", RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &items)
		if err != nil {
			return nil, fmt.Errorf("list following for user %q: %w", username, err)
		}
		return items, nil
	})
}
