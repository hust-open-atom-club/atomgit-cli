package agcmd

import (
	"fmt"
	"io"
)

// commandDiagnostic is presentation data, never a replacement for an error.
type commandDiagnostic struct {
	summary string
	detail  string
	hint    string
}

func renderDiagnostic(w io.Writer, d commandDiagnostic) {
	_, _ = fmt.Fprintf(w, "Error: %s\n", d.summary)
	for _, paragraph := range []string{d.detail, d.hint} {
		if paragraph != "" {
			_, _ = fmt.Fprintf(w, "\n%s\n", paragraph)
		}
	}
}
