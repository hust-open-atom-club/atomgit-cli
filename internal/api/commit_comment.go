package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// CommitComment is a comment attached to a repository commit. ID decodes both
// the integer form documented for list/get responses and the string form the
// create endpoint returns; identifiers stay opaque strings end to end.
type CommitComment struct {
	ID        FlexibleIdentifier `json:"id"`
	Body      string             `json:"body"`
	User      CommitCommentUser  `json:"user"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
	HTMLURL   string             `json:"html_url"`
}

// CommitCommentUser is the endpoint-specific user shape embedded in commit
// comment responses. These endpoints document user.id as an integer, unlike
// the string-identified User shape; FlexibleIdentifier also tolerates string
// forms so neither shape breaks decoding.
type CommitCommentUser struct {
	ID    FlexibleIdentifier `json:"id"`
	Login string             `json:"login"`
	Name  string             `json:"name"`
}

// ErrNotCommitComment reports that a repository comment ID does not belong to
// a commit comment. The repository comment endpoint serves commit comments
// only and rejects IDs of issue or pull request comments.
var ErrNotCommitComment = errors.New("comment is not a commit comment")

// ListCommitComments retrieves at most limit comments attached to the commit
// identified by ref, following the endpoint's page/per_page pagination.
func ListCommitComments(client *Client, owner, repo, ref string, limit int) ([]CommitComment, error) {
	comments, err := GetPaginated[CommitComment](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/commits/%s/comments?page=%d&per_page=%d",
			url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref), page, perPage)
	})
	if err != nil {
		return nil, fmt.Errorf("list commit comments for %s: %w", ref, err)
	}
	return comments, nil
}

// CreateCommitComment posts a comment body on the commit identified by sha.
func CreateCommitComment(client *Client, owner, repo, sha, body string) (CreateCommentResponse, error) {
	var comment CreateCommentResponse
	path := fmt.Sprintf("/repos/%s/%s/commits/%s/comments",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha))
	if err := client.Post(path, CommentRequest{Body: body}, &comment); err != nil {
		return CreateCommentResponse{}, fmt.Errorf("create commit comment on %s: %w", sha, err)
	}
	return comment, nil
}

// GetCommitComment fetches a repository comment by ID and verifies it is a
// commit comment. The ID is the opaque string returned by the create endpoint
// (or a decimal form of the same). Issue and pull request comment IDs are
// rejected with ErrNotCommitComment; missing IDs surface the API's not-found
// error.
func GetCommitComment(client *Client, owner, repo, commentID string) (CommitComment, error) {
	var comment CommitComment
	path := fmt.Sprintf("/repos/%s/%s/comments/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(commentID))
	if err := client.Get(path, &comment); err != nil {
		if isNotCommitCommentResponse(err) {
			return CommitComment{}, fmt.Errorf("comment #%s: %w", commentID, ErrNotCommitComment)
		}
		return CommitComment{}, fmt.Errorf("get commit comment #%s: %w", commentID, err)
	}
	return comment, nil
}

// UpdateCommitComment replaces the body of an existing commit comment. Only
// the supported body field is sent.
func UpdateCommitComment(client *Client, owner, repo, commentID, body string) (CommitComment, error) {
	var comment CommitComment
	path := fmt.Sprintf("/repos/%s/%s/comments/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(commentID))
	if err := client.Patch(path, CommentRequest{Body: body}, &comment); err != nil {
		return CommitComment{}, fmt.Errorf("update commit comment #%s: %w", commentID, err)
	}
	return comment, nil
}

// DeleteCommitComment removes a repository commit comment by ID.
func DeleteCommitComment(client *Client, owner, repo, commentID string) error {
	path := fmt.Sprintf("/repos/%s/%s/comments/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(commentID))
	if err := client.Delete(path); err != nil {
		return fmt.Errorf("delete commit comment #%s: %w", commentID, err)
	}
	return nil
}

// isNotCommitCommentResponse reports whether err carries the API's rejection
// of a comment ID whose note type is not a commit comment.
func isNotCommitCommentResponse(err error) bool {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		return false
	}
	var payload struct {
		ErrorMessage string `json:"error_message"`
	}
	if json.Unmarshal([]byte(httpErr.Body), &payload) != nil {
		return false
	}
	return payload.ErrorMessage == "Note type is not correct."
}
