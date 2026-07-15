package repo

import (
	"fmt"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
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

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}
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

func parseRepositoryName(value, defaultOwner string) (string, string, error) {
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

func repositoryListName(repo api.Repository) string {
	if repo.FullName != "" {
		return repo.FullName
	}
	return fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
}

func repositoryVisibility(repo api.Repository) string {
	if repo.Private {
		return "private"
	}
	if repo.Internal {
		return "internal"
	}
	return "public"
}

func repositoryParentName(repo api.Repository) string {
	if !repo.Fork {
		return ""
	}
	return strings.TrimSpace(repo.ParentFullName)
}

func formatRepositoryTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05 -07:00")
}

func newCmdRepoView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/]<repo>",
		Short: "View a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if opts.web {
				u := browser.BuildRepoURL(owner, repo)
				fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", u)
				if f.BrowserOpener != nil {
					if err := f.BrowserOpener(u); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "failed to open browser: %v\n", err)
					}
				}
				return nil
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			var repository api.Repository
			path := fmt.Sprintf("/repos/%s/%s", owner, repo)
			if err := client.Get(path, &repository); err != nil {
				return err
			}

			fmt.Printf("Name: %s\n", repository.FullName)
			fmt.Printf("Description: %s\n", repository.Description)
			fmt.Printf("URL: %s\n", repository.HTMLURL)
			if parent := repositoryParentName(repository); parent != "" {
				fmt.Printf("Forked from: %s\n", parent)
			}
			fmt.Printf("Default Branch: %s\n", repository.DefaultBranch)
			fmt.Printf("Visibility: %s\n", repositoryVisibility(repository))
			if repository.Language != "" {
				fmt.Printf("Language: %s\n", repository.Language)
			}
			license := strings.TrimSpace(repository.License)
			if license != "" && !strings.EqualFold(license, "NOASSERTION") {
				fmt.Printf("License: %s\n", license)
			}
			fmt.Printf("Stars: %d Forks: %d Watches: %d\n", repository.StarsCount, repository.ForksCount, repository.WatchersCount)
			fmt.Printf("Open Issues: %d\n", repository.OpenIssuesCount)
			if updatedAt := formatRepositoryTime(repository.UpdatedAt); updatedAt != "" {
				fmt.Printf("Updated: %s\n", updatedAt)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open a repository in the browser")

	return cmd
}
