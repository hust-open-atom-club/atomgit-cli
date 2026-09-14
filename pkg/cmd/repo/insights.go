package repo

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const defaultInsightsLimit = 30

type insightListOptions struct {
	Limit int
	JSON  bool
}

type languageJSON struct {
	Language   string  `json:"language"`
	Percentage float64 `json:"percentage"`
}

type contributionDayJSON struct {
	Date         string `json:"date"`
	Additions    int64  `json:"additions"`
	Deletions    int64  `json:"deletions"`
	TotalChanges int64  `json:"totalChanges"`
	CommitCount  int64  `json:"commitCount"`
}

type contributorJSON struct {
	Name          string                `json:"name"`
	Email         string                `json:"email"`
	Additions     int64                 `json:"additions"`
	Deletions     int64                 `json:"deletions"`
	TotalChanges  int64                 `json:"totalChanges"`
	CommitCount   int64                 `json:"commitCount"`
	Contributions []contributionDayJSON `json:"contributions"`
}

type repositoryEventJSON struct {
	Action           int    `json:"action"`
	ActionName       string `json:"actionName"`
	AuthorID         int64  `json:"authorId"`
	AuthorUsername   string `json:"authorUsername"`
	AuthorName       string `json:"authorName"`
	AuthorURL        string `json:"authorUrl"`
	CreatedAt        string `json:"createdAt"`
	ProjectID        int64  `json:"projectId"`
	Title            string `json:"title"`
	FilterSensitive  bool   `json:"filterSensitive"`
	TargetID         int64  `json:"targetId"`
	TargetIID        int64  `json:"targetIid"`
	TargetTitle      string `json:"targetTitle"`
	TargetType       string `json:"targetType"`
	TargetTypeFormat string `json:"targetTypeFormat"`
}

type watcherJSON struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	ObjectID  string `json:"objectId"`
	WatchedAt string `json:"watchedAt"`
}

type stargazerJSON struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	ObjectID  string `json:"objectId"`
	StarredAt string `json:"starredAt"`
}

type downloadDetailJSON struct {
	Date           string `json:"date"`
	RepositoryID   string `json:"repositoryId"`
	TodayDownloads int64  `json:"todayDownloads"`
	TotalDownloads int64  `json:"totalDownloads"`
}

type downloadsJSON struct {
	PeriodDownloads  int64                `json:"periodDownloads"`
	HistoryDownloads int64                `json:"historyDownloads"`
	Details          []downloadDetailJSON `json:"details"`
}

func newCmdRepoInsights(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Inspect repository activity and statistics",
		Long:  "Inspect read-only repository languages, contributor statistics, events, watchers, stargazers, and download statistics.",
	}
	cmd.AddCommand(newCmdRepoInsightsLanguages(f))
	cmd.AddCommand(newCmdRepoInsightsContributors(f))
	cmd.AddCommand(newCmdRepoInsightsEvents(f))
	cmd.AddCommand(newCmdRepoInsightsWatchers(f))
	cmd.AddCommand(newCmdRepoInsightsStargazers(f))
	cmd.AddCommand(newCmdRepoInsightsDownloads(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoInsightsLanguages(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "languages [<owner>/<repo>]",
		Short: "Show repository language percentages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			languages, err := api.GetRepositoryLanguages(client, repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("get languages for %s: %w", repository, err)
			}
			entries := languagesJSON(languages)
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), entries)
			}
			return writeLanguages(cmd.OutOrStdout(), entries)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output languages as JSON")
	return cmd
}

func newCmdRepoInsightsContributors(f *cmdutil.Factory) *cobra.Command {
	opts := &insightListOptions{Limit: defaultInsightsLimit}
	cmd := &cobra.Command{
		Use:   "contributors [<owner>/<repo>]",
		Short: "List repository contributor statistics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInsightsLimit(opts.Limit); err != nil {
				return err
			}
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			contributors, err := api.GetRepositoryContributorStatistics(client, repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("get contributor statistics for %s: %w", repository, err)
			}
			result := contributorsJSON(contributors, opts.Limit)
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeContributors(cmd.OutOrStdout(), result)
		},
	}
	addInsightListFlags(cmd, opts, "contributors")
	return cmd
}

func newCmdRepoInsightsEvents(f *cmdutil.Factory) *cobra.Command {
	opts := &insightListOptions{Limit: defaultInsightsLimit}
	cmd := &cobra.Command{
		Use:   "events [<owner>/<repo>]",
		Short: "List repository activity events",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInsightsLimit(opts.Limit); err != nil {
				return err
			}
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			events, err := listRepositoryEvents(client, repository.Owner, repository.Name, opts.Limit)
			if err != nil {
				return fmt.Errorf("list events for %s: %w", repository, err)
			}
			result := repositoryEventsJSON(events)
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeRepositoryEvents(cmd.OutOrStdout(), result)
		},
	}
	addInsightListFlags(cmd, opts, "events")
	return cmd
}

func newCmdRepoInsightsWatchers(f *cmdutil.Factory) *cobra.Command {
	opts := &insightListOptions{Limit: defaultInsightsLimit}
	cmd := &cobra.Command{
		Use:   "watchers [<owner>/<repo>]",
		Short: "List repository watchers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInsightsLimit(opts.Limit); err != nil {
				return err
			}
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			watchers, err := api.ListRepositoryWatchers(client, repository.Owner, repository.Name, opts.Limit)
			if err != nil {
				return fmt.Errorf("list watchers for %s: %w", repository, err)
			}
			result := watchersJSON(watchers)
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeWatchers(cmd.OutOrStdout(), result)
		},
	}
	addInsightListFlags(cmd, opts, "watchers")
	return cmd
}

func newCmdRepoInsightsStargazers(f *cmdutil.Factory) *cobra.Command {
	opts := &insightListOptions{Limit: defaultInsightsLimit}
	cmd := &cobra.Command{
		Use:   "stargazers [<owner>/<repo>]",
		Short: "List repository stargazers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInsightsLimit(opts.Limit); err != nil {
				return err
			}
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			stargazers, err := api.ListRepositoryStargazers(client, repository.Owner, repository.Name, opts.Limit)
			if err != nil {
				return fmt.Errorf("list stargazers for %s: %w", repository, err)
			}
			result := stargazersJSON(stargazers)
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeStargazers(cmd.OutOrStdout(), result)
		},
	}
	addInsightListFlags(cmd, opts, "stargazers")
	return cmd
}

func newCmdRepoInsightsDownloads(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "downloads [<owner>/<repo>]",
		Short: "Show repository download statistics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, client, err := repositoryInsightsClient(f, args)
			if err != nil {
				return err
			}
			statistics, err := api.GetRepositoryDownloadStatistics(client, repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("get download statistics for %s: %w", repository, err)
			}
			result := repositoryDownloadsJSON(statistics)
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeDownloads(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output download statistics as JSON")
	return cmd
}

func repositoryInsightsClient(f *cmdutil.Factory, args []string) (cmdutil.Repository, *api.Client, error) {
	repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
	if err != nil {
		return cmdutil.Repository{}, nil, err
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.Repository{}, nil, cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return cmdutil.Repository{}, nil, err
	}
	return repository, client, nil
}

func addInsightListFlags(cmd *cobra.Command, opts *insightListOptions, noun string) {
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultInsightsLimit, "Maximum number of "+noun+" to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output "+noun+" as JSON")
}

func validateInsightsLimit(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	return nil
}

func languagesJSON(languages map[string]float64) []languageJSON {
	result := make([]languageJSON, 0, len(languages))
	for language, percentage := range languages {
		result = append(result, languageJSON{Language: language, Percentage: percentage})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Percentage != result[j].Percentage {
			return result[i].Percentage > result[j].Percentage
		}
		return result[i].Language < result[j].Language
	})
	return result
}

func contributorsJSON(contributors []api.RepositoryContributorStatistics, limit int) []contributorJSON {
	sort.SliceStable(contributors, func(i, j int) bool {
		if contributors[i].Overview.CommitCount != contributors[j].Overview.CommitCount {
			return contributors[i].Overview.CommitCount > contributors[j].Overview.CommitCount
		}
		return contributors[i].Name < contributors[j].Name
	})
	if len(contributors) > limit {
		contributors = contributors[:limit]
	}
	result := make([]contributorJSON, 0, len(contributors))
	for _, contributor := range contributors {
		sort.SliceStable(contributor.Contributions, func(i, j int) bool {
			return contributor.Contributions[i].Date > contributor.Contributions[j].Date
		})
		days := make([]contributionDayJSON, 0, len(contributor.Contributions))
		for _, day := range contributor.Contributions {
			days = append(days, contributionDayJSON{
				Date: day.Date, Additions: day.Additions, Deletions: day.Deletions,
				TotalChanges: day.TotalChanges, CommitCount: day.CommitCount,
			})
		}
		result = append(result, contributorJSON{
			Name: contributor.Name, Email: contributor.Email,
			Additions: contributor.Overview.Additions, Deletions: contributor.Overview.Deletions,
			TotalChanges: contributor.Overview.TotalChanges, CommitCount: contributor.Overview.CommitCount,
			Contributions: days,
		})
	}
	return result
}

func listRepositoryEvents(client *api.Client, owner, repo string, limit int) ([]api.RepositoryEvent, error) {
	if err := validateInsightsLimit(limit); err != nil {
		return nil, err
	}
	result := make([]api.RepositoryEvent, 0, min(limit, 100))
	seenPages := make(map[string]struct{})
	for page := 1; ; page++ {
		response, err := api.GetRepositoryEventsPage(client, owner, repo, page, 100)
		if err != nil {
			return nil, err
		}
		if len(response.Events) == 0 {
			if bool(response.HasNextPage) {
				return nil, fmt.Errorf("pagination made no progress on page %d", page)
			}
			return result, nil
		}
		pageKeyParts := make([]string, 0, len(response.Events))
		for _, event := range response.Events {
			pageKeyParts = append(pageKeyParts, repositoryEventKey(event))
		}
		pageKey := strings.Join(pageKeyParts, "\x01")
		if _, exists := seenPages[pageKey]; exists {
			return nil, fmt.Errorf("pagination repeated an earlier event page on page %d", page)
		}
		seenPages[pageKey] = struct{}{}
		for _, event := range response.Events {
			result = append(result, event)
			if len(result) == limit {
				return result, nil
			}
		}
		if !bool(response.HasNextPage) {
			return result, nil
		}
	}
}

func repositoryEventKey(event api.RepositoryEvent) string {
	return strings.Join([]string{
		strconv.Itoa(event.Action), strconv.FormatInt(event.AuthorID, 10),
		event.CreatedAt, strconv.FormatInt(event.ProjectID, 10),
		strconv.FormatInt(event.TargetID, 10), strconv.FormatInt(event.TargetIID, 10),
		event.TargetType, event.TargetTitle, event.Title,
	}, "\x00")
}

func repositoryEventsJSON(events []api.RepositoryEvent) []repositoryEventJSON {
	result := make([]repositoryEventJSON, 0, len(events))
	for _, event := range events {
		authorID := event.AuthorID
		if authorID == 0 {
			authorID = event.Author.ID
		}
		authorUsername := event.AuthorUsername
		if authorUsername == "" {
			authorUsername = event.Author.Username
		}
		result = append(result, repositoryEventJSON{
			Action: event.Action, ActionName: event.ActionName,
			AuthorID: authorID, AuthorUsername: authorUsername,
			AuthorName: event.Author.Name, AuthorURL: event.Author.WebURL,
			CreatedAt: event.CreatedAt, ProjectID: event.ProjectID, Title: event.Title,
			FilterSensitive: bool(event.FilterSensitive), TargetID: event.TargetID,
			TargetIID: event.TargetIID, TargetTitle: event.TargetTitle,
			TargetType: event.TargetType, TargetTypeFormat: event.TargetTypeFormat,
		})
	}
	return result
}

func watchersJSON(users []api.RepositoryInsightUser) []watcherJSON {
	result := make([]watcherJSON, 0, len(users))
	for _, user := range users {
		result = append(result, watcherJSON{ID: user.ID, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL, ObjectID: user.ObjectID, WatchedAt: user.WatchedAt})
	}
	return result
}

func stargazersJSON(users []api.RepositoryInsightUser) []stargazerJSON {
	result := make([]stargazerJSON, 0, len(users))
	for _, user := range users {
		result = append(result, stargazerJSON{ID: user.ID, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL, ObjectID: user.ObjectID, StarredAt: user.StarredAt})
	}
	return result
}

func repositoryDownloadsJSON(statistics api.RepositoryDownloadStatistics) downloadsJSON {
	sort.SliceStable(statistics.Details, func(i, j int) bool {
		return statistics.Details[i].Date > statistics.Details[j].Date
	})
	details := make([]downloadDetailJSON, 0, len(statistics.Details))
	for _, detail := range statistics.Details {
		details = append(details, downloadDetailJSON{
			Date: detail.Date, RepositoryID: detail.RepositoryID,
			TodayDownloads: detail.TodayDownloads, TotalDownloads: detail.TotalDownloads,
		})
	}
	return downloadsJSON{PeriodDownloads: statistics.PeriodTotal, HistoryDownloads: statistics.HistoryTotal, Details: details}
}

func writeLanguages(out io.Writer, languages []languageJSON) error {
	if len(languages) == 0 {
		_, err := fmt.Fprintln(out, "No language statistics found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LANGUAGE\tPERCENT")
	for _, language := range languages {
		fmt.Fprintf(w, "%s\t%.2f%%\n", cmdutil.EscapeTSVField(language.Language), language.Percentage)
	}
	return w.Flush()
}

func writeContributors(out io.Writer, contributors []contributorJSON) error {
	if len(contributors) == 0 {
		_, err := fmt.Fprintln(out, "No contributor statistics found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CONTRIBUTOR\tCOMMITS\tADDITIONS\tDELETIONS\tCHANGES")
	for _, contributor := range contributors {
		name := contributor.Name
		if name == "" {
			name = contributor.Email
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n", cmdutil.EscapeTSVField(name), contributor.CommitCount, contributor.Additions, contributor.Deletions, contributor.TotalChanges)
	}
	return w.Flush()
}

func writeRepositoryEvents(out io.Writer, events []repositoryEventJSON) error {
	if len(events) == 0 {
		_, err := fmt.Fprintln(out, "No repository events found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CREATED\tACTION\tAUTHOR\tTITLE")
	for _, event := range events {
		title := event.TargetTitle
		if title == "" {
			title = event.Title
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			cmdutil.EscapeTSVField(event.CreatedAt), cmdutil.EscapeTSVField(event.ActionName),
			cmdutil.EscapeTSVField(event.AuthorUsername), cmdutil.EscapeTSVField(title))
	}
	return w.Flush()
}

func writeWatchers(out io.Writer, watchers []watcherJSON) error {
	if len(watchers) == 0 {
		_, err := fmt.Fprintln(out, "No watchers found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tWATCHED AT")
	for _, watcher := range watchers {
		fmt.Fprintf(w, "%s\t%s\t%s\n", cmdutil.EscapeTSVField(watcher.Login), cmdutil.EscapeTSVField(watcher.Name), cmdutil.EscapeTSVField(watcher.WatchedAt))
	}
	return w.Flush()
}

func writeStargazers(out io.Writer, stargazers []stargazerJSON) error {
	if len(stargazers) == 0 {
		_, err := fmt.Fprintln(out, "No stargazers found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tSTARRED AT")
	for _, stargazer := range stargazers {
		fmt.Fprintf(w, "%s\t%s\t%s\n", cmdutil.EscapeTSVField(stargazer.Login), cmdutil.EscapeTSVField(stargazer.Name), cmdutil.EscapeTSVField(stargazer.StarredAt))
	}
	return w.Flush()
}

func writeDownloads(out io.Writer, downloads downloadsJSON) error {
	if _, err := fmt.Fprintf(out, "Period downloads: %d\nHistory downloads: %d\n", downloads.PeriodDownloads, downloads.HistoryDownloads); err != nil {
		return err
	}
	if len(downloads.Details) == 0 {
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTODAY\tTOTAL")
	for _, detail := range downloads.Details {
		fmt.Fprintf(w, "%s\t%d\t%d\n", cmdutil.EscapeTSVField(detail.Date), detail.TodayDownloads, detail.TotalDownloads)
	}
	return w.Flush()
}
