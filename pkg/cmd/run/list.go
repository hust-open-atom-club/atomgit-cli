package run

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const maxRunsPerPage = 100

type listOptions struct {
	Branch        string
	Status        string
	Event         string
	Actor         string
	PullRequestID string
	WorkflowID    string
	WorkflowName  string
	StartTime     int64
	EndTime       int64
	Limit         int
}

func newCmdRunList(f *cmdutil.Factory) *cobra.Command {
	opts := listOptions{}
	cmd := &cobra.Command{
		Use:   "list <owner>/<repo>",
		Short: "List workflow runs",
		Example: `  ag run list owner/repo
  ag run list owner/repo --branch main --status failed
  ag run list owner/repo --event push --workflow-name CI --limit 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Filter by head branch")
	cmd.Flags().StringVarP(&opts.Status, "status", "s", "", "Filter by status: completed, running, failed, canceled, ignored, paused, suspend")
	cmd.Flags().StringVar(&opts.Event, "event", "", "Filter by event: mr, push, manual")
	cmd.Flags().StringVar(&opts.Actor, "actor", "", "Filter by triggering username")
	cmd.Flags().StringVar(&opts.PullRequestID, "pr", "", "Filter by pull request number")
	cmd.Flags().StringVar(&opts.WorkflowID, "workflow", "", "Filter by workflow ID")
	cmd.Flags().StringVar(&opts.WorkflowName, "workflow-name", "", "Filter by workflow name")
	cmd.Flags().Int64Var(&opts.StartTime, "start-time", 0, "Filter runs starting at or after this Unix timestamp in milliseconds")
	cmd.Flags().Int64Var(&opts.EndTime, "end-time", 0, "Filter runs ending at or before this Unix timestamp in milliseconds")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of runs to list")
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts listOptions, repository string) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}
	if opts.StartTime < 0 || opts.EndTime < 0 {
		return fmt.Errorf("start-time and end-time must not be negative")
	}
	if opts.StartTime > 0 && opts.EndTime > 0 && opts.StartTime > opts.EndTime {
		return fmt.Errorf("start-time must not be after end-time")
	}

	status, err := normalizeRunStatus(opts.Status)
	if err != nil {
		return err
	}
	event, err := normalizeRunEvent(opts.Event)
	if err != nil {
		return err
	}
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return err
	}
	token, err := requireToken(f)
	if err != nil {
		return err
	}
	client, err := newActionsClient(f, token)
	if err != nil {
		return err
	}

	runs, err := listRuns(client, owner, repo, opts, status, event)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No workflow runs found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tRUN\tTITLE\tWORKFLOW\tBRANCH\tEVENT\tID\tSTARTED")
	for _, workflowRun := range runs {
		runNumber := "-"
		if workflowRun.RunNumber > 0 {
			runNumber = fmt.Sprintf("#%d", workflowRun.RunNumber)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			singleLine(fallback(workflowRun.Status, "UNKNOWN")),
			runNumber,
			singleLine(fallback(workflowRun.Title, "-")),
			singleLine(fallback(workflowRun.WorkflowName, "-")),
			singleLine(fallback(workflowRun.HeadBranch, "-")),
			singleLine(fallback(workflowRun.Event, "-")),
			singleLine(fallback(workflowRun.WorkflowRunID, "-")),
			formatTimestamp(workflowRun.StartTime),
		)
	}
	return w.Flush()
}

func listRuns(client *actions.Client, owner, repo string, opts listOptions, status, event string) ([]actions.Run, error) {
	runs := make([]actions.Run, 0, min(opts.Limit, maxRunsPerPage))
	perPage := min(maxRunsPerPage, opts.Limit)
	for page := 1; len(runs) < opts.Limit; page++ {
		response, err := client.ListRuns(owner, repo, actions.ListRunsOptions{
			Branch:        opts.Branch,
			Status:        status,
			Event:         event,
			Executor:      opts.Actor,
			PullRequestID: opts.PullRequestID,
			WorkflowID:    opts.WorkflowID,
			WorkflowName:  opts.WorkflowName,
			StartTime:     opts.StartTime,
			EndTime:       opts.EndTime,
			Page:          page,
			PerPage:       perPage,
		})
		if err != nil {
			return nil, err
		}
		if len(response.WorkflowRuns) == 0 {
			break
		}
		runs = append(runs, response.WorkflowRuns...)
		if len(runs) >= opts.Limit || len(response.WorkflowRuns) < perPage || (response.TotalCount > 0 && len(runs) >= response.TotalCount) {
			break
		}
	}
	if len(runs) > opts.Limit {
		runs = runs[:opts.Limit]
	}
	return runs, nil
}

func normalizeRunStatus(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	for _, allowed := range []string{"COMPLETED", "RUNNING", "FAILED", "CANCELED", "IGNORED", "PAUSED", "SUSPEND"} {
		if value == allowed {
			return value, nil
		}
	}
	return "", fmt.Errorf("invalid status %q (expected completed, running, failed, canceled, ignored, paused, or suspend)", value)
}

func normalizeRunEvent(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "":
		return "", nil
	case "mr":
		return "MR", nil
	case "push":
		return "Push", nil
	case "manual":
		return "Manual", nil
	default:
		return "", fmt.Errorf("invalid event %q (expected mr, push, or manual)", value)
	}
}
