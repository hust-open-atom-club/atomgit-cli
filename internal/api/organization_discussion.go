package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrganizationDiscussion fetches a single organization discussion by number.
func GetOrganizationDiscussion(client *Client, organization string, number int) (DiscussionDetail, error) {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number))
	var detail DiscussionDetail
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json", discussionReadPolicy, &detail); err != nil {
		return DiscussionDetail{}, fmt.Errorf("get organization discussion: %w", err)
	}
	if detail.Number != number || detail.Number <= 0 {
		return DiscussionDetail{}, fmt.Errorf("get organization discussion: %w: missing or mismatched discussion number", ErrInvalidResponse)
	}
	return detail, nil
}

// CreateOrganizationDiscussion creates an organization discussion once.
func CreateOrganizationDiscussion(client *Client, organization string, request DiscussionCreateRequest) (DiscussionDetail, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return DiscussionDetail{}, fmt.Errorf("encode organization discussion creation: %w", err)
	}
	var detail DiscussionDetail
	err = doDiscussionWrite(client, http.MethodPost, OrganizationPath(organization, "discuss"), bytes.NewReader(body), "application/json", discussionWritePolicy, &detail)
	return detail, err
}

// UpdateOrganizationDiscussion sends the editable fields without replaying a write.
func UpdateOrganizationDiscussion(client *Client, organization string, number int, request DiscussionEditRequest) (DiscussionDetail, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return DiscussionDetail{}, fmt.Errorf("encode organization discussion update: %w", err)
	}
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number))
	var detail DiscussionDetail
	err = doDiscussionWrite(client, http.MethodPut, path, bytes.NewReader(body), "application/json", discussionWritePolicy, &detail)
	return detail, err
}

// DeleteOrganizationDiscussion accepts the documented empty HTTP 200 response.
func DeleteOrganizationDiscussion(client *Client, organization string, number int) error {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number))
	return doDiscussionWrite(client, http.MethodDelete, path, nil, "", discussionWritePolicy, nil)
}

// CreateOrganizationDiscussionComment posts a top-level Markdown comment.
func CreateOrganizationDiscussionComment(client *Client, organization string, number int, body string) (DiscussionComment, error) {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number), "comment")
	comment, err := writeDiscussionComment(client, http.MethodPost, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("create organization discussion comment: %w", err)
	}
	return comment, nil
}

// ReplyOrganizationDiscussionComment replies to a top-level discussion comment.
func ReplyOrganizationDiscussionComment(client *Client, organization string, number int, commentID, body string) (DiscussionComment, error) {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number), "comment", commentID, "reply")
	comment, err := writeDiscussionComment(client, http.MethodPost, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("reply to organization discussion comment %s: %w", commentID, err)
	}
	return comment, nil
}

// UpdateOrganizationDiscussionComment replaces the body of a comment or reply.
func UpdateOrganizationDiscussionComment(client *Client, organization string, number int, commentID, body string) (DiscussionComment, error) {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number), "comment", commentID)
	comment, err := writeDiscussionComment(client, http.MethodPut, path, body)
	if err != nil {
		return DiscussionComment{}, fmt.Errorf("update organization discussion comment %s: %w", commentID, err)
	}
	return comment, nil
}

// DeleteOrganizationDiscussionComment accepts the documented empty HTTP 200 response.
func DeleteOrganizationDiscussionComment(client *Client, organization string, number int, commentID string) error {
	path := OrganizationPath(organization, "discuss", strconv.Itoa(number), "comment", commentID)
	if err := doDiscussionWrite(client, http.MethodDelete, path, nil, "", discussionWritePolicy, nil); err != nil {
		return fmt.Errorf("delete organization discussion comment %s: %w", commentID, err)
	}
	return nil
}

// FindOrganizationDiscussionComment confirms discussion identity before finding
// a comment or reply through that discussion's paginated comment lists.
func FindOrganizationDiscussionComment(client *Client, organization string, number int, commentID string) (DiscussionCommentLocation, error) {
	detail, err := GetOrganizationDiscussion(client, organization, number)
	if err != nil {
		return DiscussionCommentLocation{}, err
	}
	if detail.Number != number {
		return DiscussionCommentLocation{}, fmt.Errorf("organization discussion #%d lookup returned number %d: %w", number, detail.Number, ErrInvalidResponse)
	}
	return findDiscussionComment(client, OrganizationPath(organization, "discuss", strconv.Itoa(number)), number, commentID)
}

// RequireTopLevelOrganizationDiscussionComment finds commentID and rejects replies.
func RequireTopLevelOrganizationDiscussionComment(client *Client, organization string, number int, commentID string) (DiscussionComment, error) {
	located, err := FindOrganizationDiscussionComment(client, organization, number, commentID)
	if err != nil {
		return DiscussionComment{}, err
	}
	if !located.TopLevel {
		return DiscussionComment{}, fmt.Errorf("comment %s on organization discussion #%d: %w", commentID, number, ErrDiscussionReplyParent)
	}
	return located.Comment, nil
}
