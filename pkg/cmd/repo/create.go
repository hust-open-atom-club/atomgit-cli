package repo

import (
	"fmt"
	"strings"

<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
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
		Use:   "create [<name>]",
		Short: "Create a new repository",
		Long: `Create a new repository on AtomGit.

To create a repository, use 'ag repo create' with the repository name.
Pass --public to make the repository public, or --private for private.
If neither is specified, it defaults to private.

Pass --clone to clone the repository locally after creation.`,
		Example: `  # Create a new private repository
  ag repo create my-project

  # Create a public repository and clone it
  ag repo create my-project --public --clone

  # Create a repository in an organization
  ag repo create my-org/my-project --public`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("repository name required")
			}
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

func runCreate(f *cmdutil.Factory, opts *CreateOptions) error {
	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	client := api.NewClient(token)

	// Parse owner/repo
	var owner, repoName string
	parts := strings.Split(opts.Name, "/")
	if len(parts) == 2 {
		owner, repoName = parts[0], parts[1]
	} else if len(parts) == 1 {
		// Use current user as owner
		user, err := f.Config.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		owner, repoName = user, opts.Name
	} else {
		return fmt.Errorf("invalid repository name format: %s", opts.Name)
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
	currentUser, _ := f.Config.GetUser()
	if owner != currentUser {
		path = fmt.Sprintf("/orgs/%s/repos", owner)
	} else {
		path = "/user/repos"
	}

	if err := client.Post(path, body, &result); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	fmt.Printf("✓ Created repository %s/%s\n", result.Owner.Login, result.Name)
	fmt.Printf("  URL: %s\n", result.HTMLURL)

	// Clone if requested
	if opts.Clone {
		cloneURL := result.HTMLURL
		if cloneURL == "" {
			cloneURL = fmt.Sprintf("https://atomgit.com/%s/%s.git", result.Owner.Login, result.Name)
		}
		fmt.Printf("\nTo clone this repository, run:\n")
		fmt.Printf("  git clone %s\n", cloneURL)
	}

	return nil
}
