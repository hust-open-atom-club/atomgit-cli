package agcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

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
	//nolint:errorlint // Match the selected save failure, not its causes.
	if save, ok := err.(*config.CredentialSaveError); ok {
		d := commandDiagnostic{
			summary: "Unable to save credentials.",
			detail:  fmt.Sprintf("Failed to %s.", save.Stage),
		}
		if save.Path != "" {
			d.detail += fmt.Sprintf("\nPath: %q.", save.Path)
		}
		if save.CloseErr != nil {
			d.detail += "\nClosing the temporary credential file also failed."
		}
		if save.RemoveErr != nil {
			d.detail += "\nRemoving the temporary credential file also failed. It may still be present."
		}
		switch {
		case save.Stage == config.CredentialSaveLocate:
			d.hint = "Check the home directory and XDG_CONFIG_HOME settings."
		case errors.Is(save.Err, os.ErrPermission):
			d.hint = "Check ownership and write permissions for the affected path and its parent directory."
		}
		return d, true
	}
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
	//nolint:errorlint // Match the selected error without searching its causes.
	if repair, ok := err.(*config.CredentialPermissionRepairError); ok {
		d := commandDiagnostic{
			summary: "Unable to secure the credential file.",
			detail:  fmt.Sprintf("Credential file: %q.", repair.Path),
		}
		if errors.Is(repair.Err, os.ErrPermission) {
			d.hint = "Check the file ownership and permission to change its mode."
		}
		return d, true
	}
	//nolint:errorlint // errors.As could misclassify a format error with a nested safety error.
	if safety, ok := err.(*config.CredentialFileSafetyError); ok {
		d := commandDiagnostic{
			summary: "Credential configuration is invalid.",
			detail:  "The credential file failed a safety check.",
			hint:    "Check the credential file before retrying.",
		}
		switch {
		case errors.Is(safety.Err, config.ErrTokenFileSymlink):
			d.detail = "The credential file is a symbolic link."
			d.hint = "Use a regular credential file owned by the current user."
		case errors.Is(safety.Err, config.ErrTokenFileChanged):
			d.detail = "The credential file changed while it was being opened."
		case errors.Is(safety.Err, config.ErrTokenFileUnreadable):
			d.detail = "The credential file is not readable by its owner."
			d.hint = "Check ownership and file read permissions."
			if runtime.GOOS != "windows" {
				d.hint = "Check ownership and set the credential file permissions to 0600."
			}
		case errors.Is(safety.Err, config.ErrTokenFileNotRegular):
			d.detail = "The credential file is not a regular file."
			d.hint = "Use a regular credential file owned by the current user."
		case errors.Is(safety.Err, config.ErrTokenFileTooLarge):
			d.detail = "The credential file exceeds the inspection size limit."
			d.hint = "Check the credential file size; inspection accepts at most 1 MiB."
		}
		return d, true
	}
	//nolint:errorlint // Match the selected error without searching its causes.
	switch err.(type) {
	case *config.CredentialValidationError:
		return commandDiagnostic{
			summary: "Credential configuration is invalid.",
			detail:  "The stored credential data failed validation.",
			hint:    "Check the credential store version, account records, and active account.",
		}, true
	case *config.AccountNotFoundError:
		return commandDiagnostic{
			summary: "The requested account is not saved.",
			hint:    "Run `ag-cli auth list` to view saved accounts.",
		}, true
	case *config.AccountConflictError:
		return commandDiagnostic{
			summary: "Cannot rename the active account.",
			detail:  "An account with the new name already exists.",
			hint:    "Run `ag-cli auth list` to view saved accounts.",
		}, true
	case *config.ActiveAccountRemovalError:
		return commandDiagnostic{
			summary: "Cannot remove the active account.",
			detail:  "Other saved accounts remain.",
			hint:    "Run `ag-cli auth switch <account>` before removing this account.",
		}, true
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
