package cmdutil

import (
	"fmt"
	"io"
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
