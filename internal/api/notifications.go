package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// notificationsPerPage matches the largest per_page value the repository
// notifications endpoint honors; larger requests are silently capped.
const notificationsPerPage = 100

// Notification is one entry of the repository notifications listing.
type Notification struct {
	ID         string                  `json:"id"`
	Content    string                  `json:"content"`
	Type       string                  `json:"type"`
	Unread     bool                    `json:"unread"`
	UpdateAt   string                  `json:"update_at"`
	HTMLURL    string                  `json:"html_url"`
	Actor      *NotificationActor      `json:"actor,omitempty"`
	Repository *NotificationRepository `json:"repository,omitempty"`
}

// NotificationActor identifies the user whose action produced a notification.
type NotificationActor struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// NotificationRepository identifies the repository a notification belongs to.
type NotificationRepository struct {
	ID        int    `json:"id"`
	FullName  string `json:"full_name"`
	HumanName string `json:"human_name"`
	URL       string `json:"url"`
}

// notificationPage is the {total, list} envelope the notifications endpoint
// returns instead of a bare array, so the shared GetPaginated helper does not
// apply.
type notificationPage struct {
	Total int            `json:"total"`
	List  []Notification `json:"list"`
}

// NotificationListOptions configures ListNotifications. Since and Before are
// RFC 3339 timestamps applied by the server; Type is applied client-side
// because the endpoint ignores a type query parameter.
type NotificationListOptions struct {
	// Limit is the maximum number of notifications to return and must be
	// positive.
	Limit int
	// UnreadOnly restricts the listing to unread notifications. The server
	// defaults to unread-only, so listing everything must pass unread=false.
	UnreadOnly bool
	// Type optionally keeps only notifications of the given type, for
	// example merge_requests_open.
	Type string
	// Since optionally restricts the listing to notifications updated at or
	// after the given RFC 3339 timestamp.
	Since string
	// Before optionally restricts the listing to notifications updated
	// before the given RFC 3339 timestamp.
	Before string
}

// ListNotifications fetches at most opts.Limit notifications for a repository,
// walking server-side pages and applying the client-side type filter before
// the limit truncation.
func ListNotifications(client *Client, owner, repo string, opts NotificationListOptions) ([]Notification, error) {
	if opts.Limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}
	return listNotifications(client, owner, repo, opts, opts.Limit, false)
}

// ListAllNotifications fetches every notification matching opts. It is used
// by commands such as mark-read --all that must not silently truncate at an
// arbitrary client-side limit.
func ListAllNotifications(client *Client, owner, repo string, opts NotificationListOptions) ([]Notification, error) {
	return listNotifications(client, owner, repo, opts, 0, true)
}

func listNotifications(client *Client, owner, repo string, opts NotificationListOptions, limit int, all bool) ([]Notification, error) {

	encodedOwner := url.PathEscape(owner)
	encodedRepo := url.PathEscape(repo)
	capacity := notificationsPerPage
	if !all && limit < capacity {
		capacity = limit
	}
	notifications := make([]Notification, 0, capacity)
	fetched := 0

	for page := 1; all || len(notifications) < limit; page++ {
		query := url.Values{}
		query.Set("unread", strconv.FormatBool(opts.UnreadOnly))
		if opts.Since != "" {
			query.Set("since", opts.Since)
		}
		if opts.Before != "" {
			query.Set("before", opts.Before)
		}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(notificationsPerPage))
		path := fmt.Sprintf("/repos/%s/%s/notifications?%s", encodedOwner, encodedRepo, query.Encode())

		var result notificationPage
		if err := client.Get(path, &result); err != nil {
			return nil, err
		}
		if len(result.List) == 0 {
			break
		}
		fetched += len(result.List)

		for _, notification := range result.List {
			if opts.Type != "" && notification.Type != opts.Type {
				continue
			}
			notifications = append(notifications, notification)
			if !all && len(notifications) == limit {
				break
			}
		}
		if len(result.List) < notificationsPerPage || (result.Total > 0 && fetched >= result.Total) {
			break
		}
	}

	if !all && len(notifications) > limit {
		notifications = notifications[:limit]
	}
	return notifications, nil
}

// MarkNotificationsRead marks exactly the given notification IDs as read.
// The endpoint accepts an x-www-form-urlencoded body with one ids field per
// notification and responds 200 with an empty body.
func MarkNotificationsRead(client *Client, owner, repo string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("at least one notification ID is required")
	}

	path := fmt.Sprintf("/repos/%s/%s/notifications", url.PathEscape(owner), url.PathEscape(repo))
	fields := url.Values{}
	for _, id := range ids {
		fields.Add("ids", id)
	}
	return client.PutForm(path, fields, nil)
}
