package run

import (
	"encoding/json"
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type artifactViewOptions struct {
	JSON bool
}

func newCmdRunArtifact(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Inspect workflow artifacts",
	}
	cmd.AddCommand(newCmdRunArtifactView(f))
	return cmd
}

func newCmdRunArtifactView(f *cmdutil.Factory) *cobra.Command {
	opts := artifactViewOptions{}
	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <artifact-id>",
		Short: "View artifact metadata without downloading the archive",
		Long: `Display AtomGit Actions artifact metadata.

This command does not download the archive. Use ag run view --artifact to
download a zip from a specific workflow run.`,
		Example: `  ag run artifact view owner/repo <artifact-id>
  ag run artifact view <artifact-id> --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArtifactView(cmd, f, opts, args)
		},
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output artifact metadata as JSON")
	return cmd
}

func runArtifactView(cmd *cobra.Command, f *cmdutil.Factory, opts artifactViewOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
	if err != nil {
		return err
	}
	artifactID := strings.TrimSpace(remaining[0])
	if artifactID == "" {
		return fmt.Errorf("artifact ID is required")
	}

	token, err := requireToken(f)
	if err != nil {
		return err
	}
	client, err := newActionsClient(f, token)
	if err != nil {
		return err
	}

	artifact, err := client.GetArtifact(repository.Owner, repository.Name, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get artifact %s for %s/%s: %w", artifactID, repository.Owner, repository.Name, err)
	}

	out := cmd.OutOrStdout()
	if opts.JSON {
		data, err := json.MarshalIndent(artifact, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json output: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintf(out, "ID: %s\n", singleLine(fallback(artifact.ID, artifactID)))
	fmt.Fprintf(out, "Name: %s\n", singleLine(fallback(artifact.Name, "-")))
	fmt.Fprintf(out, "Size: %s\n", formatBytes(artifact.SizeBytes))
	fmt.Fprintf(out, "Workflow: %s\n", singleLine(fallback(artifact.WorkflowID, "-")))
	fmt.Fprintf(out, "Run: %s\n", singleLine(fallback(artifact.WorkflowRunID, "-")))
	fmt.Fprintf(out, "Digest: %s\n", singleLine(fallback(artifact.Digest, "-")))
	fmt.Fprintf(out, "Created: %s\n", formatTimestamp(artifact.CreatedAt))
	fmt.Fprintf(out, "Updated: %s\n", formatTimestamp(artifact.UpdatedAt))
	fmt.Fprintf(out, "Expires: %s\n", formatTimestamp(artifact.ExpiresAt))
	return nil
}
