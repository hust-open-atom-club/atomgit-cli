package agcmd

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"context"
	"errors"
	"fmt"
)

const errorResponseReadDetail = "The error response body could not be read."

func apiDiagnostic(err error) (commandDiagnostic, bool) {
	d := commandDiagnostic{summary: "API request failed."}
	role := commandErrorRole(err)
	//nolint:errorlint // Only the selected API operation supplies response or execution facts.
	switch failure := role.(type) {
	case *api.HTTPError:
		return apiStatusDiagnostic(failure.StatusCode, failure.ReadError != nil), true
	case *actions.HTTPError:
		return apiStatusDiagnostic(failure.StatusCode, failure.ReadError != nil), true
	case *actions.JobNotFoundError:
		return commandDiagnostic{
			summary: fmt.Sprintf("Job %q was not found.", api.SanitizeErrorText(failure.JobID)),
			detail:  "The job API returned an empty response.",
		}, true
	case *api.ResponseReadError:
		switch {
		case errors.Is(failure.Err, context.Canceled):
			d.detail = "Reading the server response was canceled."
		case errors.Is(failure.Err, context.DeadlineExceeded), failure.Timeout():
			d.detail = "Reading the server response timed out."
		default:
			d.detail = "The server response could not be read."
		}
		return d, true
	case *api.RequestExecutionError:
		switch {
		case errors.Is(failure.Err, context.Canceled):
			d.detail = "The HTTP request was canceled."
		case errors.Is(failure.Err, context.DeadlineExceeded), failure.Timeout():
			d.detail = "The HTTP request timed out."
		default:
			d.detail = "The HTTP request could not be completed."
		}
		return d, true
	}
	//nolint:errorlint // Wrappers were traversed; compare only the selected marker, not its causes.
	switch role {
	case api.ErrRequestPreparation:
		d.detail = "The HTTP request could not be prepared."
		return d, true
	case api.ErrInvalidResponse:
		d.detail = "The server returned an invalid response."
		return d, true
	}
	return commandDiagnostic{}, false
}

func apiStatusDiagnostic(status int, readFailed bool) commandDiagnostic {
	d := commandDiagnostic{
		summary: "API request failed.",
		detail:  fmt.Sprintf("The server returned HTTP %d.", status),
	}
	if readFailed {
		d.detail += "\n" + errorResponseReadDetail
	}
	return d
}
