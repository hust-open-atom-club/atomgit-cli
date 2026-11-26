package search

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	searchMaxPerPage           = 50
	repositorySearchMaxPerPage = 20
)

var (
	searchOrders          = []string{"asc", "desc"}
	userSearchSorts       = []string{"joined_at"}
	repositorySearchSorts = []string{"last_push_at", "stars_count", "forks_count"}
	issueSearchSorts      = []string{"created_at", "last_push_at"}
	issueSearchStates     = []string{"open", "closed"}
)

func setOptional(values url.Values, name, value string) {
	if value != "" {
		values.Set(name, value)
	}
}

func validateChoice(name, value string, choices []string) error {
	if value == "" {
		return nil
	}
	if slices.Contains(choices, value) {
		return nil
	}
	return fmt.Errorf("invalid %s: %q (must be one of %s)", name, value, strings.Join(choices, ", "))
}

func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "search",
		Short:        "search atomgit",
		Long:         "Search AtomGit repositories, users, issues, pull requests, and organizations.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCmdSearchUsers(f))
	cmd.AddCommand(newCmdSearchRepositories(f))
	cmd.AddCommand(newCmdSearchIssues(f))
	cmd.AddCommand(newCmdUnifiedSearch(f, "prs", "pull requests", "pullrequest"))
	cmd.AddCommand(newCmdUnifiedSearch(f, "orgs", "organizations", "group"))
	cmd.AddCommand(newCmdSearchByType(f))

	return cmd
}

type SearchIssue struct {
	api.Issue
	Repository struct {
		FullName string `json:"full_name"`
		URL      string `json:"url"`
	} `json:"repository"`
}

func newCmdSearchUsers(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
		Sort  string
		Order string
	}

	cmd := &cobra.Command{
		Use:   "users <query>",
		Short: "search users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateChoice("sort", opts.Sort, userSearchSorts); err != nil {
				return err
			}
			if err := validateChoice("order", opts.Order, searchOrders); err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			query := args[0]

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			users, err := api.GetPaginatedWithPageSize[api.User](client, opts.Limit, searchMaxPerPage, func(page, perPage int) string {
				v := url.Values{}
				v.Set("q", query)
				setOptional(v, "sort", opts.Sort)
				setOptional(v, "order", opts.Order)
				v.Set("page", fmt.Sprintf("%d", page))
				v.Set("per_page", fmt.Sprintf("%d", perPage))
				return "/search/users?" + v.Encode()
			})

			if err != nil {
				return fmt.Errorf("search users failed: %w", err)
			}

			out := cmd.OutOrStdout()
			if opts.JSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", " ")
				return enc.Encode(users)
			}

			if len(users) == 0 {
				fmt.Fprintln(out, "No users found.")
				return nil
			}

			for _, u := range users {
				fmt.Fprintf(out, "%s\t%s\t%s\n", u.Login, u.Name, u.HTMLURL)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results as JSON")
	cmd.Flags().StringVar(&opts.Sort, "sort", "", "Sort by joined_at")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Sort order: asc or desc")
	return cmd
}

func newCmdSearchRepositories(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit      int
		JSON       bool
		Sort       string
		Order      string
		Owner      string
		Fork       bool
		Language   string
		Repository repositorySearchFlags
	}

	cmd := &cobra.Command{
		Use:     "repositories <query>",
		Aliases: []string{"repos"},
		Short:   "search repositories",
		Long: `Search repositories. With --repo-type, use unified search and return its result fields.

` + modelSearchFilterNote + `

Unified search accepts created_at, updated_at, star, or fork for --sort.
Without --repo-type, use last_push_at, stars_count, or forks_count.

` + unifiedSearchSortNote,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repositorySearchRequested(cmd) {
				flags := cmd.Flags()
				if len(opts.Repository.RepoType) > 0 &&
					(flags.Changed("owner") || flags.Changed("fork") || flags.Changed("language")) {
					return fmt.Errorf("--repo-type cannot be combined with --owner, --fork, or --language")
				}
				return runUnifiedSearch(cmd, f, "repo", unifiedSearchOptions{
					SearchOptions: api.SearchOptions{Query: args[0], Limit: opts.Limit, Sort: opts.Sort, Order: opts.Order},
					JSON:          opts.JSON,
					Repository:    opts.Repository,
				})
			}
			if err := validateChoice("sort", opts.Sort, repositorySearchSorts); err != nil {
				return err
			}
			if err := validateChoice("order", opts.Order, searchOrders); err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			query := args[0]

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			repositories, err := api.GetPaginatedWithPageSize[api.Repository](client, opts.Limit, repositorySearchMaxPerPage, func(page, perPage int) string {
				v := url.Values{}
				v.Set("q", query)
				setOptional(v, "sort", opts.Sort)
				setOptional(v, "order", opts.Order)
				setOptional(v, "owner", opts.Owner)
				if opts.Fork {
					v.Set("fork", "true")
				}
				setOptional(v, "language", opts.Language)
				v.Set("page", fmt.Sprintf("%d", page))
				v.Set("per_page", fmt.Sprintf("%d", perPage))
				return "/search/repositories?" + v.Encode()
			})

			if err != nil {
				return fmt.Errorf("search repositories failed: %w", err)
			}

			out := cmd.OutOrStdout()
			if opts.JSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", " ")
				return enc.Encode(repositories)
			}

			if len(repositories) == 0 {
				fmt.Fprintln(out, "No repositories found.")
				return nil
			}

			for _, r := range repositories {
				fmt.Fprintf(out, "%s\t★%d\t%s\n", r.FullName, r.StarsCount, strings.Join(strings.Fields(r.Description), " "))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results as JSON")
	cmd.Flags().StringVar(&opts.Sort, "sort", "", "Server sort field (see command description for values)")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Sort order: asc or desc")
	cmd.Flags().StringVar(&opts.Owner, "owner", "", "Filter by repository owner path")
	cmd.Flags().BoolVar(&opts.Fork, "fork", false, "Include forked repositories")
	cmd.Flags().StringVar(&opts.Language, "language", "", "Filter by repository language")
	addUnifiedRepositoryFlags(cmd, &opts.Repository)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Search keyword."},
		Output:      "text,json",
		Effects:     "read",
		Notes: []string{
			"--repo-type uses unified search, accepts one type, and cannot be combined with --owner, --fork, or --language.",
			modelSearchFilterNote,
			"Unified search can run anonymously and outputs its own result fields. The legacy repository search requires authentication.",
			unifiedSearchSortNote,
		},
	})
	return cmd
}

func newCmdSearchIssues(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
		Sort  string
		Order string
		Repo  string
		State string
	}

	cmd := &cobra.Command{
		Use:   "issues <query>",
		Short: "search issues",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateChoice("sort", opts.Sort, issueSearchSorts); err != nil {
				return err
			}
			if err := validateChoice("order", opts.Order, searchOrders); err != nil {
				return err
			}
			if err := validateChoice("state", opts.State, issueSearchStates); err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			query := args[0]

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			issues, err := api.GetPaginatedUntilEmptyWithPageSize[SearchIssue](client, opts.Limit, searchMaxPerPage, searchIssueKey, func(page, perPage int) string {
				v := url.Values{}
				v.Set("q", query)
				setOptional(v, "sort", opts.Sort)
				setOptional(v, "order", opts.Order)
				setOptional(v, "repo", opts.Repo)
				setOptional(v, "state", opts.State)
				v.Set("page", fmt.Sprintf("%d", page))
				v.Set("per_page", fmt.Sprintf("%d", perPage))
				return "/search/issues?" + v.Encode()
			})

			if err != nil {
				return fmt.Errorf("search issues failed: %w", err)
			}

			out := cmd.OutOrStdout()
			if opts.JSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", " ")
				return enc.Encode(issues)
			}

			if len(issues) == 0 {
				fmt.Fprintln(out, "No issues found.")
				return nil
			}

			for _, i := range issues {
				fmt.Fprintf(out, "%s\t#%s [%s]\t%s\n",
					i.Repository.FullName,
					i.GetNumber(),
					i.State,
					i.Title,
				)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results as JSON")
	cmd.Flags().StringVar(&opts.Sort, "sort", "", "Sort by created_at or last_push_at")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Sort order: asc or desc")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Filter by repository path")
	cmd.Flags().StringVar(&opts.State, "state", "", "Filter by state: open or closed")
	return cmd
}

func searchIssueKey(issue SearchIssue) string {
	if issue.ID != 0 {
		return fmt.Sprintf("id:%d", issue.ID)
	}
	return issue.Repository.FullName + "\x00" + issue.GetNumber()
}
