package api

import (
	"net/url"
	"strings"
)

// RepositoryPath builds /repos/{owner}/{repo}[/{segments...}]. Each argument
// is escaped once as a single path segment. Callers must pass raw identifier
// values, not pre-escaped strings. Query parameters belong in url.Values
// appended by the caller, not in these segments.
func RepositoryPath(owner, repo string, segments ...string) string {
	parts := make([]string, 0, 3+len(segments))
	parts = append(parts, "/repos", url.PathEscape(owner), url.PathEscape(repo))
	for _, segment := range segments {
		parts = append(parts, url.PathEscape(segment))
	}
	return strings.Join(parts, "/")
}
