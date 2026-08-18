package workflow

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	File string
	JSON bool
}

func newCmdValidate(f *cmdutil.Factory) *cobra.Command {
	opts := &validateOptions{}
	cmd := &cobra.Command{
		Use:   "validate [<owner>/<repo>]",
		Short: "Validate a local workflow YAML file",
		Long: `Validate a local AtomGit Actions workflow file against the documented v8 endpoint.

The file is not modified. HTTP 200 with valid=false is treated as a command
error so CI can fail on invalid YAML.`,
		Example: `  ag workflow validate --file .gitcode/workflows/ci.yml
  ag workflow validate owner/repo --file workflow.yml
  ag workflow validate owner/repo --file workflow.yml --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVar(&opts.File, "file", "", "Path to a local workflow YAML file")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the validation response as JSON")
	return cmd
}

func runValidate(cmd *cobra.Command, f *cmdutil.Factory, opts *validateOptions, args []string) error {
	filePath := strings.TrimSpace(opts.File)
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read workflow file %s: %w", filePath, err)
	}

	token, err := requireToken(f)
	if err != nil {
		return err
	}

	repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
	if err != nil {
		return err
	}

	client, err := newActionsClient(f, token)
	if err != nil {
		return err
	}

	result, err := client.ValidateWorkflow(repository.Owner, repository.Name, actions.WorkflowValidationRequest{
		Base64Content: base64.StdEncoding.EncodeToString(contents),
	})
	if err != nil {
		return fmt.Errorf("failed to validate workflow for %s/%s: %w", repository.Owner, repository.Name, err)
	}

	out := cmd.OutOrStdout()
	if opts.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json output: %w", err)
		}
		fmt.Fprintln(out, string(data))
	} else if result.Valid {
		fmt.Fprintln(out, "Workflow is valid.")
	} else {
		fmt.Fprintln(out, "Workflow is invalid:")
		if len(result.Diagnostics) == 0 {
			fmt.Fprintln(out, "  (no diagnostics returned)")
		}
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprintf(out, "  %s\n", formatDiagnostic(diagnostic))
		}
	}

	if !result.Valid {
		return fmt.Errorf("workflow is invalid")
	}
	return nil
}

func formatDiagnostic(diagnostic actions.Diagnostic) string {
	severity := strings.TrimSpace(diagnostic.Severity)
	if severity == "" {
		severity = "Error"
	}
	message := strings.TrimSpace(diagnostic.Message)
	if message == "" {
		message = "validation failed"
	}
	start := diagnostic.Range.Start
	if start.Line == 0 && start.Column == 0 {
		return fmt.Sprintf("%s: %s", severity, message)
	}
	return fmt.Sprintf("%s: line %d, column %d: %s", severity, start.Line, start.Column, message)
}
