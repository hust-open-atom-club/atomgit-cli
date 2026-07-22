package repo

import (
	"fmt"
	"io"
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
  ag repo create my-project

  # Create a public repository and clone it
  ag repo create my-project --public --clone

  # Create a repository in an organization
  ag repo create my-org/my-project --public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			return runCreate(cmd.OutOrStdout(), f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the repository")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the repository private")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the repository after creation")

	return cmd
}

func createdRepositoryURL(result api.Repository, host, owner, repo string) string {
	return cmdutil.ResolveWebURL(result.HTMLURL, host, owner, repo)
}

func runCreate(out io.Writer, f *cmdutil.Factory, opts *CreateOptions) error {
	currentUser, err := f.Config.GetUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	owner, repoName, err := parseRepositoryName(opts.Name, currentUser)
	if err != nil {
		return err
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	client, err := newAPIClient(f, token)
	if err != nil {
		return err
	}

	// Determine visibility
	visibility := "private"
	if opts.Public {
		visibility = "public"
	}

	// Build request body
	body := map[string]interface{}{
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

	repoURL := createdRepositoryURL(result, f.Config.GetHost(), owner, repoName)
	fmt.Fprintf(out, "✓ Created repository %s/%s\n", owner, repoName)
	fmt.Fprintf(out, "  URL: %s\n", repoURL)

	// Clone if requested
	if opts.Clone {
		cloneURL := strings.TrimSuffix(repoURL, ".git") + ".git"
		fmt.Fprintf(out, "\nTo clone this repository, run:\n")
		fmt.Fprintf(out, "  git clone %s\n", cloneURL)
	}

	return nil
}
