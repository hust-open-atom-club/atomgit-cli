package run

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

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
	cmd := &cobra.Command{
		Use:   action + " [<owner>/<repo>] <run-id>",
		Short: description,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			runID := strings.TrimSpace(remaining[0])
			if runID == "" || strings.Contains(runID, "/") {
				return fmt.Errorf("invalid run ID: %q", remaining[0])
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
			case "stop":
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
			message := map[string]string{"stop": "Stopped", "rerun": "Reran", "retry": "Retried"}[action]
			fmt.Fprintf(cmd.OutOrStdout(), "%s run %s in %s\n", message, singleLine(runID), repository.String())
			return nil
		},
	}
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
			return nil, fmt.Errorf("invalid job run ID: %q", id)
		}
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result, nil
}
