package repo

import (
	"fmt"
	"strings"

	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
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

			return runCreate(f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the repository")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the repository private")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the repository after creation")

	return cmd
}

func parseCreateRepositoryName(value, defaultOwner string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) > 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected repo or owner/repo)", value)
	}

	owner := strings.TrimSpace(defaultOwner)
	repo := strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		owner = strings.TrimSpace(parts[0])
		repo = strings.TrimSpace(parts[1])
	}
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: %s (expected repo or owner/repo)", value)
	}
	return owner, repo, nil
}

func createdRepositoryURL(result api.Repository, owner, repo string) string {
	if url := strings.TrimSpace(result.HTMLURL); url != "" {
		return url
	}
	return fmt.Sprintf("https://atomgit.com/%s/%s", owner, repo)
}

func runCreate(f *cmdutil.Factory, opts *CreateOptions) error {
	currentUser, err := f.Config.GetUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	owner, repoName, err := parseCreateRepositoryName(opts.Name, currentUser)
	if err != nil {
		return err
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	client := api.NewClient(token)

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

	repoURL := createdRepositoryURL(result, owner, repoName)
	fmt.Printf("✓ Created repository %s/%s\n", owner, repoName)
	fmt.Printf("  URL: %s\n", repoURL)

	// Clone if requested
	if opts.Clone {
		cloneURL := strings.TrimSuffix(repoURL, ".git") + ".git"
		fmt.Printf("\nTo clone this repository, run:\n")
		fmt.Printf("  git clone %s\n", cloneURL)
	}

	return nil
}
