package repo

import (
	"fmt"
	"strings"

	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	Limit int
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
		Limit: 30,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
			}

			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			client := api.NewClient(token)
			repos, err := listRepos(client, opts.Limit)
			if err != nil {
				return err
			}

			for _, repo := range repos {
				fmt.Println(repositoryListName(repo))
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of repositories to list")

	return cmd
}

func listRepos(client *api.Client, limit int) ([]api.Repository, error) {
	const maxPerPage = 100

	var repos []api.Repository
	for page := 1; len(repos) < limit; page++ {
		var pageRepos []api.Repository
		path := fmt.Sprintf("/user/repos?page=%d&per_page=%d", page, maxPerPage)
		if err := client.Get(path, &pageRepos); err != nil {
			return nil, err
		}
		if len(pageRepos) == 0 {
			break
		}

		repos = append(repos, pageRepos...)
		if len(pageRepos) < maxPerPage {
			break
		}
	}

	if len(repos) > limit {
		repos = repos[:limit]
	}
	return repos, nil
}

func repositoryListName(repo api.Repository) string {
	if repo.FullName != "" {
		return repo.FullName
	}
	return fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
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
