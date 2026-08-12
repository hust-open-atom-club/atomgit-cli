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
	JSON  bool
}

type repositoryJSON struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	Visibility    string `json:"visibility"`
	DefaultBranch string `json:"defaultBranch"`
	Language      string `json:"language"`
	License       string `json:"license"`
	Fork          bool   `json:"fork"`
	Parent        string `json:"parent"`
	UpdatedAt     string `json:"updatedAt"`
	Stars         int    `json:"stars"`
	Forks         int    `json:"forks"`
	Watchers      int    `json:"watchers"`
	OpenIssues    int    `json:"openIssues"`
	Owner         string `json:"owner"`
}

func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		Long:  "Create, clone, edit, fork, sync, view, and manage repository collaborators and webhooks.\n\nFor repository-scoped commands, OWNER/REPO may be omitted and inferred from the current Git repository.",
	}

	cmd.AddCommand(newCmdRepoList(f))
	cmd.AddCommand(newCmdRepoView(f))
	cmd.AddCommand(newCmdRepoCreate(f))
	cmd.AddCommand(newCmdRepoEdit(f))
	cmd.AddCommand(newCmdRepoClone(f))
	cmd.AddCommand(newCmdRepoDelete(f))
	cmd.AddCommand(newCmdRepoFork(f))
	cmd.AddCommand(newCmdRepoSync(f))
	cmd.AddCommand(newCmdRepoCollaborator(f))
	cmd.AddCommand(newCmdRepoWebhook(f))

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
				return cmdutil.AuthenticationError(err)
			}

			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			repos, err := listRepos(client, opts.Limit)
			if err != nil {
				return err
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), repositoriesJSON(repos))
			}

			out := cmd.OutOrStdout()
			for _, repo := range repos {
				fmt.Fprintln(out, repositoryListName(repo))
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of repositories to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output repositories as JSON")

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

func repositoriesJSON(repositories []api.Repository) []repositoryJSON {
	result := make([]repositoryJSON, len(repositories))
	for index, repository := range repositories {
		result[index] = newRepositoryJSON(repository)
	}
	return result
}

func newRepositoryJSON(repository api.Repository) repositoryJSON {
	url := strings.TrimSpace(repository.HTMLURL)
	if url == "" {
		url = strings.TrimSpace(repository.AlternateHTMLURL)
	}
	return repositoryJSON{
		ID:            repository.ID,
		Name:          repository.Name,
		FullName:      repositoryListName(repository),
		Description:   repository.Description,
		URL:           url,
		Visibility:    repositoryVisibility(repository),
		DefaultBranch: repository.DefaultBranch,
		Language:      repository.Language,
		License:       repository.License,
		Fork:          repository.Fork,
		Parent:        repositoryParentName(repository),
		UpdatedAt:     repository.UpdatedAt,
		Stars:         repository.StarsCount,
		Forks:         repository.ForksCount,
		Watchers:      repository.WatchersCount,
		OpenIssues:    repository.OpenIssuesCount,
		Owner:         repository.Owner.Login,
	}
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
		web  bool
		json bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>]",
		Short: "View a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextRepository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := contextRepository.Owner, contextRepository.Name

			if opts.web {
				u := browser.BuildRepoURL(owner, repo)
				fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", u)
				if f.BrowserOpener != nil {
					if err := f.BrowserOpener(u); err != nil {
						return fmt.Errorf("failed to open browser: %w", err)
					}
				}
				return nil
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return err
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			var repository api.Repository
			path := fmt.Sprintf("/repos/%s/%s", owner, repo)
			if err := client.Get(path, &repository); err != nil {
				return err
			}
			if opts.json {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newRepositoryJSON(repository))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name: %s\n", repository.FullName)
			fmt.Fprintf(out, "Description: %s\n", repository.Description)
			fmt.Fprintf(out, "URL: %s\n", repository.HTMLURL)
			if parent := repositoryParentName(repository); parent != "" {
				fmt.Fprintf(out, "Forked from: %s\n", parent)
			}
			fmt.Fprintf(out, "Default Branch: %s\n", repository.DefaultBranch)
			fmt.Fprintf(out, "Visibility: %s\n", repositoryVisibility(repository))
			if repository.Language != "" {
				fmt.Fprintf(out, "Language: %s\n", repository.Language)
			}
			license := strings.TrimSpace(repository.License)
			if license != "" && !strings.EqualFold(license, "NOASSERTION") {
				fmt.Fprintf(out, "License: %s\n", license)
			}
			fmt.Fprintf(out, "Stars: %d Forks: %d Watches: %d\n", repository.StarsCount, repository.ForksCount, repository.WatchersCount)
			fmt.Fprintf(out, "Open Issues: %d\n", repository.OpenIssuesCount)
			if updatedAt := formatRepositoryTime(repository.UpdatedAt); updatedAt != "" {
				fmt.Fprintf(out, "Updated: %s\n", updatedAt)
			}

			return nil
		},
	}
	cmdutil.AddRepositoryContextHelp(cmd)

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open a repository in the browser")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output repository as JSON")
	cmd.MarkFlagsMutuallyExclusive("web", "json")

	return cmd
}
