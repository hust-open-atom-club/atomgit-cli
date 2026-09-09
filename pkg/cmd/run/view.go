package run

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
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
		Use:   "view [<owner>/<repo>] <run-id>",
		Short: "View a workflow run, jobs, logs, and artifacts",
		Example: `  ag run view owner/repo 12345
  ag run view 12345
  ag run view owner/repo 12345 --job job-id
  ag run view owner/repo 12345 --job job-id --log
  ag run view owner/repo 12345 --job job-id --log-file job-logs.zip
  ag run view owner/repo 12345 --artifact artifact-id
  ag run view owner/repo 12345 --artifact artifact-id --artifact-file build.zip --overwrite`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			return runView(cmd, f, opts, repository.String(), remaining[0])
		},
	}
	cmdutil.AddRepositoryContextHelp(cmd)

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
		apiJobs, err := listRunJobs(client, owner, repo, runID)
		if err != nil {
			return err
		}
		jobs = mergeJobs(apiJobs, jobsFromStages(workflowRun.Stages))
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

func listRunJobs(client *actions.Client, owner, repo, runID string) ([]actions.Job, error) {
	const (
		perPage  = 100
		maxPages = 100
	)

	jobs := make([]actions.Job, 0)
	seenJobs := make(map[string]struct{})
	seenPages := make(map[string]struct{})
	expectedTotal := 0
	totalTrusted := true

	for page := 1; page <= maxPages; page++ {
		response, err := client.ListJobs(owner, repo, runID, actions.ListJobsOptions{Page: page, PerPage: perPage})
		if err != nil {
			return nil, err
		}

		if response.TotalCount > 0 {
			if expectedTotal == 0 {
				expectedTotal = response.TotalCount
			} else if response.TotalCount != expectedTotal {
				totalTrusted = false
			}
		}
		if len(response.Jobs) == 0 {
			if totalTrusted && expectedTotal > len(jobs) {
				return nil, jobsPaginationNoProgressError(page, len(jobs), expectedTotal, totalTrusted)
			}
			return jobs, nil
		}

		fingerprint := jobPageFingerprint(response.Jobs)
		if _, exists := seenPages[fingerprint]; exists {
			return nil, jobsPaginationNoProgressError(page, len(jobs), expectedTotal, totalTrusted)
		}
		seenPages[fingerprint] = struct{}{}

		added := 0
		for _, job := range response.Jobs {
			if key := jobIdentity(job); key != "" {
				if _, exists := seenJobs[key]; exists {
					continue
				}
				seenJobs[key] = struct{}{}
			}
			jobs = append(jobs, job)
			added++
		}

		if expectedTotal > 0 && expectedTotal < len(jobs) {
			totalTrusted = false
		}
		if totalTrusted && expectedTotal > 0 && len(jobs) >= expectedTotal {
			return jobs, nil
		}
		if len(response.Jobs) < perPage {
			return jobs, nil
		}
		if added == 0 {
			return nil, jobsPaginationNoProgressError(page, len(jobs), expectedTotal, totalTrusted)
		}
	}

	return nil, fmt.Errorf("workflow run jobs pagination exceeded %d pages", maxPages)
}

func jobsPaginationNoProgressError(page, collected, expectedTotal int, totalTrusted bool) error {
	if totalTrusted && expectedTotal > 0 {
		return fmt.Errorf("workflow run jobs pagination made no progress on page %d: collected %d of %d jobs", page, collected, expectedTotal)
	}
	return fmt.Errorf("workflow run jobs pagination made no progress on page %d after collecting %d jobs: total_count was unavailable or inconsistent", page, collected)
}

func jobIdentity(job actions.Job) string {
	if id := strings.TrimSpace(job.ID); id != "" {
		return "id:" + id
	}
	if identifier := strings.TrimSpace(job.Identifier); identifier != "" {
		return "identifier:" + identifier
	}
	return ""
}

func jobPageFingerprint(jobs []actions.Job) string {
	var fingerprint strings.Builder
	for _, job := range jobs {
		fingerprint.WriteString(jobIdentity(job))
		fingerprint.WriteByte(0)
		fingerprint.WriteString(job.Name)
		fingerprint.WriteByte(0)
		fingerprint.WriteString(job.Status)
		fingerprint.WriteByte(0)
		fingerprint.WriteString(strconv.FormatInt(int64(job.StartTime), 10))
		fingerprint.WriteByte('\n')
	}
	return fingerprint.String()
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
	destination, err := preflightDownloadDestination(destination, overwrite)
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
	destination, err = preflightDownloadDestination(destination, overwrite)
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
				formatStepLabel(step),
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

func mergeJobs(primary, fallback []actions.Job) []actions.Job {
	jobs := append([]actions.Job(nil), primary...)
	seen := make(map[string]struct{}, len(primary))
	for _, job := range primary {
		if key := jobIdentity(job); key != "" {
			seen[key] = struct{}{}
		}
	}
	for _, job := range fallback {
		if key := jobIdentity(job); key != "" {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}
		jobs = append(jobs, job)
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
