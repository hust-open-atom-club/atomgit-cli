package org

import (
	"fmt"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

const defaultOrganizationCollectionLimit = 30

type collectionOptions struct {
	Limit int
	JSON  bool
}

type organizationViewJSON struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Visibility  string `json:"visibility"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type organizationMemberJSON struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type organizationRepositoryJSON struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Visibility    string `json:"visibility"`
	Description   string `json:"description"`
	DefaultBranch string `json:"defaultBranch"`
	Language      string `json:"language"`
	Stars         int    `json:"stars"`
	Forks         int    `json:"forks"`
	UpdatedAt     string `json:"updatedAt"`
	URL           string `json:"url"`
}

func parseOrganization(value string) (string, error) {
	organization := strings.TrimSpace(value)
	if organization == "" || strings.Contains(organization, "/") {
		return "", fmt.Errorf("invalid organization: %q", value)
	}
	return organization, nil
}

func organizationEndpoint(organization, suffix string) string {
	return "/orgs/" + url.PathEscape(organization) + suffix
}

func organizationVisibility(organization api.Organization) string {
	if organization.Public {
		return "public"
	}
	return "private"
}

func repositoryPath(repository api.Repository) string {
	if path := strings.TrimSpace(repository.Path); path != "" {
		return path
	}
	return strings.TrimSpace(repository.Name)
}

func organizationRepositoryVisibility(repository api.Repository) string {
	if repository.Private {
		return "private"
	}
	if repository.Internal {
		return "internal"
	}
	return "public"
}

func repositoryURL(repository api.Repository, organization, host string) string {
	for _, webURL := range []string{repository.HTMLURL, repository.AlternateHTMLURL} {
		if webURL = strings.TrimSpace(webURL); webURL != "" {
			return webURL
		}
	}
	path := repositoryPath(repository)
	if path == "" {
		return "-"
	}
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		host = "atomgit.com"
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return host + "/" + url.PathEscape(organization) + "/" + url.PathEscape(path)
}
