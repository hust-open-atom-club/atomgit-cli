package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type authStatusReport struct {
	OK                 bool               `json:"ok"`
	Host               string             `json:"host"`
	CredentialsPresent *bool              `json:"credentialsPresent"`
	LocalAccount       *string            `json:"localAccount"`
	LocalStatus        string             `json:"localStatus"`
	Verification       statusVerification `json:"verification"`
	Message            string             `json:"message"`
}

type statusVerification struct {
	Requested  bool    `json:"requested"`
	Performed  bool    `json:"performed"`
	Status     string  `json:"status"`
	Account    *string `json:"account"`
	HTTPStatus *int    `json:"httpStatus"`
}

func newCmdAuthStatus(f *cmdutil.Factory) *cobra.Command {
	var asJSON, verify bool
	cmd := &cobra.Command{
		Use: "status", Short: "View local authentication status or verify identity online",
		Long:    "Inspect local credentials without modifying them. Local presence does not prove token validity. Use --verify to check identity with the read-only /user API (30 second timeout); this does not verify access to other resources. No token or token fragment is displayed.",
		Example: "  ag-cli auth status\n  ag-cli auth status --json\n  ag-cli auth status --verify\n  ag-cli auth status --verify --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r := inspectAuthStatus(cmd.Context(), f, verify)
			if asJSON {
				if err := cmdutil.WriteJSON(cmd.OutOrStdout(), r); err != nil {
					return err
				}
			} else {
				text := r.Message + "\n"
				if r.LocalAccount != nil {
					text += "  Local account: " + *r.LocalAccount + "\n"
				}
				if r.Verification.Account != nil {
					text += "  Verified account: " + *r.Verification.Account + "\n"
				}
				text += "  Online verification: " + r.Verification.Status + "\n"
				if _, err := fmt.Fprint(cmd.OutOrStdout(), text); err != nil {
					return err
				}
			}
			if !r.OK {
				return errors.New("auth status: " + r.Message)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output authentication status as JSON, including failures")
	cmd.Flags().BoolVar(&verify, "verify", false, "Verify the active identity online without refreshing or changing credentials")
	return cmd
}

func inspectAuthStatus(ctx context.Context, f *cmdutil.Factory, verify bool) authStatusReport {
	r := authStatusReport{Host: "atomgit.com", LocalStatus: "invalid", Verification: statusVerification{Requested: verify, Status: "not_requested"}}
	if verify {
		r.Verification.Status = "skipped"
	}
	cred, _, err := config.InspectCredentials()
	if errors.Is(err, config.ErrTokenNotFound) {
		present := false
		r.CredentialsPresent = &present
		r.LocalStatus = "missing"
		r.Message = "No local credentials; run ag-cli auth login."
		return r
	}
	if err != nil || cred == nil || strings.TrimSpace(cred.AccessToken) == "" || strings.TrimSpace(cred.User) == "" {
		r.Message = "Local credentials are incomplete, invalid or unreadable; check the active account configuration."
		return r
	}
	present := true
	r.CredentialsPresent = &present
	r.LocalAccount = safeStatusAccount(cred.User, cred)
	r.LocalStatus, r.OK = "configured", true
	r.Message = "Local credentials are configured; online validity has not been verified."
	if !verify {
		return r
	}
	r.OK = false
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client, err := f.NewAPIClient(cred.AccessToken)
	if err != nil {
		r.Verification.Status = "client_error"
		r.Message = "Unable to initialize the HTTP client."
		return r
	}
	if ctx.Err() != nil {
		r.Verification.Status, r.Message = verificationFailure(ctx.Err())
		return r
	}
	r.Verification.Performed = true
	login, err := client.WithContext(ctx).CurrentUserLogin()
	if err != nil {
		if httpErr, ok := errors.AsType[*api.HTTPError](err); ok {
			r.Verification.HTTPStatus = &httpErr.StatusCode
		}
		r.Verification.Status, r.Message = verificationFailure(err)
		return r
	}
	status := 200
	r.Verification.HTTPStatus = &status
	r.Verification.Account = safeStatusAccount(login, cred)
	if !strings.EqualFold(cred.User, login) {
		r.Verification.Status = "identity_mismatch"
		r.Message = "The server identity differs from the local account; credentials were not changed."
		return r
	}
	r.Verification.Status, r.OK = "verified", true
	r.Message = "Identity verified with the AtomGit user API."
	return r
}

// Account fields are untrusted metadata too. Never echo known credentials even
// if a malformed file or API response places them in an account name.
func safeStatusAccount(account string, cred *config.StoredCredentials) *string {
	for _, secret := range []string{cred.AccessToken, cred.RefreshToken} {
		if secret != "" && strings.Contains(account, secret) {
			redacted := "[redacted]"
			return &redacted
		}
	}
	account = api.SanitizeErrorText(account)
	return &account
}

// Classify errors without printing response bodies, URLs or transport errors.
func verificationFailure(err error) (string, string) {
	var httpErr *api.HTTPError
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled", "Online verification was canceled."
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout", "Online verification timed out."
	case errors.As(err, &httpErr):
		switch {
		case httpErr.StatusCode == 401:
			return "unauthorized", "The server rejected authentication."
		case httpErr.StatusCode == 403:
			return "forbidden", "The server denied access to the user API; this does not establish token expiry."
		case httpErr.StatusCode >= 500:
			return "server_error", "The user API returned a server error."
		default:
			return "http_error", "The user API returned an unexpected HTTP status."
		}
	case errors.Is(err, api.ErrInvalidUserIdentity):
		return "invalid_response", "The user API did not return a valid identity."
	case errors.As(err, &netErr) && netErr.Timeout():
		return "timeout", "Online verification timed out."
	default:
		return "network_error", "Online verification failed while communicating with the user API."
	}
}
