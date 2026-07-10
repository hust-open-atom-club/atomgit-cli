package repo

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type ForkOptions struct {
	Name        string
	Description string
	Private     bool
	Public      bool
	Clone       bool
}

func newCmdRepoFork(f *cmdutil.Factory) *cobra.Command {
	opts := &ForkOptions{}

	cmd := &cobra.Command{
		Use:   "fork [<owner>/]<repo>",
		Short: "Fork a repository",
		Long: `Fork a repository on AtomGit.

Creates a fork of the specified repository under your account or an organization.

By default, the fork will have the same visibility as the original repository.
Use --public or --private to override this.`,
		Example: `  # Fork a repository to your account
  ag repo fork shinwell_hu/my-project

  # Fork and rename
  ag repo fork shinwell_hu/my-project --name my-fork

  # Fork as public repository
  ag repo fork shinwell_hu/my-project --public

  # Fork and clone locally
  ag repo fork shinwell_hu/my-project --clone`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFork(f, opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Name for the forked repository")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description for the forked repository")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the forked repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the forked repository private")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the forked repository")

	return cmd
}

func runFork(f *cmdutil.Factory, opts *ForkOptions, repoArg string) error {
	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	client := api.NewClient(token)

	// Parse owner/repo
	var owner, repoName string
	parts := strings.Split(repoArg, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s (expected owner/repo)", repoArg)
	}
	owner, repoName = parts[0], parts[1]

	// Get current user
	currentUser, err := f.Config.GetUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Build request body
	body := map[string]interface{}{}

	if opts.Name != "" {
		body["name"] = opts.Name
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}

	// Determine visibility
	if opts.Public {
		body["private"] = false
	} else if opts.Private {
		body["private"] = true
	}

	// Fork the repository
	var result api.Repository
	path := fmt.Sprintf("/repos/%s/%s/forks", owner, repoName)
	if err := client.Post(path, body, &result); err != nil {
		return fmt.Errorf("failed to fork repository: %w", err)
	}

	forkName := result.Name
	if forkName == "" {
		forkName = opts.Name
		if forkName == "" {
			forkName = repoName
		}
	}

	fmt.Printf("✓ Forked %s/%s to %s/%s\n", owner, repoName, currentUser, forkName)
	if result.HTMLURL != "" {
		fmt.Printf("  URL: %s\n", result.HTMLURL)
	}

	// Clone if requested
	if opts.Clone {
		cloneURL := fmt.Sprintf("https://atomgit.com/%s/%s.git", currentUser, forkName)
		fmt.Printf("\nTo clone this repository, run:\n")
		fmt.Printf("  git clone %s\n", cloneURL)
	}

	return nil
}
