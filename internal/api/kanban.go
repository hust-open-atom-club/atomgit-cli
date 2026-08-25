package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

const kanbanMaxPerPage = 100

// Kanban is an organization dashboard board exposed by the AtomGit API.
type Kanban struct {
	ID          string `json:"id"`
	IID         *int   `json:"iid,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	Visibility  int    `json:"visibility"`
	UpdatedAt   string `json:"updated_at"`
}

type kanbanListResponse struct {
	CloseCount int      `json:"close_count"`
	OpenCount  int      `json:"open_count"`
	AllCount   int      `json:"all_count"`
	Content    []Kanban `json:"content"`
}

// KanbanItem is an issue or pull request associated with a board.
type KanbanItem struct {
	ID         int64             `json:"id"`
	Number     interface{}       `json:"number"`
	Title      string            `json:"title"`
	SourceType string            `json:"source_type"`
	Status     string            `json:"status"`
	HTMLURL    string            `json:"html_url"`
	URL        string            `json:"url"`
	Values     []KanbanItemValue `json:"values"`
}

// KanbanItemValue describes a board field returned for an item.
type KanbanItemValue struct {
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
	Value     string `json:"value"`
}

// GetNumber returns the item number in a stable string form.
func (item *KanbanItem) GetNumber() string {
	return formatIdentifier(item.Number)
}

// Column returns the board status/column value when the API provides one.
func (item *KanbanItem) Column() string {
	for _, value := range item.Values {
		fieldType := strings.ToLower(strings.TrimSpace(value.FieldType))
		if fieldType == "status" && strings.TrimSpace(value.Value) != "" {
			return strings.TrimSpace(value.Value)
		}
	}
	return ""
}

// ValidateKanbanOwner validates an organization path before authentication or
// network access. Organization paths are a single URL path segment.
func ValidateKanbanOwner(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("organization owner is required")
	}
	if value == "." || value == ".." || strings.ContainsAny(value, `/\\?#%`) {
		return "", fmt.Errorf("invalid organization owner %q", value)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("invalid organization owner %q", value)
		}
	}
	return value, nil
}

// ValidateKanbanID validates the numeric board identifier before any request.
func ValidateKanbanID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("kanban ID is required")
	}
	if value == "0" {
		return "", fmt.Errorf("invalid kanban ID %q (expected a positive integer)", value)
	}
	if _, err := strconv.ParseUint(value, 10, 64); err != nil {
		return "", fmt.Errorf("invalid kanban ID %q (expected a positive integer)", value)
	}
	return value, nil
}

// GetKanbans lists at most limit boards for an organization.
func GetKanbans(client *Client, owner string, limit int) ([]Kanban, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	owner, err := ValidateKanbanOwner(owner)
	if err != nil {
		return nil, err
	}

	boards := make([]Kanban, 0, min(limit, kanbanMaxPerPage))
	for page := 1; len(boards) < limit; page++ {
		var response kanbanListResponse
		perPage := min(kanbanMaxPerPage, limit-len(boards))
		path := fmt.Sprintf("/org/%s/kanban/list?page=%d&per_page=%d", url.PathEscape(owner), page, perPage)
		if err := client.Get(path, &response); err != nil {
			return nil, err
		}
		if len(response.Content) == 0 {
			break
		}
		boards = append(boards, response.Content...)
		if len(response.Content) < perPage {
			break
		}
	}
	if len(boards) > limit {
		boards = boards[:limit]
	}
	return boards, nil
}

// GetKanban retrieves a single board by its API identifier.
func GetKanban(client *Client, owner, kanbanID string) (Kanban, error) {
	owner, err := ValidateKanbanOwner(owner)
	if err != nil {
		return Kanban{}, err
	}
	kanbanID, err = ValidateKanbanID(kanbanID)
	if err != nil {
		return Kanban{}, err
	}

	var board Kanban
	path := fmt.Sprintf("/org/%s/kanban/%s/detail", url.PathEscape(owner), url.PathEscape(kanbanID))
	if err := client.Get(path, &board); err != nil {
		return Kanban{}, err
	}
	return board, nil
}

// GetKanbanItems lists at most limit items associated with a board.
func GetKanbanItems(client *Client, owner, kanbanID string, limit int) ([]KanbanItem, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	owner, err := ValidateKanbanOwner(owner)
	if err != nil {
		return nil, err
	}
	kanbanID, err = ValidateKanbanID(kanbanID)
	if err != nil {
		return nil, err
	}

	items := make([]KanbanItem, 0, min(limit, kanbanMaxPerPage))
	for page := 1; len(items) < limit; page++ {
		var pageItems []KanbanItem
		perPage := min(kanbanMaxPerPage, limit-len(items))
		path := fmt.Sprintf("/org/%s/kanban/%s/item_list?page=%d&per_page=%d", url.PathEscape(owner), url.PathEscape(kanbanID), page, perPage)
		if err := client.Get(path, &pageItems); err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			break
		}
		items = append(items, pageItems...)
		if len(pageItems) < perPage {
			break
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
