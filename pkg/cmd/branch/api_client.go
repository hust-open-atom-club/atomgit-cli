package branch

import (
	"fmt"
	"net/url"
	"strings"
)

func parseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repository)
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repository)
	}
	return owner, repo, nil
}

func escapePathSegment(value string) string {
	return url.PathEscape(value)
}
