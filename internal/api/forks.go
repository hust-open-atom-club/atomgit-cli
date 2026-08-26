package api

import (
	"fmt"
	"net/url"
	"strings"
)

// repositoryForkResponse matches GET /repos/{owner}/{repo}/forks. That
// endpoint returns a compact repository shape whose URL is an API URL and
// whose parent is nested, unlike the general Repository response.
type repositoryForkResponse struct {
	ID          int64  `json:"id"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Private     bool   `json:"private"`
	Public      bool   `json:"public"`
	UpdatedAt   string `json:"updated_at"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Namespace struct {
		Path    string `json:"path"`
		HTMLURL string `json:"html_url"`
	} `json:"namespace"`
	Parent *struct {
		FullName string `json:"full_name"`
		URL      string `json:"url"`
	} `json:"parent"`
}

// ListRepositoryForks fetches up to limit forks of a repository.
func ListRepositoryForks(client *Client, owner, repo string, limit int) ([]Repository, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}

	escapedOwner := url.PathEscape(owner)
	escapedRepo := url.PathEscape(repo)
	responses, err := GetPaginated[repositoryForkResponse](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/forks?page=%d&per_page=%d", escapedOwner, escapedRepo, page, perPage)
	})
	if err != nil {
		return nil, err
	}

	forks := make([]Repository, len(responses))
	for index, response := range responses {
		forks[index] = response.repository()
	}
	return forks, nil
}

func (response repositoryForkResponse) repository() Repository {
	fullName := strings.Trim(strings.TrimSpace(response.FullName), "/")
	name := fullName
	if slash := strings.LastIndex(fullName, "/"); slash >= 0 {
		name = fullName[slash+1:]
	}

	webURL := ""
	if fullName != "" {
		webURL = (&url.URL{Scheme: "https", Host: "atomgit.com", Path: "/" + fullName}).String()
	}

	repository := Repository{
		ID:          response.ID,
		Name:        name,
		FullName:    fullName,
		Description: response.Description,
		HTMLURL:     webURL,
		Private:     response.Private,
		Fork:        true,
		UpdatedAt:   response.UpdatedAt,
	}
	repository.Owner.Login = strings.TrimSpace(response.Owner.Login)
	repository.Namespace.Path = strings.TrimSpace(response.Namespace.Path)
	if response.Parent != nil {
		repository.ParentFullName = strings.TrimSpace(response.Parent.FullName)
	}
	return repository
}
