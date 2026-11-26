package search

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	unifiedSearchSortNote = "Sort options are sent to AtomGit. Effective sorting depends on the search type and server support."
	unifiedSearchTypes    = "repo, user, group, pullrequest, issue"
	modelSearchFilterNote = "--task, --library, --param-min, and --param-max require --repo-type model."
)

var unifiedSearchSorts = []string{"created_at", "updated_at", "star", "fork"}
var modelSearchFlags = []string{"task", "library", "param-min", "param-max"}

type unifiedSearchOptions struct {
	api.SearchOptions
	JSON       bool
	Repository repositorySearchFlags
}

type repositorySearchFlags struct {
	RepoType []string
	Model    api.SearchModelFilters
}

func repositorySearchRequested(cmd *cobra.Command) bool {
	flags := cmd.Flags()
	if flags.Changed("repo-type") {
		return true
	}
	return slices.ContainsFunc(modelSearchFlags, flags.Changed)
}

func newCmdUnifiedSearch(f *cmdutil.Factory, name, label, kind string) *cobra.Command {
	var opts unifiedSearchOptions
	cmd := &cobra.Command{
		Use:   name + " <query>",
		Short: "search " + label,
		Long:  "Search AtomGit " + label + " using the unified search API. " + unifiedSearchSortNote,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[0]
			return runUnifiedSearch(cmd, f, kind, opts)
		},
	}
	addUnifiedSearchFlags(cmd, &opts)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Search keyword."},
		Output:      "text,json",
		Effects:     "read",
		Notes:       []string{"Can run anonymously.", unifiedSearchSortNote},
	})
	return cmd
}

func newCmdSearchByType(f *cmdutil.Factory) *cobra.Command {
	var opts unifiedSearchOptions
	cmd := &cobra.Command{
		Use:   "by-type <type> <query>",
		Short: "search by resource type",
		Long: "Search using the unified API. Types are case-insensitive: " + unifiedSearchTypes + ".\n\n" +
			"--repo-type applies to repo searches. " + modelSearchFilterNote + "\n\n" + unifiedSearchSortNote,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[1]
			return runUnifiedSearch(cmd, f, strings.ToLower(args[0]), opts)
		},
	}
	addUnifiedSearchFlags(cmd, &opts)
	addUnifiedRepositoryFlags(cmd, &opts.Repository)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "A search type (" + unifiedSearchTypes + ") and a keyword."},
		Output:      "text,json",
		Effects:     "read",
		Notes:       []string{"Can run anonymously.", "--repo-type accepts one type and applies to repo searches.", modelSearchFilterNote, unifiedSearchSortNote},
	})
	return cmd
}

func addUnifiedSearchFlags(cmd *cobra.Command, opts *unifiedSearchOptions) {
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output results as JSON")
	cmd.Flags().StringVar(&opts.Sort, "sort", "", "Server sort field: created_at, updated_at, star, or fork")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Server sort order: asc or desc")
}

func addUnifiedRepositoryFlags(cmd *cobra.Command, opts *repositorySearchFlags) {
	flags := cmd.Flags()
	flags.StringArrayVar(&opts.RepoType, "repo-type", nil, "Filter by one repository type: code, model, or dataset")
	flags.StringVar(&opts.Model.Task, "task", "", "Model task labels, comma-separated (e.g. text-generation)")
	flags.StringVar(&opts.Model.Library, "library", "", "Model framework labels, comma-separated (e.g. transformers)")
	flags.StringVar(&opts.Model.ParamMin, "param-min", "", "Minimum model parameter size (e.g. 1B)")
	flags.StringVar(&opts.Model.ParamMax, "param-max", "", "Maximum model parameter size (e.g. 7B)")
}

func parseRepoType(values []string) (api.SearchRepositoryType, error) {
	switch len(values) {
	case 0:
		return "", nil
	case 1:
		switch values[0] {
		case "code":
			return api.SearchCodeRepository, nil
		case "model":
			return api.SearchModelRepository, nil
		case "dataset":
			return api.SearchDatasetRepository, nil
		default:
			return "", fmt.Errorf("invalid repo-type: %q (must be code, model, or dataset)", values[0])
		}
	default:
		return "", fmt.Errorf("--repo-type accepts only one value")
	}
}

func runUnifiedSearch(cmd *cobra.Command, f *cmdutil.Factory, kind string, opts unifiedSearchOptions) error {
	if strings.TrimSpace(opts.Query) == "" {
		return fmt.Errorf("search query must not be empty")
	}
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}
	if err := validateChoice("sort", opts.Sort, unifiedSearchSorts); err != nil {
		return err
	}
	if err := validateChoice("order", opts.Order, searchOrders); err != nil {
		return err
	}
	if kind != "repo" && repositorySearchRequested(cmd) {
		return fmt.Errorf("repository filters are only supported for repo searches")
	}

	switch kind {
	case "repo":
		for _, name := range modelSearchFlags {
			flag := cmd.Flags().Lookup(name)
			if flag.Changed && strings.TrimSpace(flag.Value.String()) == "" {
				return fmt.Errorf("--%s must not be empty", name)
			}
		}
		repoType, err := parseRepoType(opts.Repository.RepoType)
		if err != nil {
			return err
		}
		model := opts.Repository.Model
		if model != (api.SearchModelFilters{}) && repoType != api.SearchModelRepository {
			return errors.New(modelSearchFilterNote)
		}
		fetch := func(client *api.Client, opts api.SearchOptions) ([]api.SearchRepository, error) {
			return client.SearchRepositories(opts, repoType, model)
		}
		return searchAndPrint(cmd, f, opts, "repositories", fetch, func(r api.SearchRepository) []string {
			stars := ""
			if r.StarCount != nil {
				stars = fmt.Sprintf("★%d", *r.StarCount)
			}
			return []string{stringValue(r.PathWithNamespace), stars, stringValue(r.Description)}
		})
	case "user":
		return searchAndPrint(cmd, f, opts, "users", (*api.Client).SearchUsers, func(u api.SearchUser) []string {
			return []string{stringValue(u.Username), stringValue(u.Nickname), stringValue(u.URL)}
		})
	case "group":
		return searchAndPrint(cmd, f, opts, "organizations", (*api.Client).SearchOrganizations, func(o api.SearchOrganization) []string {
			return []string{stringValue(o.FullPath), stringValue(o.FullName), stringValue(o.Description), stringValue(o.URL)}
		})
	case "pullrequest":
		return searchAndPrint(cmd, f, opts, "pull requests", (*api.Client).SearchPullRequests, func(p api.SearchPullRequest) []string {
			return searchIssueFields(p.SearchIssueOrPullRequest, "!")
		})
	case "issue":
		return searchAndPrint(cmd, f, opts, "issues", (*api.Client).SearchIssues, func(i api.SearchIssue) []string {
			return searchIssueFields(i.SearchIssueOrPullRequest, "#")
		})
	default:
		return fmt.Errorf("invalid search type: %q (must be one of %s)", kind, unifiedSearchTypes)
	}
}

func searchIssueFields(issue api.SearchIssueOrPullRequest, prefix string) []string {
	var number string
	if issue.IID != nil {
		number = fmt.Sprintf("%s%d", prefix, *issue.IID)
	}
	if issue.State != nil {
		number = strings.TrimSpace(number + " [" + *issue.State + "]")
	}
	return []string{stringValue(issue.PathWithNamespace), number, stringValue(issue.Title), stringValue(issue.URL)}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func searchAndPrint[T any](cmd *cobra.Command, f *cmdutil.Factory, opts unifiedSearchOptions, label string,
	fetch func(*api.Client, api.SearchOptions) ([]T, error), fields func(T) []string,
) error {
	client, err := f.OptionalAPIClient()
	if err != nil {
		return err
	}
	items, err := fetch(client, opts.SearchOptions)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if opts.JSON {
		return cmdutil.WriteJSON(out, items)
	}
	if len(items) == 0 {
		_, err := fmt.Fprintf(out, "No %s found.\n", label)
		return err
	}
	for _, item := range items {
		row := fields(item)
		for i, field := range row {
			row[i] = cmdutil.EscapeTSVField(field)
		}
		if _, err := fmt.Fprintln(out, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}
