package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

var discussionReadPolicy = RequestPolicy{
	AllowedStatuses: []int{http.StatusOK},
	CanRetry:        true,
}

var (
	discussionWritePolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        false,
	}
	discussionDeletePolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK, http.StatusNoContent},
		CanRetry:        false,
	}
)

// ErrDiscussionCommentNotFound reports that a comment or reply ID was not
// found in the specified discussion. The write endpoints accept any comment
// ID in the path, so callers must confirm membership before mutating.
var ErrDiscussionCommentNotFound = errors.New("comment does not belong to discussion")

// ErrDiscussionReplyParent reports that a reply was requested against a
// comment that is itself a reply. Replies are only one level deep.
var ErrDiscussionReplyParent = errors.New("reply parent must be a top-level discussion comment")

// DiscussionCommentLocation is where a comment was found inside one discussion.
type DiscussionCommentLocation struct {
	Comment  DiscussionComment
	TopLevel bool
	ParentID string
}

// DiscussionDetail is a single discussion as returned by the detail endpoint.
// It carries the listing fields plus the Markdown body and category pins.
type DiscussionDetail struct {
	Discussion
	MDContent        string `json:"md_content"`
	MDContentPresent bool   `json:"-"`
}

// UnmarshalJSON distinguishes an empty body from a missing body so edits
// cannot accidentally clear content omitted by the server.
func (d *DiscussionDetail) UnmarshalJSON(data []byte) error {
	type plainDiscussionDetail DiscussionDetail
	var decoded struct {
		plainDiscussionDetail
		MDContent *string `json:"md_content"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*d = DiscussionDetail(decoded.plainDiscussionDetail)
	if decoded.MDContent != nil {
		d.MDContent = *decoded.MDContent
		d.MDContentPresent = true
	}
	return nil
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

// UpdateDiscussion sends all editable fields once; an uncertain write must
// be inspected rather than automatically replayed.
func UpdateDiscussion(client *Client, owner, repo string, number int, request DiscussionEditRequest) (DiscussionDetail, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return DiscussionDetail{}, fmt.Errorf("encode discussion update: %w", err)
	}
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number))
	// Redirects can replay PUT independently of the API retry policy.
	var detail DiscussionDetail
	if err := doDiscussionWrite(client, http.MethodPut, path, bytes.NewReader(body),
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &detail); err != nil {
		return DiscussionDetail{}, err
	}
	return detail, nil
}

// doDiscussionWrite disables redirect following. Go otherwise replays the
// original method and body, including DELETE, outside the retry policy.
func doDiscussionWrite(client *Client, method, path string, body io.Reader, policy RequestPolicy, result any) error {
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client.doJSONRequestContext(client.requestContext(), &httpClient, method, path, body, "application/json", "application/json", policy, result)
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

type discussionCommentBody struct {
	MDContent string `json:"md_content"`
}

// CreateDiscussionComment posts a top-level Markdown comment on a discussion.
func CreateDiscussionComment(client *Client, owner, repo string, number int, body string) (DiscussionComment, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment")
	comment, err := writeDiscussionComment(client, http.MethodPost, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("create discussion comment: %w", err)
	}
	return comment, nil
}

// ReplyDiscussionComment posts a reply to a top-level comment of a discussion.
func ReplyDiscussionComment(client *Client, owner, repo string, number int, commentID, body string) (DiscussionComment, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment", commentID, "reply")
	comment, err := writeDiscussionComment(client, http.MethodPost, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("reply to discussion comment %s: %w", commentID, err)
	}
	return comment, nil
}

// UpdateDiscussionComment replaces the Markdown body of a comment or reply.
func UpdateDiscussionComment(client *Client, owner, repo string, number int, commentID, body string) (DiscussionComment, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment", commentID)
	comment, err := writeDiscussionComment(client, http.MethodPut, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("update discussion comment %s: %w", commentID, err)
	}
	return comment, nil
}

// DeleteDiscussionComment removes a comment or reply. A successful empty body
// is not an error.
func DeleteDiscussionComment(client *Client, owner, repo string, number int, commentID string) error {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment", commentID)
	if err := doDiscussionWrite(client, http.MethodDelete, path, nil, discussionDeletePolicy, nil); err != nil {
		return fmt.Errorf("delete discussion comment %s: %w", commentID, err)
	}
	return nil
}

func writeDiscussionComment(client *Client, method, path, body string) (DiscussionComment, error) {
	payload, err := json.Marshal(discussionCommentBody{MDContent: body})
	if err != nil {
		return DiscussionComment{}, err
	}
	var comment DiscussionComment
	if err := doDiscussionWrite(client, method, path, bytes.NewReader(payload), discussionWritePolicy, &comment); err != nil {
		return DiscussionComment{}, err
	}
	return comment, nil
}

// FindDiscussionComment walks a discussion's top-level comments and their
// one-level replies until commentID is found. A missing comment is
// ErrDiscussionCommentNotFound; lookup failures are returned unchanged so a
// caller can refuse to write.
func FindDiscussionComment(client *Client, owner, repo string, number int, commentID string) (DiscussionCommentLocation, error) {
	if _, err := GetDiscussion(client, owner, repo, number); err != nil {
		return DiscussionCommentLocation{}, err
	}
	comments, err := listAllDiscussionComments(client, owner, repo, number)
	if err != nil {
		return DiscussionCommentLocation{}, err
	}
	for _, comment := range comments {
		if comment.ID == commentID {
			return DiscussionCommentLocation{Comment: comment, TopLevel: true}, nil
		}
	}
	for _, comment := range comments {
		replies, err := listAllDiscussionReplies(client, owner, repo, number, comment.ID)
		if err != nil {
			return DiscussionCommentLocation{}, err
		}
		for _, reply := range replies {
			if reply.ID == commentID {
				return DiscussionCommentLocation{Comment: reply, ParentID: comment.ID}, nil
			}
		}
	}
	return DiscussionCommentLocation{}, fmt.Errorf("comment %s on discussion #%d: %w", commentID, number, ErrDiscussionCommentNotFound)
}

// RequireTopLevelDiscussionComment finds commentID and rejects replies.
func RequireTopLevelDiscussionComment(client *Client, owner, repo string, number int, commentID string) (DiscussionComment, error) {
	located, err := FindDiscussionComment(client, owner, repo, number, commentID)
	if err != nil {
		return DiscussionComment{}, err
	}
	if !located.TopLevel {
		return DiscussionComment{}, fmt.Errorf("comment %s on discussion #%d: %w", commentID, number, ErrDiscussionReplyParent)
	}
	return located.Comment, nil
}

// Membership scans continue until an empty page. A short page is not terminal:
// the server may ignore per_page or cap a page below 100. comment_total and
// reply_total are display hints; a stale count must not hide a comment or
// reply and allow a write against the wrong discussion.
func listAllDiscussionComments(client *Client, owner, repo string, number int) ([]DiscussionComment, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment")
	return listDiscussionPages(client, func(page int) string {
		return pageQuery(path, page, defaultMaxPerPage)
	})
}

func listAllDiscussionReplies(client *Client, owner, repo string, number int, commentID string) ([]DiscussionComment, error) {
	path := RepositoryPath(owner, repo, "discuss", strconv.Itoa(number), "comment", commentID, "reply")
	return listDiscussionPages(client, func(page int) string {
		return pageQuery(path, page, defaultMaxPerPage)
	})
}

func listDiscussionPages(client *Client, pagePath func(page int) string) ([]DiscussionComment, error) {
	items := make([]DiscussionComment, 0)
	for page := 1; page <= maxUntilEmptyRequestPages; page++ {
		var pageItems []DiscussionComment
		if err := client.doJSONRequest(http.MethodGet, pagePath(page), nil, "", "application/json", discussionReadPolicy, &pageItems); err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			return items, nil
		}
		items = append(items, pageItems...)
	}
	return nil, fmt.Errorf("discussion comment pagination reached the maximum of %d pages", maxUntilEmptyRequestPages)
}
