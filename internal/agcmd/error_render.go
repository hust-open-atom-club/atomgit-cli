package agcmd

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"fmt"
	"io"
)

// renderCommandError writes an execution error to the configured command writer.
// Main remains responsible for flushing and choosing the exit code.
func renderCommandError(w io.Writer, err error) {
	role := commandErrorRole(err)
	if diagnostic, ok := apiDiagnostic(role); ok {
		renderDiagnostic(w, diagnostic)
		return
	}
	if diagnostic, ok := credentialDiagnostic(role); ok {
		renderDiagnostic(w, diagnostic)
		return
	}
	_, _ = fmt.Fprintf(w, "%s\n", err)
}

// commandErrorRole stops at the first explicit operation boundary. Its causes
// remain available for inspection, but cannot replace that operation's role.
func commandErrorRole(err error) error {
	for err != nil {
		// These exact markers own the following causes in existing multi-cause
		// wrappers. Do not search through another operation to find a marker.
		//nolint:errorlint // Match this layer's marker without searching through an owning operation.
		switch err {
		case api.ErrRequestPreparation, api.ErrInvalidResponse, config.ErrNotAuthenticated:
			return err
		}
		//nolint:errorlint // Inspect this layer before its causes so an inner role cannot replace an outer one.
		switch err.(type) {
		case *config.CredentialSaveError:
			return err
		case *api.HTTPError, *actions.HTTPError, *actions.JobNotFoundError, *api.RequestExecutionError, *api.ResponseReadError,
			*api.RepositoryModuleUpdateUncertainError, *api.OrganizationCreationUncertainError, *api.OrganizationEditUncertainError, *api.RepositorySettingsUpdateUncertainError, *config.CredentialFormatError, *config.CredentialAccessError, *config.CredentialFileSafetyError,
			*config.CredentialValidationError, *config.CredentialPermissionRepairError,
			*config.AccountNotFoundError, *config.AccountConflictError, *config.ActiveAccountRemovalError:
			return err
		}
		//nolint:errorlint // Inspect the wrapping protocol to traverse error layers explicitly.
		switch wrapped := err.(type) {
		case interface{ Unwrap() error }:
			err = wrapped.Unwrap()
		case interface{ Unwrap() []error }:
			// Unclassified multi-cause wrappers retain depth-first traversal
			// order. A classified composite handles its own primary/cleanup
			// fields instead of reaching this branch.
			causes := wrapped.Unwrap()
			for _, cause := range causes {
				if role := commandErrorRole(cause); role != nil {
					if len(causes) > 1 {
						//nolint:errorlint // Match the selected role so Git rollback failures stay visible.
						switch role.(type) {
						case *config.CredentialSaveError, *config.AccountNotFoundError, *config.AccountConflictError, *config.ActiveAccountRemovalError:
							return nil
						}
					}
					return role
				}
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}
