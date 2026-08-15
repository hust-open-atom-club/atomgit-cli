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

// ListDiscussionComments fetches the comment thread of a discussion.
func ListDiscussionComments(client *Client, owner, repo string, number int) ([]DiscussionComment, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/discuss/%s/comment",
		url.PathEscape(owner),
		url.PathEscape(repo),
		strconv.Itoa(number),
	)

	var comments []DiscussionComment
	if err := client.Get(path, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// ListDiscussionReplies fetches the replies to one comment of a discussion.
func ListDiscussionReplies(client *Client, owner, repo string, number int, commentID string) ([]DiscussionComment, error) {
	path := fmt.Sprintf(
		"/repos/%s/%s/discuss/%s/comment/%s/reply",
		url.PathEscape(owner),
		url.PathEscape(repo),
		strconv.Itoa(number),
		url.PathEscape(commentID),
	)

	var replies []DiscussionComment
	if err := client.Get(path, &replies); err != nil {
		return nil, err
	}
	return replies, nil
}
