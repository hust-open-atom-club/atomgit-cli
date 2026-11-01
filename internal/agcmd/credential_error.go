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
	if format, ok := errors.AsType[*config.CredentialFormatError](err); ok {
		d := commandDiagnostic{
			summary: "Credential configuration is invalid.",
			hint:    "Check the credential file syntax and field types.",
		}
		switch {
		case format.Offset > 0:
			d.detail = fmt.Sprintf("The credential file contains invalid JSON at byte %d.", format.Offset)
		case format.InvalidFieldType:
			d.detail = "The credential file has an invalid field type."
		default:
			d.detail = "The credential file could not be decoded."
		}
		return d, true
	}
	if errors.Is(err, config.ErrNotAuthenticated) {
		return commandDiagnostic{
			summary: "No credentials are configured.",
			hint:    "Run `ag-cli auth login` to log in.",
		}, true
	}
	return commandDiagnostic{}, false
}
