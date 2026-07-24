package repo

import (
	"fmt"
	"io"

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
		Use:   "fork [<owner>/<repo>]",
		Short: "Fork a repository",
		Long: `Fork a repository on AtomGit.

Creates a fork of the specified repository under your account or an organization.

By default, the fork will have the same visibility as the original repository.
Use --private or --public to override.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			return runFork(cmd.OutOrStdout(), f, opts, repository.String())
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Name for the forked repository")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description for the forked repository")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the forked repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the forked repository private")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the forked repository")
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func runFork(out io.Writer, f *cmdutil.Factory, opts *ForkOptions, repoArg string) error {
	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}

	client, err := newAPIClient(f, token)
	if err != nil {
		return err
	}

	repository, err := cmdutil.ParseRepository(repoArg)
	if err != nil {
		return err
	}
	owner, repoName := repository.Owner, repository.Name

	currentUser, err := f.Config.GetUser()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}

	body := map[string]interface{}{}
	if opts.Name != "" {
		body["name"] = opts.Name
	}
	if opts.Public {
		body["private"] = false
	} else if opts.Private {
		body["private"] = true
	}

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

	if opts.Description != "" {
		if err := setAndVerifyForkDescription(client, currentUser, forkName, opts.Description); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "✓ Forked %s/%s to %s/%s\n", owner, repoName, currentUser, forkName)
	if result.HTMLURL != "" {
		fmt.Fprintf(out, "  URL: %s\n", result.HTMLURL)
	}

	// Clone if requested
	if opts.Clone {
		cloneURL := fmt.Sprintf("https://atomgit.com/%s/%s.git", currentUser, forkName)
		fmt.Fprintf(out, "\nTo clone this repository, run:\n")
		fmt.Fprintf(out, "  git clone %s\n", cloneURL)
	}

	return nil
}

func setAndVerifyForkDescription(client *api.Client, owner, repo, description string) error {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	body := map[string]interface{}{
		"name":        repo,
		"description": description,
	}
	var updated api.Repository
	if err := client.Patch(path, body, &updated); err != nil {
		return fmt.Errorf("failed to update fork description: %w", err)
	}

	var verified api.Repository
	if err := client.Get(path, &verified); err != nil {
		return fmt.Errorf("failed to verify fork description: %w", err)
	}
	if verified.Description != description {
		return fmt.Errorf("fork description mismatch: requested %q, got %q", description, verified.Description)
	}
	return nil
}
