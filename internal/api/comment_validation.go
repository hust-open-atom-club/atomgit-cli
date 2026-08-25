package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const commentValidationPageSize = 100

// GetIssueCommentForParent returns a comment only when it is attached to the
// requested issue. The repository-level comment endpoint does not carry the
// parent number, so callers must validate against the parent's comment list
// before performing a mutation.
func GetIssueCommentForParent(client *Client, owner, repo string, number, commentID int) (Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%s/comments",
		url.PathEscape(owner), url.PathEscape(repo), strconv.Itoa(number))
	return getCommentForParent(client, path, "issue", number, commentID)
}

// GetPullRequestCommentForParent returns a comment only when it is attached
// to the requested pull request. The view=all response can nest replies, so
// the search includes the complete comment tree.
func GetPullRequestCommentForParent(client *Client, owner, repo string, number, commentID int) (Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/comments?view=all",
		url.PathEscape(owner), url.PathEscape(repo), strconv.Itoa(number))
	return getCommentForParent(client, path, "pull request", number, commentID)
}

func getCommentForParent(client *Client, path, parentKind string, number, commentID int) (Comment, error) {
	for page := 1; page <= maxUntilEmptyRequestPages; page++ {
		var comments []Comment
		pagePath := commentPagePath(path, page)
		if err := client.Get(pagePath, &comments); err != nil {
			return Comment{}, fmt.Errorf("get comments for %s #%d: %w", parentKind, number, err)
		}
		if comment, ok := findCommentByID(comments, int64(commentID)); ok {
			return comment, nil
		}
		if len(comments) == 0 {
			return Comment{}, fmt.Errorf("comment #%d does not belong to %s #%d", commentID, parentKind, number)
		}
	}
	return Comment{}, fmt.Errorf("get comments for %s #%d: pagination reached the maximum of %d pages", parentKind, number, maxUntilEmptyRequestPages)
}

func commentPagePath(path string, page int) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%spage=%d&per_page=%d", path, separator, page, commentValidationPageSize)
}

func findCommentByID(comments []Comment, commentID int64) (Comment, bool) {
	for _, comment := range comments {
		if comment.ID == commentID {
			return comment, true
		}
		if nested, ok := findCommentByID(comment.Reply, commentID); ok {
			return nested, true
		}
	}
	return Comment{}, false
}
