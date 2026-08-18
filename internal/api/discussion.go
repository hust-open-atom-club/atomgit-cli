package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// DiscussionDetail is a single discussion as returned by the detail endpoint.
// It carries the listing fields plus the Markdown body and category pins.
type DiscussionDetail struct {
	Discussion
	MDContent string `json:"md_content"`
}

// DiscussionComment is one comment (or, via the reply endpoint, one reply) in
// a discussion thread. Comments and replies share the same shape.
type DiscussionComment struct {
	ID         string           `json:"id"`
	Author     DiscussionAuthor `json:"author"`
	Content    string           `json:"content"`
	MDContent  string           `json:"md_content"`
	CreatedAt  string           `json:"created_at"`
	IsDeleted  FlexibleBool     `json:"is_deleted"`
	IsHidden   FlexibleBool     `json:"is_hide"`
	LikeTotal  int              `json:"like_total"`
	ReplyTotal int              `json:"reply_total"`
}

// GetDiscussion fetches a single discussion by number.
func GetDiscussion(client *Client, owner, repo string, number int) (DiscussionDetail, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/discuss/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
		strconv.Itoa(number),
	)

	var detail DiscussionDetail
	if err := client.Get(path, &detail); err != nil {
		return DiscussionDetail{}, err
	}
	return detail, nil
}

// ListDiscussionComments fetches every comment in a discussion thread. total
// comes from the discussion detail endpoint and lets the shared paginator stop
// exactly after the reported number of comments.
func ListDiscussionComments(client *Client, owner, repo string, number, total int) ([]DiscussionComment, error) {
	if total <= 0 {
		return []DiscussionComment{}, nil
	}

	path := fmt.Sprintf(
		"/repos/%s/%s/discuss/%s/comment",
		url.PathEscape(owner),
		url.PathEscape(repo),
		strconv.Itoa(number),
	)

	return GetPaginated[DiscussionComment](client, total, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", path, page, perPage)
	})
}

// ListDiscussionReplies fetches every reply to one comment of a discussion.
// total is the comment's reply_total from the comments endpoint.
func ListDiscussionReplies(client *Client, owner, repo string, number int, commentID string, total int) ([]DiscussionComment, error) {
	if total <= 0 {
		return []DiscussionComment{}, nil
	}

	path := fmt.Sprintf(
		"/repos/%s/%s/discuss/%s/comment/%s/reply",
		url.PathEscape(owner),
		url.PathEscape(repo),
		strconv.Itoa(number),
		url.PathEscape(commentID),
	)

	return GetPaginated[DiscussionComment](client, total, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", path, page, perPage)
	})
}
