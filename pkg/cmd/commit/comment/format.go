package comment

import (
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

// commentJSON is the stable JSON shape for one commit comment.
type commentJSON struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func commentsJSON(comments []api.CommitComment) []commentJSON {
	result := make([]commentJSON, 0, len(comments))
	for _, comment := range comments {
		result = append(result, newCommentJSON(comment))
	}
	return result
}

func newCommentJSON(comment api.CommitComment) commentJSON {
	return commentJSON{
		ID:        string(comment.ID),
		Body:      comment.Body,
		Author:    comment.User.Login,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

// formatCommentTime normalizes the API timestamp for single-line output.
func formatCommentTime(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02 15:04")
}

// commentBody renders the comment body inline, marking the current user's
// comments the way the issue comment listing does.
func commentBody(comment api.CommitComment, currentUser string) string {
	body := strings.TrimSpace(comment.Body)
	if comment.User.Login != "" && comment.User.Login == currentUser {
		return "(你) " + body
	}
	return body
}
