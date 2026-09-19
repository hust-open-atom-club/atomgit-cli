package api

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

var (
	labelReadPolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        true,
	}
	labelCreatePolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK, http.StatusCreated},
		CanRetry:        false,
	}
	labelUpdatePolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        false,
	}
	labelDeletePolicy = RequestPolicy{
		AllowedStatuses: []int{http.StatusOK, http.StatusNoContent},
		CanRetry:        false,
	}
)

// CreateLabelRequest is the form payload for creating a repository label.
type CreateLabelRequest struct {
	Name  string
	Color string
}

// UpdateLabelRequest is the multipart payload for editing a repository label.
// Pointer fields distinguish omitted values from explicitly supplied strings.
type UpdateLabelRequest struct {
	Name  *string
	Color *string
}

// ListLabels fetches at most limit labels for a repository.
func ListLabels(client *Client, owner, repo string, limit int) ([]Label, error) {
	path := RepositoryPath(owner, repo, "labels")
	return getPaginatedWithPolicy[Label](client, limit, labelReadPolicy, func(page, perPage int) string {
		return pageQuery(path, page, perPage)
	})
}

// CreateLabel creates a repository label using form-urlencoded fields.
func CreateLabel(client *Client, owner, repo string, request CreateLabelRequest) (Label, error) {
	path := RepositoryPath(owner, repo, "labels")
	fields := url.Values{}
	fields.Set("name", request.Name)
	fields.Set("color", request.Color)

	var created Label
	err := client.doJSONRequest(
		http.MethodPost,
		path,
		strings.NewReader(fields.Encode()),
		"application/x-www-form-urlencoded",
		"application/json",
		labelCreatePolicy,
		&created,
	)
	if err != nil {
		return Label{}, fmt.Errorf("create label: %w", err)
	}
	return created, nil
}

// UpdateLabel edits a repository label using multipart form fields.
func UpdateLabel(client *Client, owner, repo, name string, request UpdateLabelRequest) error {
	path := RepositoryPath(owner, repo, "labels", name)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if request.Name != nil {
		if err := writer.WriteField("name", *request.Name); err != nil {
			return fmt.Errorf("encode label name: %w", err)
		}
	}
	if request.Color != nil {
		if err := writer.WriteField("color", *request.Color); err != nil {
			return fmt.Errorf("encode label color: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close label update form: %w", err)
	}

	if err := client.doJSONRequest(
		http.MethodPatch,
		path,
		&body,
		writer.FormDataContentType(),
		"application/json",
		labelUpdatePolicy,
		nil,
	); err != nil {
		return fmt.Errorf("update label: %w", err)
	}
	return nil
}

// DeleteLabel deletes a repository label.
func DeleteLabel(client *Client, owner, repo, name string) error {
	path := RepositoryPath(owner, repo, "labels", name)
	if err := client.doJSONRequest(
		http.MethodDelete,
		path,
		nil,
		"",
		"application/json",
		labelDeletePolicy,
		nil,
	); err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	return nil
}
