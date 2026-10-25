package api

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// DiscussionCommentThread pairs a top-level comment with its nested replies.
type DiscussionCommentThread struct {
	Comment DiscussionComment
	Replies []DiscussionComment
}

// ListOrganizationDiscussions fetches at most limit unique discussions.
func ListOrganizationDiscussions(client *Client, org string, limit int) ([]Discussion, error) {
	items, err := organizationDiscussionPages[Discussion](client, OrganizationPath(org, "discuss"), limit, func(item Discussion) string {
		if item.Number <= 0 {
			return ""
		}
		return strconv.Itoa(item.Number)
	})
	if err != nil {
		return nil, fmt.Errorf("list organization discussions: %w", err)
	}
	return items, nil
}

// ListOrganizationDiscussionThreads reads all comments and replies before
// returning. Counts are advisory: omitted or stale totals must not hide threads.
func ListOrganizationDiscussionThreads(client *Client, org string, number int) ([]DiscussionCommentThread, error) {
	path := OrganizationPath(org, "discuss", strconv.Itoa(number), "comment")
	commentKey := func(comment DiscussionComment) string { return strings.TrimSpace(comment.ID) }
	comments, err := organizationDiscussionPages[DiscussionComment](client, path, math.MaxInt, commentKey)
	if err != nil {
		return nil, fmt.Errorf("list organization discussion comments: %w", err)
	}
	threads := make([]DiscussionCommentThread, 0, len(comments))
	for _, comment := range comments {
		path := OrganizationPath(org, "discuss", strconv.Itoa(number), "comment", comment.ID, "reply")
		replies, err := organizationDiscussionPages[DiscussionComment](client, path, math.MaxInt, commentKey)
		if err != nil {
			return nil, fmt.Errorf("list replies of organization discussion comment %s: %w", comment.ID, err)
		}
		threads = append(threads, DiscussionCommentThread{Comment: comment, Replies: replies})
	}
	return threads, nil
}

// Organization endpoints use page/per_page. Continue through short pages and
// stop at an empty page or the requested limit; the shared paginator rejects
// non-progressing pages and bounds scans to avoid silently incomplete threads.
func organizationDiscussionPages[T any](client *Client, path string, limit int, key func(T) string) ([]T, error) {
	return getPaginated[T](limit, defaultMaxPerPage, false, key, func(page, perPage int) ([]T, error) {
		var items []T
		if err := client.doJSONRequest(http.MethodGet, pageQuery(path, page, perPage), nil, "", "application/json", discussionReadPolicy, &items); err != nil {
			return nil, err
		}
		if items == nil {
			return nil, fmt.Errorf("%w: expected an organization discussion array on page %d", ErrInvalidResponse, page)
		}
		for _, item := range items {
			if key(item) == "" {
				return nil, fmt.Errorf("%w: missing organization discussion item identity on page %d", ErrInvalidResponse, page)
			}
		}
		return items, nil
	})
}
