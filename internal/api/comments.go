package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// ListPullRequestComments reads all top-level comments, including review
// discussions and their nested replies. Replies are not separate list items.
func ListPullRequestComments(client *Client, owner, repo string, number int) ([]Comment, error) {
	path := RepositoryPath(owner, repo, "pulls", strconv.Itoa(number), "comments") + "?view=all"
	return listAllComments(client, path)
}

// ListIssueComments reads all top-level comments on an issue.
func ListIssueComments(client *Client, owner, repo string, number int) ([]Comment, error) {
	path := RepositoryPath(owner, repo, "issues", strconv.Itoa(number), "comments")
	return listAllComments(client, path)
}

func listAllComments(client *Client, path string) ([]Comment, error) {
	// Bound the whole traversal, not just each individual HTTP request. A
	// shorter caller deadline or cancellation still takes precedence.
	ctx, cancel := context.WithTimeout(client.requestContext(), defaultMetadataTimeout)
	defer cancel()
	comments := make([]Comment, 0)
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}
	for page := 1; page <= maxUntilEmptyRequestPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("read comment pages: %w", err)
		}
		var items []*Comment
		if err := client.doJSONRequestContext(ctx, client.httpClient, http.MethodGet,
			commentPagePath(path, page), nil, "", "application/json", policy, &items); err != nil {
			return nil, fmt.Errorf("read comments on page %d: %w", page, err)
		}
		if items == nil {
			return nil, fmt.Errorf("%w: expected a comment array on page %d", ErrInvalidResponse, page)
		}
		if len(items) == 0 {
			return comments, nil
		}
		for _, item := range items {
			if item == nil {
				return nil, fmt.Errorf("%w: null comment on page %d", ErrInvalidResponse, page)
			}
			comments = append(comments, *item)
		}
		// Do not stop on a short page: the server may cap per_page below the
		// requested size. Only an empty page establishes the end of the list.
	}
	return nil, fmt.Errorf("read comments: pagination reached the maximum of %d pages without an empty page", maxUntilEmptyRequestPages)
}
