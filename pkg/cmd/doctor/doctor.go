package doctor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type check struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type report struct {
	Version  string  `json:"version"`
	Platform string  `json:"platform"`
	Live     bool    `json:"live"`
	Checks   []check `json:"checks"`
	OK       bool    `json:"ok"`
}

func (r *report) add(id, status, message, hint string) {
	r.Checks = append(r.Checks, check{id, status, message, hint})
	if status == "fail" {
		r.OK = false
	}
}

// NewCmdDoctor provides bounded, read-only diagnostics. Raw config, HTTP bodies,
// URLs and transport errors are deliberately excluded from the report.
func NewCmdDoctor(f *cmdutil.Factory) *cobra.Command {
	var live, asJSON bool
	cmd := &cobra.Command{
		Use:     "doctor [<owner>/<repo>]",
		Short:   "CLI health check: config, auth, and connectivity",
		Long:    "Check local configuration without modifying it. Use --live for read-only API probes (30 second overall timeout). Missing optional capabilities are skipped. Failures return a nonzero exit status; warnings alone do not.",
		Args:    cobra.MaximumNArgs(1),
		Example: "ag-cli doctor\nag-cli doctor --live\nag-cli doctor owner/repo --live --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if _, err := cmdutil.ParseRepository(args[0]); err != nil {
					return errors.New("invalid repository: expected owner/repo without URL credentials or unsafe path characters")
				}
			}
			r := diagnose(f, args, live)
			if asJSON {
				if err := cmdutil.WriteJSON(cmd.OutOrStdout(), r); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "AtomGit CLI %s (%s)\n", r.Version, r.Platform); err != nil {
					return err
				}
				for _, c := range r.Checks {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", c.Status, c.ID, c.Message); err != nil {
						return err
					}
					if c.Hint != "" {
						if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Next: %s\n", c.Hint); err != nil {
							return err
						}
					}
				}
			}
			if !r.OK {
				return errors.New("doctor found failed checks")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&live, "live", false, "Run read-only connectivity and authentication probes")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output a redacted health report as JSON")
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func diagnose(f *cmdutil.Factory, args []string, live bool) report {
	r := report{Version: api.SanitizeErrorText(version.Get().Version), Platform: runtime.GOOS + "/" + runtime.GOARCH, Live: live, OK: true, Checks: []check{}}
	if _, err := exec.LookPath("git"); err != nil {
		r.add("git", "warn", "Git executable not found", "Install Git to use clone and repository inference.")
	} else {
		r.add("git", "pass", "Git executable available", "")
	}
	if _, err := config.LoadAliases(); err != nil {
		r.add("config", "fail", "Alias configuration cannot be read or parsed", "Check ag-cli/config.json in your XDG configuration directory.")
	} else {
		r.add("config", "pass", "Alias configuration is readable or absent", "")
	}
	cred, mode, err := config.InspectCredentials()
	token := ""
	switch {
	case errors.Is(err, config.ErrTokenNotFound):
		r.add("credentials", "warn", "No stored credentials", "Run ag-cli auth login if authenticated access is needed.")
	case err != nil:
		r.add("credentials", "fail", "Credential file or active account is invalid or unreadable", "Check token.json format, ownership and active account; doctor does not repair it.")
	default:
		token = cred.AccessToken
		r.add("credentials", "pass", "Active account and token are configured locally; validity is not verified", "")
		if runtime.GOOS != "windows" && mode.Perm() != 0o600 {
			r.add("credential_permissions", "fail", "Credential file permissions are not 0600", "Set the selected token file permissions to 0600; doctor leaves them unchanged.")
			token = "" // Do not use insecure credentials for probes.
		} else if runtime.GOOS == "windows" {
			r.add("credential_permissions", "skip", "Windows ACL inspection is not supported", "")
		} else {
			r.add("credential_permissions", "pass", "Credential file permissions are 0600", "")
		}
		if cred.CreatedAt > 0 && cred.ExpiresIn > 0 {
			if time.Now().Unix()-cred.CreatedAt >= cred.ExpiresIn {
				r.add("credential_expiry", "warn", "Stored expiry time has passed; server validity is not verified", "Run ag-cli auth refresh for OAuth credentials, or authorize again.")
			} else {
				r.add("credential_expiry", "pass", "Stored expiry time has not passed; server validity is not verified", "")
			}
		} else {
			r.add("credential_expiry", "skip", "Credential expiry is not recorded", "")
		}
	}
	repo, _, repoErr := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
	if repoErr == nil {
		_, repoErr = cmdutil.ParseRepository(repo.String())
	}
	if repoErr != nil {
		r.add("repository", "skip", "No unambiguous AtomGit repository context", "Pass owner/repo to enable repository probes.")
	} else {
		source := "Git context"
		if len(args) > 0 {
			source = "explicit argument"
		}
		// Never print a resolver error or a remote URL, which can contain secrets.
		r.add("repository", "pass", "AtomGit repository resolved from "+source, "")
	}
	if !live {
		for _, id := range []string{"connectivity", "service", "authentication", "repository_access", "actions", "discussion"} {
			r.add(id, "skip", "Online check disabled", "Run ag-cli doctor --live for read-only probes.")
		}
		return r
	}
	ctx, cancel := context.WithTimeout(f.CommandContext(), 30*time.Second)
	defer cancel()
	copyFactory := cmdutil.Factory{}
	if f != nil {
		copyFactory = *f
	}
	copyFactory.Context = func() context.Context { return ctx }
	client, err := copyFactory.NewAPIClient(token)
	if err != nil {
		r.add("connectivity", "fail", "Unable to initialize HTTP client", "Check HTTP client configuration.")
		return r
	}
	var user struct {
		Login string `json:"login"`
	}
	err = client.Get("/user", &user)
	if err == nil || statusCode(err) != 0 {
		r.add("connectivity", "pass", "API v5 responded over HTTP", "")
	} else {
		r.result("connectivity", err)
	}
	// HTTP reachability does not establish service health. An anonymous 401/403
	// prevents this probe from assessing the service; other HTTP failures must
	// remain visible even when no credentials are available.
	switch {
	case err == nil:
		r.add("service", "pass", "API v5 returned a successful response", "")
	case statusCode(err) == 401 || statusCode(err) == 403:
		r.add("service", "skip", "Service health cannot be assessed because access was rejected", "Resolve authentication or access permissions and retry.")
	case statusCode(err) != 0:
		r.result("service", err)
	default:
		r.add("service", "skip", "No usable API response; see connectivity check", "")
	}
	if token == "" {
		r.add("authentication", "skip", "No usable local credentials", "Resolve the credentials check first.")
	} else {
		if err == nil && user.Login == "" {
			err = errors.New("invalid user response")
		}
		r.result("authentication", err)
	}
	if token == "" || repoErr != nil {
		for _, id := range []string{"repository_access", "actions", "discussion"} {
			r.add(id, "skip", "Requires usable credentials and repository context", "Pass owner/repo and resolve credential checks.")
		}
		return r
	}
	var repository struct {
		FullName string `json:"full_name"`
	}
	err = client.Get(api.RepositoryPath(repo.Owner, repo.Name), &repository)
	if err == nil {
		// AtomGit may canonicalize both owner and repository casing.
		owner, name, found := strings.Cut(repository.FullName, "/")
		if !found || !strings.EqualFold(owner, repo.Owner) || !strings.EqualFold(name, repo.Name) {
			err = errors.New("unexpected repository response")
		}
	}
	r.result("repository_access", err)
	ac, err := copyFactory.NewActionsClient(token)
	if err == nil {
		_, err = ac.ListWorkflows(repo.Owner, repo.Name, actions.ListWorkflowsOptions{Page: 1, PerPage: 1})
	}
	r.result("actions", err)
	_, err = api.ListDiscussions(client, repo.Owner, repo.Name, 1)
	r.result("discussion", err)
	return r
}

func statusCode(err error) int {
	if v5, ok := errors.AsType[*api.HTTPError](err); ok {
		return v5.StatusCode
	}
	if v8, ok := errors.AsType[*actions.HTTPError](err); ok {
		return v8.StatusCode
	}
	return 0
}

func (r *report) result(id string, err error) {
	if err == nil {
		r.add(id, "pass", "Read-only API probe succeeded; write permissions are not tested", "")
		return
	}
	message, hint := "Request failed or response was invalid", "Check service availability and retry."
	var dns *net.DNSError
	var cert *tls.CertificateVerificationError
	var unknown x509.UnknownAuthorityError
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		message, hint = "Probe canceled", "Run doctor again when ready."
	case errors.Is(err, context.DeadlineExceeded):
		message, hint = "Probe timed out", "Check connectivity and proxy settings."
	case errors.As(err, &dns):
		message, hint = "DNS lookup failed", "Check DNS and proxy settings."
	case errors.As(err, &cert), errors.As(err, &unknown):
		message, hint = "TLS certificate verification failed", "Check system time, trust store and proxy certificates."
	case statusCode(err) == 401:
		message, hint = "HTTP 401: credentials were rejected", "Run ag-cli auth login --force to authorize again."
	case statusCode(err) == 403:
		message, hint = "HTTP 403: access denied; the specific cause is not established", "Check token permissions, repository role and organization policy."
	case statusCode(err) == 404:
		message, hint = "HTTP 404: resource missing or not visible", "Check the repository name, access and feature availability."
	case statusCode(err) == 429:
		message, hint = "HTTP 429: rate limited", "Wait before retrying."
	case statusCode(err) != 0:
		message = fmt.Sprintf("HTTP %d from API", statusCode(err))
	case errors.As(err, &netErr):
		message, hint = "Network connection failed", "Check connectivity and proxy settings."
	}
	r.add(id, "fail", message, hint)
}
