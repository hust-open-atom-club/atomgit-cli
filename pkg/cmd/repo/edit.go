package repo

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type EditOptions struct {
	Name          string
	Description   string
	DefaultBranch string
	Visibility    string
	Public        bool
	Private       bool
	Yes           bool
}

func newCmdRepoEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &EditOptions{}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>]",
		Short: "Edit repository settings",
		Long: `Edit supported repository metadata and visibility on AtomGit.

Only flags explicitly provided are sent to AtomGit; omitted settings remain
unchanged. Name and visibility updates require confirmation unless --yes is
used. --visibility, --public, and --private are mutually exclusive.

This command does not change the repository path, owner, homepage, LFS state,
module switches, merge policies, or other unsupported GitHub CLI settings.`,
		Example: `  # Update the current Git repository
  ag repo edit --description "New description"

  # Update an explicitly selected repository
  ag repo edit owner/repo --description "New description"

  # Clear a description without changing other settings
  ag repo edit owner/repo --description ""

  # Update several settings
  ag repo edit owner/repo --name "New name" --default-branch main --visibility private

  # Skip confirmation for a visibility update
  ag repo edit owner/repo --public --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request, consequential, err := buildRepoEditRequest(cmd, opts)
			if err != nil {
				return err
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repoName := repository.Owner, repository.Name

			if consequential && !opts.Yes {
				confirmed, err := confirmRepoEdit(cmd.InOrStdin(), cmd.OutOrStdout(), owner, repoName, request)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Repository update cancelled.")
					return nil
				}
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("failed to edit repository %s/%s: not authenticated: %w", owner, repoName, err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return fmt.Errorf("failed to edit repository %s/%s: %w", owner, repoName, err)
			}

			path := fmt.Sprintf("/repos/%s/%s", owner, repoName)
			var updated api.Repository
			if err := client.Patch(path, request, &updated); err != nil {
				return fmt.Errorf("failed to edit repository %s/%s: %w", owner, repoName, err)
			}

			if !usableRepoEditResult(updated) {
				if err := client.Get(path, &updated); err != nil {
					return fmt.Errorf("repository %s/%s was updated, but failed to read the updated repository: %w", owner, repoName, err)
				}
				if !usableRepoEditResult(updated) {
					return fmt.Errorf("repository %s/%s was updated, but AtomGit did not return usable updated repository details", owner, repoName)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated repository %s\n", repoEditDisplayName(updated, owner, repoName))
			fmt.Fprintf(cmd.OutOrStdout(), "  URL: %s\n", repoEditBrowserURL(updated, f.Config.GetHost(), owner, repoName))
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "New repository name (does not change the repository path)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "New repository description")
	cmd.Flags().StringVar(&opts.DefaultBranch, "default-branch", "", "New default branch")
	cmd.Flags().StringVar(&opts.Visibility, "visibility", "", "New visibility: public or private")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the repository private")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation for name or visibility changes")
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func buildRepoEditRequest(cmd *cobra.Command, opts *EditOptions) (map[string]interface{}, bool, error) {
	request := make(map[string]interface{})
	consequential := false

	if cmd.Flags().Changed("name") {
		name := strings.TrimSpace(opts.Name)
		if err := validateRepositoryEditName(name); err != nil {
			return nil, false, err
		}
		request["name"] = name
		consequential = true
	}
	if cmd.Flags().Changed("description") {
		request["description"] = opts.Description
	}
	if cmd.Flags().Changed("default-branch") {
		defaultBranch := strings.TrimSpace(opts.DefaultBranch)
		if defaultBranch == "" {
			return nil, false, fmt.Errorf("default branch cannot be empty")
		}
		request["default_branch"] = defaultBranch
	}

	visibilitySelectors := 0
	if cmd.Flags().Changed("visibility") {
		visibilitySelectors++
	}
	if cmd.Flags().Changed("public") {
		visibilitySelectors++
	}
	if cmd.Flags().Changed("private") {
		visibilitySelectors++
	}
	if visibilitySelectors > 1 {
		return nil, false, fmt.Errorf("--visibility, --public, and --private are mutually exclusive")
	}
	if cmd.Flags().Changed("public") && !opts.Public {
		return nil, false, fmt.Errorf("--public=false is not supported; use --private or --visibility private")
	}
	if cmd.Flags().Changed("private") && !opts.Private {
		return nil, false, fmt.Errorf("--private=false is not supported; use --public or --visibility public")
	}

	switch {
	case cmd.Flags().Changed("visibility"):
		switch opts.Visibility {
		case "public":
			request["private"] = false
		case "private":
			request["private"] = true
		case "internal":
			return nil, false, fmt.Errorf("visibility %q is not supported; AtomGit repository editing supports only public and private", opts.Visibility)
		default:
			return nil, false, fmt.Errorf("invalid visibility %q (expected public or private)", opts.Visibility)
		}
		consequential = true
	case cmd.Flags().Changed("public"):
		request["private"] = false
		consequential = true
	case cmd.Flags().Changed("private"):
		request["private"] = true
		consequential = true
	}

	if len(request) == 0 {
		return nil, false, fmt.Errorf("at least one of --name, --description, --default-branch, --visibility, --public, or --private must be provided")
	}
	return request, consequential, nil
}

func validateRepositoryEditName(name string) error {
	if name == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid repository name %q", name)
	}
	for _, r := range name {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return fmt.Errorf("invalid repository name %q: names cannot contain slashes, backslashes, or control characters", name)
		}
	}
	return nil
}

func confirmRepoEdit(in io.Reader, out io.Writer, owner, repo string, request map[string]interface{}) (bool, error) {
	var changes []string
	if name, ok := request["name"].(string); ok {
		changes = append(changes, fmt.Sprintf("name to %q", name))
	}
	if private, ok := request["private"].(bool); ok {
		visibility := "public"
		if private {
			visibility = "private"
		}
		changes = append(changes, "visibility to "+visibility)
	}
	fmt.Fprintf(out, "Update %s/%s %s? [y/N] ", owner, repo, strings.Join(changes, " and "))

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

func usableRepoEditResult(repository api.Repository) bool {
	hasIdentity := strings.TrimSpace(repository.Name) != "" ||
		strings.TrimSpace(repository.FullName) != "" ||
		strings.TrimSpace(repository.Path) != ""
	hasBrowserLocation := strings.TrimSpace(repository.HTMLURL) != "" ||
		strings.TrimSpace(repository.AlternateHTMLURL) != "" ||
		strings.TrimSpace(repository.Path) != ""
	return hasIdentity && hasBrowserLocation
}

func repoEditDisplayName(repository api.Repository, owner, fallbackRepo string) string {
	if name := strings.TrimSpace(repository.Name); name != "" {
		return fmt.Sprintf("%s/%s", owner, name)
	}
	if name := strings.TrimSpace(repository.FullName); name != "" {
		return name
	}
	if path := strings.TrimSpace(repository.Path); path != "" {
		return fmt.Sprintf("%s/%s", owner, path)
	}
	return fmt.Sprintf("%s/%s", owner, fallbackRepo)
}

func repoEditBrowserURL(repository api.Repository, host, owner, fallbackRepo string) string {
	if url := strings.TrimSpace(repository.HTMLURL); url != "" {
		return url
	}
	if url := strings.TrimSpace(repository.AlternateHTMLURL); url != "" {
		return url
	}

	repoPath := strings.TrimSpace(repository.Path)
	if repoPath == "" {
		repoPath = fallbackRepo
	}
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		host = "atomgit.com"
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return fmt.Sprintf("%s/%s/%s", host, owner, repoPath)
}
