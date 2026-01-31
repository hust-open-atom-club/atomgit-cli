package repo

import (
	"fmt"
	"strings"

	"github.com/shinwell/ag-cli/internal/api"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
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
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
			}

			client := api.NewClient(token)

			var repos []api.Repository
			path := "/user/repos"
			if err := client.Get(path, &repos); err != nil {
				return err
			}

			for _, repo := range repos {
				fmt.Printf("%s/%s\n", repo.Owner.Login, repo.Name)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of repositories to list")

	return cmd
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
