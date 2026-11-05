package agcmd

import (
	"errors"
	"fmt"
	"io"
	"os"

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
	//nolint:errorlint // Only the selected operation supplies access facts, not an inner cause.
	if access, ok := err.(*config.CredentialAccessError); ok {
		d := commandDiagnostic{summary: "Unable to read credentials."}
		if access.Path == "" {
			d.detail = "No credential file paths are available."
			d.hint = "Check the home directory and XDG_CONFIG_HOME settings."
		} else {
			d.detail = fmt.Sprintf("Credential file: %q.", access.Path)
			if errors.Is(access.Err, os.ErrPermission) {
				d.hint = "Check the file and parent directory ownership and read permissions."
			}
		}
		return d, true
	}
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
