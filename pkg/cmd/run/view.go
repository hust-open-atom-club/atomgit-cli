package run

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type viewOptions struct {
	JobID        string
	Log          bool
	LogFile      string
	ArtifactID   string
	ArtifactFile string
	Overwrite    bool
}

func newCmdRunView(f *cmdutil.Factory) *cobra.Command {
	opts := viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <owner>/<repo> <run-id>",
		Short: "View a workflow run, jobs, logs, and artifacts",
		Example: `  ag run view owner/repo 12345
  ag run view owner/repo 12345 --job job-id
  ag run view owner/repo 12345 --job job-id --log
  ag run view owner/repo 12345 --job job-id --log-file job-logs.zip
  ag run view owner/repo 12345 --artifact artifact-id
  ag run view owner/repo 12345 --artifact artifact-id --artifact-file build.zip --overwrite`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd, f, opts, args[0], args[1])
		},
	}

	cmd.Flags().StringVarP(&opts.JobID, "job", "j", "", "View a specific job")
	cmd.Flags().BoolVar(&opts.Log, "log", false, "Write the selected job log text to stdout")
	cmd.Flags().StringVar(&opts.LogFile, "log-file", "", "Download the selected job log archive to a file")
	cmd.Flags().StringVar(&opts.ArtifactID, "artifact", "", "Download a specific artifact as a zip archive")
	cmd.Flags().StringVar(&opts.ArtifactFile, "artifact-file", "", "Artifact destination path (defaults to the artifact name)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Replace an existing download destination")
	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts viewOptions, repository, runID string) error {
	opts.JobID = strings.TrimSpace(opts.JobID)
	opts.ArtifactID = strings.TrimSpace(opts.ArtifactID)
	if err := validateViewOptions(opts); err != nil {
		return err
	}
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return err
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("run ID must not be empty")
	}
	token, err := requireToken(f)
	if err != nil {
		return err
	}
	client, err := newActionsClient(f, token)
	if err != nil {
		return err
	}

	switch {
	case opts.Log:
		return writeJobLog(cmd.OutOrStdout(), client, owner, repo, runID, opts.JobID)
	case opts.LogFile != "":
		return downloadJobLog(cmd, client, owner, repo, runID, opts.JobID, opts.LogFile, opts.Overwrite)
	case opts.ArtifactID != "":
		return downloadArtifact(cmd, client, owner, repo, runID, opts.ArtifactID, opts.ArtifactFile, opts.Overwrite)
	default:
		return displayRun(cmd, f, client, owner, repo, runID, opts.JobID)
	}
}

func validateViewOptions(opts viewOptions) error {
	if opts.Log && opts.LogFile != "" {
		return fmt.Errorf("--log and --log-file cannot be used together")
	}
	if (opts.Log || opts.LogFile != "") && strings.TrimSpace(opts.JobID) == "" {
		return fmt.Errorf("--job is required with --log or --log-file")
	}
	if opts.ArtifactFile != "" && strings.TrimSpace(opts.ArtifactID) == "" {
		return fmt.Errorf("--artifact is required with --artifact-file")
	}
	if opts.ArtifactID != "" && (opts.JobID != "" || opts.Log || opts.LogFile != "") {
		return fmt.Errorf("--artifact cannot be combined with job or log options")
	}
	if opts.Overwrite && opts.LogFile == "" && opts.ArtifactID == "" {
		return fmt.Errorf("--overwrite requires --log-file or --artifact")
	}
	return nil
}

func displayRun(cmd *cobra.Command, f *cmdutil.Factory, client *actions.Client, owner, repo, runID, jobID string) error {
	workflowRun, err := client.GetRun(owner, repo, runID)
	if err != nil {
		return err
	}

	var jobs []actions.Job
	if jobID != "" {
		job, err := client.GetJob(owner, repo, runID, jobID)
		if err != nil {
			return err
		}
		jobs = []actions.Job{job}
	} else {
		response, err := client.ListJobs(owner, repo, runID)
		if err != nil {
			return err
		}
		jobs = response.Jobs
		if len(jobs) == 0 {
			jobs = jobsFromStages(workflowRun.Stages)
		}
	}

	artifacts := []actions.Artifact(nil)
	if jobID == "" {
		artifacts, err = listRunArtifacts(client, owner, repo, runID)
		if err != nil {
			return err
		}
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Run: %s\n", singleLine(fallback(workflowRun.WorkflowRunID, runID)))
	if workflowRun.RunNumber > 0 {
		fmt.Fprintf(out, "Number: #%d\n", workflowRun.RunNumber)
	}
	fmt.Fprintf(out, "Workflow: %s\n", singleLine(fallback(workflowRun.WorkflowName, "-")))
	if workflowRun.Title != "" {
		fmt.Fprintf(out, "Title: %s\n", singleLine(workflowRun.Title))
	}
	fmt.Fprintf(out, "Status: %s\n", singleLine(fallback(workflowRun.Status, "UNKNOWN")))
	fmt.Fprintf(out, "Event: %s\n", singleLine(fallback(workflowRun.Event, "-")))
	fmt.Fprintf(out, "Branch: %s\n", singleLine(fallback(workflowRun.HeadBranch, "-")))
	fmt.Fprintf(out, "Commit: %s\n", singleLine(fallback(workflowRun.HeadSHA, "-")))
	fmt.Fprintf(out, "Actor: %s\n", singleLine(fallback(workflowRun.Actor.Login, workflowRun.Actor.Name, "-")))
	fmt.Fprintf(out, "Started: %s\n", formatTimestamp(workflowRun.StartTime))
	fmt.Fprintf(out, "Finished: %s\n", formatTimestamp(workflowRun.EndTime))
	fmt.Fprintf(out, "URL: %s\n", workflowRunURL(f, owner, repo, runID))

	printJobs(out, jobs)
	if jobID == "" {
		printArtifacts(out, artifacts)
	}
	return nil
}

func writeJobLog(out io.Writer, client *actions.Client, owner, repo, runID, jobID string) error {
	resp, err := client.DownloadJobLog(owner, repo, runID, jobID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return writeJobLogOutput(out, resp.Body)
}

func downloadJobLog(cmd *cobra.Command, client *actions.Client, owner, repo, runID, jobID, destination string, overwrite bool) error {
	destination, err := validateDownloadDestination(destination, overwrite)
	if err != nil {
		return fmt.Errorf("download job log: %w", err)
	}
	resp, err := client.DownloadJobLog(owner, repo, runID, jobID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	path, err := writeDownload(destination, resp.Body, overwrite)
	if err != nil {
		return fmt.Errorf("download job log: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Downloaded job log to %s\n", path)
	return nil
}

func downloadArtifact(cmd *cobra.Command, client *actions.Client, owner, repo, runID, artifactID, destination string, overwrite bool) error {
	artifact, err := client.GetArtifact(owner, repo, artifactID)
	if err != nil {
		return err
	}
	if artifact.WorkflowRunID != "" && artifact.WorkflowRunID != runID {
		return fmt.Errorf("artifact %s belongs to run %s, not run %s", artifactID, artifact.WorkflowRunID, runID)
	}
	if destination == "" {
		destination = artifactFilename(artifact)
	}
	destination, err = validateDownloadDestination(destination, overwrite)
	if err != nil {
		return fmt.Errorf("download artifact: %w", err)
	}

	resp, err := client.DownloadArtifact(owner, repo, artifactID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	path, err := writeDownload(destination, resp.Body, overwrite)
	if err != nil {
		return fmt.Errorf("download artifact: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Downloaded artifact to %s\n", path)
	return nil
}

func listRunArtifacts(client *actions.Client, owner, repo, runID string) ([]actions.Artifact, error) {
	const perPage = 100
	artifacts := make([]actions.Artifact, 0)
	for page := 1; ; page++ {
		response, err := client.ListRunArtifacts(owner, repo, runID, actions.ListArtifactsOptions{
			Sort:      "created",
			Direction: "desc",
			Page:      page,
			PerPage:   perPage,
		})
		if err != nil {
			return nil, err
		}
		if len(response.Artifacts) == 0 {
			break
		}
		artifacts = append(artifacts, response.Artifacts...)
		if len(response.Artifacts) < perPage || (response.TotalCount > 0 && len(artifacts) >= response.TotalCount) {
			break
		}
	}
	return artifacts, nil
}

func printJobs(out io.Writer, jobs []actions.Job) {
	if len(jobs) == 0 {
		fmt.Fprintln(out, "Jobs: none")
		return
	}
	fmt.Fprintln(out, "Jobs:")
	for _, job := range jobs {
		fmt.Fprintf(out, "  [%s] %s (%s)\n",
			singleLine(fallback(job.Status, "UNKNOWN")),
			singleLine(fallback(job.Name, job.Identifier, "unnamed")),
			singleLine(fallback(job.ID, "-")),
		)
		if len(job.Steps) == 0 {
			fmt.Fprintln(out, "    Steps: none")
			continue
		}
		for _, step := range job.Steps {
			fmt.Fprintf(out, "    [%s] %s\n",
				singleLine(fallback(step.Status, "UNKNOWN")),
				singleLine(fallback(step.Name, step.Task, "unnamed")),
			)
		}
	}
}

func printArtifacts(out io.Writer, artifacts []actions.Artifact) {
	if len(artifacts) == 0 {
		fmt.Fprintln(out, "Artifacts: none")
		return
	}
	fmt.Fprintln(out, "Artifacts:")
	for _, artifact := range artifacts {
		fmt.Fprintf(out, "  %s (%s, %s)\n",
			singleLine(fallback(artifact.Name, "unnamed")),
			singleLine(fallback(artifact.ID, "-")),
			formatBytes(artifact.SizeBytes),
		)
	}
}

func jobsFromStages(stages []actions.Stage) []actions.Job {
	var jobs []actions.Job
	for _, stage := range stages {
		jobs = append(jobs, stage.Jobs...)
	}
	return jobs
}

func workflowRunURL(f *cmdutil.Factory, owner, repo, runID string) string {
	host := "atomgit.com"
	if f != nil && f.Config != nil && strings.TrimSpace(f.Config.GetHost()) != "" {
		host = strings.TrimSpace(f.Config.GetHost())
	}
	return (&url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/" + owner + "/" + repo + "/actions/runs/" + runID,
	}).String()
}
