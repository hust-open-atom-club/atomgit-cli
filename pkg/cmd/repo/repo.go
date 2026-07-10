package repo

import (
	"fmt"
	"strings"

	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	Page    int
	PerPage int
}

func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		Long:  `Create, clone, fork, and view repositories.`,
	}

	cmd.AddCommand(newCmdRepoList(f))
	cmd.AddCommand(newCmdRepoView(f))
	cmd.AddCommand(newCmdRepoCreate(f))
	cmd.AddCommand(newCmdRepoClone(f))
	cmd.AddCommand(newCmdRepoDelete(f))
	cmd.AddCommand(newCmdRepoFork(f))

	return cmd
}

func newCmdRepoList(f *cmdutil.Factory) *cobra.Command {
	opts := &ListOptions{
		Page:    1,
		PerPage: 30,
	}

	cmd := &cobra.Command{
		Use:   "list [<page> <per-page>]",
		Short: "List repositories",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
			}

			if len(args) >= 1 {
				page, err := parseIntArg(args[0], "page")
				if err != nil {
					return err
				}
				opts.Page = page
			}
			if len(args) >= 2 {
				perPage, err := parseIntArg(args[1], "per-page")
				if err != nil {
					return err
				}
				opts.PerPage = perPage
			}

			client := api.NewClient(token)

			var repos []api.Repository
			path := fmt.Sprintf("/user/repos?page=%d&per_page=%d", opts.Page, opts.PerPage)
			if err := client.Get(path, &repos); err != nil {
				return err
			}

			for _, repo := range repos {
				fmt.Printf("%s/%s\n", repo.Owner.Login, repo.Name)
			}

			return nil
		},
	}

	return cmd
}

func parseIntArg(s string, name string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid %s: %q (must be a number)", name, s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid %s: %d (must be positive)", name, n)
	}
	return n, nil
}

func newCmdRepoView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view [<owner>/]<repo>",
		Short: "View a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			if len(args) == 0 {
				user, err := f.Config.GetUser()
				if err != nil {
					return err
				}
				owner = user
				repo = ""
			} else {
				// Parse owner/repo format
				parts := strings.Split(args[0], "/")
				if len(parts) != 2 {
					return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
				}
				owner, repo = parts[0], parts[1]
			}

			if repo == "" {
				return fmt.Errorf("repository name required")
			}

			var repository api.Repository
			path := fmt.Sprintf("/repos/%s/%s", owner, repo)
			if err := client.Get(path, &repository); err != nil {
				return err
			}

			fmt.Printf("Name: %s\n", repository.FullName)
			fmt.Printf("Description: %s\n", repository.Description)
			fmt.Printf("URL: %s\n", repository.HTMLURL)
			fmt.Printf("Stars: %d\n", repository.StarsCount)
			fmt.Printf("Forks: %d\n", repository.ForksCount)
			fmt.Printf("Open Issues: %d\n", repository.OpenIssuesCount)
			fmt.Printf("Default Branch: %s\n", repository.DefaultBranch)
			fmt.Printf("Private: %v\n", repository.Private)

			return nil
		},
	}

	return cmd
}
