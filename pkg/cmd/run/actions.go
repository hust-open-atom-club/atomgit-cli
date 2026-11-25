package run

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdRunCancel(f *cmdutil.Factory) *cobra.Command {
	return newRunActionCommand(f, "cancel", "Cancel a running workflow run", nil)
}

func newCmdRunStop(f *cmdutil.Factory) *cobra.Command {
	return newRunActionCommand(f, "stop", "Stop a running workflow run", nil)
}

func newCmdRunRerun(f *cmdutil.Factory) *cobra.Command {
	return newRunActionCommand(f, "rerun", "Rerun a workflow run", nil)
}

func newCmdRunRetry(f *cmdutil.Factory) *cobra.Command {
	var jobIDs []string
	cmd := newRunActionCommand(f, "retry", "Retry failed jobs in a workflow run", &jobIDs)
	cmd.Flags().StringArrayVarP(&jobIDs, "job", "j", nil, "Retry a job by the ID shown in 'ag-cli run view' (repeatable, defaults to all failed jobs)")
	return cmd
}

func newRunActionCommand(f *cmdutil.Factory, action, description string, jobIDs *[]string) *cobra.Command {
	var yes bool
	validate := func(args []string) error {
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("expected a run ID and optional owner/repo")
		}
		runID := strings.TrimSpace(args[len(args)-1])
		if runID == "" || strings.Contains(runID, "/") {
			return fmt.Errorf("invalid run ID: %q", args[len(args)-1])
		}
		if jobIDs != nil && len(*jobIDs) > 0 {
			_, err := validateJobRunIDs(*jobIDs)
			return err
		}
		return nil
	}
	cmd := &cobra.Command{
		Use:   action + " [<owner>/<repo>] <run-id>",
		Short: description,
		Long: description + `. Requires confirmation unless --yes is supplied.
Writes are never automatically retried or redirected. Success requires HTTP 200
and success=true. Check the run before retrying an uncertain request.`,
		Args: func(_ *cobra.Command, args []string) error { return validate(args) },
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			runID := strings.TrimSpace(remaining[0])
			if !yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Repository: %s\nRun: %s\nAction: %s\n", repository, singleLine(runID), action); err != nil {
					return err
				}
				if jobIDs != nil {
					selection := "all failed jobs (queried after confirmation)"
					if len(*jobIDs) > 0 {
						selection = strings.Join(*jobIDs, ", ")
					}
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Jobs: %s\n", singleLine(selection)); err != nil {
						return err
					}
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply this run action? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "Run action canceled.")
					return err
				}
			}
			token, err := requireToken(f)
			if err != nil {
				return err
			}
			client, err := newActionsClient(f, token)
			if err != nil {
				return err
			}
			client = client.WithContext(cmd.Context())
			switch action {
			case "stop", "cancel":
				err = client.StopRun(repository.Owner, repository.Name, runID)
			case "rerun":
				err = client.RerunRun(repository.Owner, repository.Name, runID)
			case "retry":
				ids := *jobIDs
				if len(ids) == 0 {
					ids, err = failedJobRunIDs(client, repository.Owner, repository.Name, runID)
					if err != nil {
						return fmt.Errorf("list failed jobs for run %s: %w", singleLine(runID), err)
					}
				}
				ids, err = validateJobRunIDs(ids)
				if err != nil {
					return err
				}
				err = client.RetryRunJobs(repository.Owner, repository.Name, runID, ids)
			}
			if err != nil {
				return fmt.Errorf("%s run %s in %s: %w", action, singleLine(runID), repository.String(), err)
			}
			message := map[string]string{"cancel": "Canceled", "stop": "Stopped", "rerun": "Reran", "retry": "Retried"}[action]
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s run %s in %s\n", message, singleLine(runID), repository.String())
			return err
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip run action confirmation")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Run ID, optionally preceded by owner/repo."},
		Output:      "text", Effects: "write", Notes: []string{"Requires confirmation unless --yes is supplied. Writes are never retried or redirected; success requires HTTP 200 and success=true.", "Explicit job IDs are validated before credential loading. Retry without --job selects failed job IDs after confirmation."},
	})
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func failedJobRunIDs(client *actions.Client, owner, repo, runID string) ([]string, error) {
	jobs, err := listRunJobs(client, owner, repo, runID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for _, job := range jobs {
		if strings.EqualFold(strings.TrimSpace(job.Status), "FAILED") {
			if strings.TrimSpace(job.ID) == "" {
				if name := fallback(singleLine(job.Name), singleLine(job.Identifier)); name != "" {
					return nil, fmt.Errorf("failed job %q has no ID", name)
				}
				return nil, fmt.Errorf("failed job has no ID")
			}
			ids = append(ids, job.ID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("run %s has no failed jobs", singleLine(runID))
	}
	return ids, nil
}

func validateJobRunIDs(ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "/") {
			return nil, fmt.Errorf("invalid job ID: %q", id)
		}
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result, nil
}
