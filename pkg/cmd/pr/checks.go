package pr

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	defaultChecksInterval = 10 * time.Second
	checksPageSize        = 100
	maxChecksPages        = 100
)

type checksOptions struct {
	Watch    bool
	Interval time.Duration
}

type checksState int

const (
	checksSuccess checksState = iota
	checksPending
	checksFailure
)

func newCmdPRChecks(f *cmdutil.Factory) *cobra.Command {
	opts := checksOptions{Interval: defaultChecksInterval}
	cmd := &cobra.Command{
		Use:   "checks [<owner>/<repo>] <number>",
		Short: "Show CI checks for a pull request's current head commit",
		Long:  "Show AtomGit Actions runs for a pull request's current head commit. This command does not infer or report required-check semantics.",
		Example: `  ag pr checks owner/repo 42
  ag pr checks 42 --watch
  ag pr checks owner/repo 42 --watch --interval 5s`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPRChecks(cmd, f, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.Watch, "watch", "w", false, "Watch checks until they reach a terminal state")
	cmd.Flags().DurationVarP(&opts.Interval, "interval", "i", defaultChecksInterval, "Polling interval when using --watch")
	return cmd
}

func runPRChecks(cmd *cobra.Command, f *cmdutil.Factory, opts checksOptions, args []string) error {
	if opts.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if cmd.Flags().Changed("interval") && !opts.Watch {
		return fmt.Errorf("--interval requires --watch")
	}

	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
	if err != nil {
		return err
	}
	number, err := parsePRNumber(remaining[0])
	if err != nil {
		return err
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	prClient, actionsClient, err := newChecksClients(f, token)
	if err != nil {
		return err
	}

	for {
		runs, err := currentPRChecks(prClient, actionsClient, repository.Owner, repository.Name, number)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			return fmt.Errorf("no workflow runs found for PR #%s current head commit", number)
		}

		if err := printPRChecks(cmd, f, repository, runs); err != nil {
			return err
		}
		state := summarizeChecks(runs)
		switch state {
		case checksFailure:
			return fmt.Errorf("one or more checks failed or were canceled")
		case checksSuccess:
			return nil
		case checksPending:
			if !opts.Watch {
				return fmt.Errorf("checks are still pending")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Checks are pending; polling again in %s.\n\n", opts.Interval)
		}

		if err := waitForChecksPoll(cmd.Context(), opts.Interval); err != nil {
			return err
		}
	}
}

func newChecksClients(f *cmdutil.Factory, token string) (*api.Client, *actions.Client, error) {
	if f.HttpClient == nil {
		return api.NewClient(token), actions.NewClient(token), nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), actions.NewClientWithHTTPClient(token, httpClient), nil
}

func currentPRChecks(prClient *api.Client, actionsClient *actions.Client, owner, repo, number string) ([]actions.Run, error) {
	var pullRequest api.PullRequest
	if err := prClient.Get(fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number), &pullRequest); err != nil {
		return nil, fmt.Errorf("failed to get PR #%s: %w", number, err)
	}
	headSHA := strings.TrimSpace(pullRequest.Head.SHA)
	if headSHA == "" {
		return nil, fmt.Errorf("PR #%s response did not include a head commit SHA", number)
	}

	return listChecksForHead(actionsClient, owner, repo, number, headSHA)
}

func listChecksForHead(client *actions.Client, owner, repo, number, headSHA string) ([]actions.Run, error) {
	matched := make([]actions.Run, 0)
	seen := make(map[string]struct{})
	fetched := 0
	for page := 1; page <= maxChecksPages; page++ {
		response, err := client.ListRuns(owner, repo, actions.ListRunsOptions{
			PullRequestID: number,
			Page:          page,
			PerPage:       checksPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list checks for PR #%s: %w", number, err)
		}
		fetched += len(response.WorkflowRuns)
		for _, run := range response.WorkflowRuns {
			if !strings.EqualFold(strings.TrimSpace(run.HeadSHA), headSHA) {
				continue
			}
			key := strings.TrimSpace(run.WorkflowRunID)
			if key == "" {
				key = fmt.Sprintf("%s\x00%d\x00%s", run.WorkflowID, run.RunNumber, run.WorkflowName)
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			matched = append(matched, run)
		}

		if len(response.WorkflowRuns) == 0 || len(response.WorkflowRuns) < checksPageSize || (response.TotalCount > 0 && fetched >= response.TotalCount) {
			return matched, nil
		}
	}
	return nil, fmt.Errorf("failed to list checks for PR #%s: pagination exceeded %d pages", number, maxChecksPages)
}

func printPRChecks(cmd *cobra.Command, f *cmdutil.Factory, repository cmdutil.Repository, runs []actions.Run) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CHECK\tSTATUS\tBRANCH\tCOMMIT\tURL")
	for _, run := range runs {
		name := firstNonEmpty(run.WorkflowName, run.Title, run.WorkflowRunID, "-")
		status := firstNonEmpty(run.Status, "UNKNOWN")
		branch := firstNonEmpty(run.HeadBranch, "-")
		commit := firstNonEmpty(run.HeadSHA, "-")
		runID := strings.TrimSpace(run.WorkflowRunID)
		runURL := "-"
		if runID != "" {
			runURL = cmdutil.ResolveWebURL("", f.Config.GetHost(), repository.Owner, repository.Name, "actions", "runs", runID)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", singleLineCheck(name), singleLineCheck(status), singleLineCheck(branch), singleLineCheck(commit), runURL)
	}
	return w.Flush()
}

func summarizeChecks(runs []actions.Run) checksState {
	state := checksSuccess
	for _, run := range runs {
		switch strings.ToUpper(strings.TrimSpace(run.Status)) {
		case "COMPLETED", "IGNORED":
		case "FAILED", "CANCELED":
			return checksFailure
		default:
			state = checksPending
		}
	}
	return state
}

func waitForChecksPoll(ctx context.Context, interval time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func singleLineCheck(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
