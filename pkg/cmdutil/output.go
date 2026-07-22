package cmdutil

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// PrintResultWithOptionalURL prints a result summary and appends the URL only
// when the API returned one.
func PrintResultWithOptionalURL(out io.Writer, summary, rawURL string) {
	if url := strings.TrimSpace(rawURL); url != "" {
		fmt.Fprintf(out, "%s: %s\n", summary, url)
		return
	}
	fmt.Fprintln(out, summary)
}

// ResolveWebURL returns an API-provided URL when present, otherwise it builds
// an AtomGit browser URL from the configured host and path segments.
func ResolveWebURL(rawURL, host string, pathSegments ...string) string {
	if result := strings.TrimSpace(rawURL); result != "" {
		return result
	}

	base := strings.TrimRight(strings.TrimSpace(host), "/")
	if base == "" {
		base = "atomgit.com"
	}
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}

	escaped := make([]string, 0, len(pathSegments))
	for _, segment := range pathSegments {
		escaped = append(escaped, url.PathEscape(segment))
	}
	if len(escaped) == 0 {
		return base
	}
	return base + "/" + strings.Join(escaped, "/")
}
