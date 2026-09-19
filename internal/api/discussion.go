package api

import (
	"net/http"
	"strconv"
)

var discussionReadPolicy = RequestPolicy{
	AllowedStatuses: []int{http.StatusOK},
	CanRetry:        true,
}

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

// ListDiscussions fetches at most limit discussions for a repository.
func ListDiscussions(client *Client, owner, repo string, limit int) ([]Discussion, error) {
	path := RepositoryPath(owner, repo, "discuss")
	return getPaginatedWithPolicy[Discussion](client, limit, discussionReadPolicy, func(page, perPage int) string {
		return pageQuery(path, page, perPage)
	})
}

// GetDiscussion fetches a single discussion by number.
func GetDiscussion(client *Client, owner, repo string, number int) (DiscussionDetail, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number))
	var detail DiscussionDetail
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json", discussionReadPolicy, &detail); err != nil {
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

	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment")
	return getPaginatedWithPolicy[DiscussionComment](client, total, discussionReadPolicy, func(page, perPage int) string {
		return pageQuery(path, page, perPage)
	})
}

// ListDiscussionReplies fetches every reply to one comment of a discussion.
// total is the comment's reply_total from the comments endpoint.
func ListDiscussionReplies(client *Client, owner, repo string, number int, commentID string, total int) ([]DiscussionComment, error) {
	if total <= 0 {
		return []DiscussionComment{}, nil
	}

	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment", commentID, "reply")
	return getPaginatedWithPolicy[DiscussionComment](client, total, discussionReadPolicy, func(page, perPage int) string {
		return pageQuery(path, page, perPage)
	})
}
