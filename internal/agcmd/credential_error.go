package agcmd

import (
	"errors"
	"fmt"
	"io"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
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

func credentialDiagnostic(err error) (commandDiagnostic, bool) {
	if errors.Is(err, config.ErrNotAuthenticated) {
		return commandDiagnostic{
			summary: "No credentials are configured.",
			hint:    "Run `ag-cli auth login` to log in.",
		}, true
	}
	return commandDiagnostic{}, false
}
