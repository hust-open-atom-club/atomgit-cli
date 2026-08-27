package repo

import (
	"fmt"
	"io"
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
	cmd.AddCommand(newCmdRepoForkList(f))

	return cmd
}

func newCmdRepoForkList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
	}
	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List forks of a repository",
		Long:  "List existing forks of a repository.\n\nThis is read-only and is separate from `ag repo fork`, which creates a new fork.\nThe repository can be supplied explicitly or inferred from the current Git repository.",
		Example: `  ag repo fork list owner/repo
  ag repo fork list owner/repo --limit 100 --json
  ag repo fork list`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			forks, err := api.ListRepositoryForks(client, repository.Owner, repository.Name, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list forks for %s: %w", repository, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), repositoriesJSON(forks))
			}

			for _, fork := range forks {
				fmt.Fprintln(cmd.OutOrStdout(), forkListLine(fork))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of forks to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output forks as JSON")
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func forkListLine(fork api.Repository) string {
	name := strings.TrimSpace(repositoryListName(fork))
	if name == "" {
		name = "-"
	}
	owner := strings.TrimSpace(repositoryOwner(fork))
	visibility := repositoryVisibility(fork)
	defaultBranch := strings.TrimSpace(fork.DefaultBranch)
	urlValue := strings.TrimSpace(fork.HTMLURL)
	if urlValue == "" {
		urlValue = strings.TrimSpace(fork.AlternateHTMLURL)
	}

	line := fmt.Sprintf("%s [%s]", name, visibility)
	if owner != "" {
		line += fmt.Sprintf(" owner=%s", owner)
	}
	if defaultBranch != "" {
		line += fmt.Sprintf(" default=%s", defaultBranch)
	}
	if urlValue != "" {
		line += " " + urlValue
	}
	return line
}

func runFork(out io.Writer, f *cmdutil.Factory, opts *ForkOptions, repoArg string) error {
	if opts.Public && opts.Private {
		return fmt.Errorf("--public and --private are mutually exclusive")
	}
	repository, err := cmdutil.ParseRepository(repoArg)
	if err != nil {
		return err
	}
	owner, repoName := repository.Owner, repository.Name

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}

	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

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
