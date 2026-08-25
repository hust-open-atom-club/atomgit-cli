package run

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type artifactViewOptions struct {
	JSON bool
}

type artifactDeleteOptions struct {
	Yes bool
}

func newCmdRunArtifact(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Inspect and manage workflow artifacts",
	}
	cmd.AddCommand(newCmdRunArtifactView(f))
	cmd.AddCommand(newCmdRunArtifactDelete(f))
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

func newCmdRunArtifactDelete(f *cmdutil.Factory) *cobra.Command {
	opts := artifactDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <artifact-id>",
		Short: "Delete a workflow artifact",
		Long: `Delete an AtomGit Actions artifact after reading its metadata.

By default, the command displays the artifact details and asks for
confirmation. Deletion cannot be undone. Use --yes to skip the prompt.`,
		Example: `  ag run artifact delete owner/repo <artifact-id>
  ag run artifact delete <artifact-id> --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArtifactDelete(cmd, f, opts, args)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip deletion confirmation")
	return cmd
}

func runArtifactDelete(cmd *cobra.Command, f *cmdutil.Factory, opts artifactDeleteOptions, args []string) error {
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
		return fmt.Errorf("failed to get artifact %s for %s/%s: %w", singleLine(artifactID), repository.Owner, repository.Name, err)
	}

	if !opts.Yes {
		confirmed, err := confirmArtifactDelete(cmd, repository, artifactID, artifact)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Artifact deletion cancelled")
			return nil
		}
	}

	if err := client.DeleteArtifact(repository.Owner, repository.Name, artifactID); err != nil {
		return fmt.Errorf("failed to delete artifact %s from %s/%s: %w", singleLine(artifactID), repository.Owner, repository.Name, err)
	}
	name := singleLine(fallback(artifact.Name, "unnamed"))
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted artifact %s (%s) from %s/%s\n", name, singleLine(artifactID), repository.Owner, repository.Name)
	return nil
}

func confirmArtifactDelete(cmd *cobra.Command, repository cmdutil.Repository, artifactID string, artifact actions.Artifact) (bool, error) {
	repositoryName := singleLine(repository.Owner + "/" + repository.Name)
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), `Repository: %s
Artifact ID: %s
Name: %s
Workflow run: %s
Expires: %s
Deleting this artifact cannot be undone.
Delete this artifact? [y/N] `,
		repositoryName,
		singleLine(artifactID),
		singleLine(fallback(artifact.Name, "-")),
		singleLine(fallback(artifact.WorkflowRunID, "-")),
		formatTimestamp(artifact.ExpiresAt),
	); err != nil {
		return false, fmt.Errorf("write deletion confirmation: %w", err)
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read deletion confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
