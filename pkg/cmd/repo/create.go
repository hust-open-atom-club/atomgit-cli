package repo

import (
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	Name        string
	Description string
	Private     bool
	Public      bool
	Clone       bool
}

type cloneRepositoryFunc func(io.Reader, io.Writer, io.Writer, string, *CloneOptions, *cloneCredentials) error

func newCmdRepoCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "create <repo> | <owner>/<repo>",
		Short: "Create a new repository",
		Long: `Create a new repository on AtomGit.

Use repo to create the repository under your current account, or owner/repo to
create it under an explicitly specified user or organization namespace.
Pass --public to make the repository public, or --private for private.
If neither is specified, it defaults to private.

Pass --clone to clone the repository locally after creation.`,
		Example: `  # Create a new private repository under your account
  ag-cli repo create my-project

  # Create a public repository and clone it
  ag-cli repo create my-project --public --clone

  # Create a repository in an organization
  ag-cli repo create my-org/my-project --public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			return runCreate(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the repository")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the repository private")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the repository after creation")

	return cmd
}

func createdRepositoryURL(host, owner, repo string) string {
	return cmdutil.ResolveWebURL("", host, owner, repo)
}

func createdRepositoryURLFromResponse(f *cmdutil.Factory, result api.Repository, owner, repo string) (string, bool) {
	for _, candidate := range []string{result.HTMLURL, result.AlternateHTMLURL} {
		candidate = strings.TrimSuffix(strings.TrimSpace(candidate), "/")
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			continue
		}
		origin := (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
		if _, err := cmdutil.NormalizeBaseURL(origin); err != nil {
			continue
		}
		if !createdRepositoryURLHostAllowed(f, parsed) {
			continue
		}
		if !strings.EqualFold(parsed.Path, "/"+owner+"/"+repo) {
			continue
		}
		return candidate, true
	}
	return "", false
}

func createdRepositoryURLHostAllowed(f *cmdutil.Factory, repositoryURL *url.URL) bool {
	if repositoryURL == nil {
		return false
	}
	selectedOrigin := api.BaseURL
	if f != nil && strings.TrimSpace(f.BaseURL) != "" {
		selectedOrigin = f.BaseURL
	}
	selectedOrigin, err := cmdutil.NormalizeBaseURL(selectedOrigin)
	if err != nil {
		return false
	}
	selectedURL, err := url.Parse(selectedOrigin)
	if err != nil {
		return false
	}
	if strings.EqualFold(repositoryURL.Scheme, selectedURL.Scheme) && strings.EqualFold(repositoryURL.Host, selectedURL.Host) {
		return true
	}
	if !isPublicAPIHost(selectedURL) || repositoryURL.Port() != "" {
		return repositoryURL.Scheme == "https" && repositoryURL.Port() == selectedURL.Port() && relatedDeploymentHost(selectedURL.Hostname(), repositoryURL.Hostname())
	}
	return repositoryURL.Scheme == "https" && cmdutil.IsAtomGitHost(repositoryURL.Hostname())
}

func relatedDeploymentHost(apiHost, repositoryHost string) bool {
	apiHost = strings.ToLower(strings.TrimSuffix(apiHost, "."))
	repositoryHost = strings.ToLower(strings.TrimSuffix(repositoryHost, "."))
	if apiHost == "" || repositoryHost == "" {
		return false
	}
	candidates := []string{apiHost}
	if after, ok := strings.CutPrefix(apiHost, "api."); ok {
		candidates = append(candidates, after)
	}
	labels := strings.Split(apiHost, ".")
	if len(labels) > 1 {
		first := labels[0]
		for _, webFirst := range []string{strings.TrimSuffix(first, "-api"), strings.TrimPrefix(first, "api-")} {
			if webFirst != "" && webFirst != first {
				candidate := append([]string{webFirst}, labels[1:]...)
				candidates = append(candidates, strings.Join(candidate, "."))
			}
		}
	}
	return slices.Contains(candidates, repositoryHost)
}

func isPublicAPIHost(selectedURL *url.URL) bool {
	if selectedURL == nil || selectedURL.Scheme != "https" || selectedURL.Port() != "" {
		return false
	}
	switch strings.ToLower(selectedURL.Hostname()) {
	case "api.atomgit.com", "api.gitcode.com":
		return true
	default:
		return false
	}
}

func canUseDefaultRepositoryURLFallback(f *cmdutil.Factory) bool {
	if f == nil || strings.TrimSpace(f.BaseURL) == "" {
		return true
	}
	origin, err := cmdutil.NormalizeBaseURL(f.BaseURL)
	defaultOrigin, defaultErr := cmdutil.NormalizeBaseURL(api.BaseURL)
	return err == nil && defaultErr == nil && strings.EqualFold(origin, defaultOrigin)
}

func createdRepositoryCloneCredentials(f *cmdutil.Factory, cloneURL string) *cloneCredentials {
	if f == nil || f.Config == nil {
		return nil
	}
	if canUseDefaultRepositoryURLFallback(f) {
		return resolveCloneAuth(f.Config, cloneURL)
	}

	// A custom API origin does not establish trust in an unrelated Web or Git
	// host. Forward the command credential only to the same HTTPS host or a host
	// related by one of the conservative API/Web naming rules above.
	origin, err := cmdutil.NormalizeBaseURL(f.BaseURL)
	if err != nil {
		return nil
	}
	apiURL, err := url.Parse(origin)
	if err != nil {
		return nil
	}
	cloneTarget, err := url.Parse(cloneURL)
	if err != nil || cloneTarget.Scheme != "https" || cloneTarget.User != nil {
		return nil
	}
	if !strings.EqualFold(cloneTarget.Host, apiURL.Host) && (cloneTarget.Port() != apiURL.Port() || !relatedDeploymentHost(apiURL.Hostname(), cloneTarget.Hostname())) {
		return nil
	}
	token, err := f.Config.GetToken()
	if err != nil || strings.TrimSpace(token) == "" {
		return nil
	}
	username, err := f.Config.GetUser()
	if err != nil || strings.TrimSpace(username) == "" {
		return nil
	}
	return &cloneCredentials{host: cloneTarget.Host, username: username, token: token}
}

func runCreate(in io.Reader, out, errOut io.Writer, f *cmdutil.Factory, opts *CreateOptions) error {
	return runCreateWithClone(in, out, errOut, f, opts, runClone)
}

func runCreateWithClone(in io.Reader, out, errOut io.Writer, f *cmdutil.Factory, opts *CreateOptions, clone cloneRepositoryFunc) error {
	// Validate conflicting visibility flags before any config/auth/client work so
	// the error is reported even when unauthenticated (e.g. missing token would
	// otherwise surface first as "not authenticated").
	if opts.Public && opts.Private {
		return fmt.Errorf("--public and --private are mutually exclusive")
	}

	var owner, repoName string
	var err error
	if strings.Contains(opts.Name, "/") {
		owner, repoName, err = parseRepositoryName(opts.Name, "")
	} else {
		_, repoName, err = parseRepositoryName(opts.Name, "current-user")
	}
	if err != nil {
		return err
	}
	currentUser, err := f.Config.GetUser()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	if owner == "" {
		owner = currentUser
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}

	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	// Determine visibility
	visibility := "private"
	if opts.Public {
		visibility = "public"
	}

	// Build request body
	body := map[string]any{
		"name":        repoName,
		"description": opts.Description,
		"private":     visibility == "private",
	}

	var result api.Repository
	var path string

	// If owner is different from current user, create under organization
	if owner != currentUser {
		path = fmt.Sprintf("/orgs/%s/repos", owner)
	} else {
		path = "/user/repos"
	}

	if err := client.Post(path, body, &result); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	repoURL, responseURL := createdRepositoryURLFromResponse(f, result, owner, repoName)
	if !responseURL && canUseDefaultRepositoryURLFallback(f) {
		repoURL = createdRepositoryURL(f.Config.GetHost(), owner, repoName)
	}
	fmt.Fprintf(out, "✓ Created repository %s/%s\n", owner, repoName)
	if repoURL == "" {
		const unavailable = "the selected API service did not return a usable repository URL"
		if opts.Clone {
			return fmt.Errorf("repository was created, but clone was skipped: %s", unavailable)
		}
		fmt.Fprintf(errOut, "warning: %s\n", unavailable)
		return nil
	}
	fmt.Fprintf(out, "  URL: %s\n", repoURL)

	// Clone if requested
	if opts.Clone {
		cloneURL := strings.TrimSuffix(repoURL, ".git") + ".git"
		cloneOpts := &CloneOptions{Directory: repoName}
		if err := clone(in, out, errOut, cloneURL, cloneOpts, createdRepositoryCloneCredentials(f, cloneURL)); err != nil {
			return fmt.Errorf("failed to clone newly created repository: %w", err)
		}
	}

	return nil
}
