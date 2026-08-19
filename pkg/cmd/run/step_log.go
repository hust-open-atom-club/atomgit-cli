package run

import (
	"fmt"
	"io"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	stepLogPageLimit = 1000
	stepLogSort      = "asc"
)

type stepLogOptions struct {
	Output    string
	Overwrite bool
}

func newCmdRunStepLog(f *cmdutil.Factory) *cobra.Command {
	opts := stepLogOptions{}
	cmd := &cobra.Command{
		Use:   "step-log [<owner>/<repo>] <run-id> <job-id> <step-id>",
		Short: "Fetch step-level logs for a workflow job",
		Long: `Retrieve paginated AtomGit Actions step logs and write them as text.

Use ag run view to discover step IDs. --output writes the complete log
atomically and refuses to replace an existing file unless --overwrite is set.`,
		Example: `  ag run step-log owner/repo <run-id> <job-id> <step-id>
  ag run step-log <run-id> <job-id> <step-id>
  ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log
  ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log --overwrite`,
		Args: cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStepLog(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVar(&opts.Output, "output", "", "Write the complete log to a file instead of stdout")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Replace an existing --output destination")
	return cmd
}

func runStepLog(cmd *cobra.Command, f *cmdutil.Factory, opts stepLogOptions, args []string) error {
	if opts.Overwrite && strings.TrimSpace(opts.Output) == "" {
		return fmt.Errorf("--overwrite requires --output")
	}

	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 3)
	if err != nil {
		return err
	}
	runID := strings.TrimSpace(remaining[0])
	jobID := strings.TrimSpace(remaining[1])
	stepID := strings.TrimSpace(remaining[2])
	if runID == "" || jobID == "" || stepID == "" {
		return fmt.Errorf("run ID, job ID, and step ID are required")
	}

	token, err := requireToken(f)
	if err != nil {
		return err
	}
	client, err := newActionsClient(f, token)
	if err != nil {
		return err
	}

	if destination := strings.TrimSpace(opts.Output); destination != "" {
		destination, err = preflightDownloadDestination(destination, opts.Overwrite)
		if err != nil {
			return fmt.Errorf("write step log: %w", err)
		}
		reader, writer := io.Pipe()
		errc := make(chan error, 1)
		go func() {
			err := writeStepLogs(writer, client, repository.Owner, repository.Name, runID, jobID, stepID)
			if err != nil {
				_ = writer.CloseWithError(err)
			} else {
				_ = writer.Close()
			}
			errc <- err
		}()
		path, writeErr := writeDownload(destination, reader, opts.Overwrite)
		if writeErr != nil {
			_ = reader.Close()
		}
		copyErr := <-errc
		if writeErr != nil {
			return fmt.Errorf("write step log: %w", writeErr)
		}
		if copyErr != nil {
			return copyErr
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Wrote step log to %s\n", path)
		return nil
	}

	return writeStepLogs(cmd.OutOrStdout(), client, repository.Owner, repository.Name, runID, jobID, stepID)
}

func writeStepLogs(out io.Writer, client *actions.Client, owner, repo, runID, jobID, stepID string) error {
	var offset int64
	for {
		page, err := client.GetStepLog(owner, repo, runID, jobID, actions.StepLogRequest{
			StepID: stepID,
			Offset: offset,
			Limit:  stepLogPageLimit,
			Sort:   stepLogSort,
		})
		if err != nil {
			return fmt.Errorf("failed to get step log for %s/%s run %s job %s step %s: %w", owner, repo, runID, jobID, stepID, err)
		}
		if page.Log != "" {
			if _, err := io.WriteString(out, page.Log); err != nil {
				return fmt.Errorf("write step log: %w", err)
			}
		}
		if !page.HasMore {
			return nil
		}
		if page.EndOffset <= offset {
			return fmt.Errorf("step log pagination stalled at offset %d", offset)
		}
		offset = page.EndOffset
	}
}
