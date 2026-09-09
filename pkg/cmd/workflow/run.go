package workflow

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type runOptions struct {
	Ref       string
	RawFields []string
	Fields    []string
}

func newCmdRun(f *cmdutil.Factory) *cobra.Command {
	opts := &runOptions{}

	cmd := &cobra.Command{
		Use:     "run [<owner>/<repo>] <workflow_id>",
		Aliases: []string{"dispatch"},
		Short:   "Run a workflow",
		Long:    `Manually trigger an AtomGit Actions workflow run (workflow_dispatch).`,
		Example: `  ag workflow run owner/repo 12345 --ref main
  ag workflow run owner/repo ci.yml --ref feature-branch -f env=prod -f debug=true`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := requireToken(f)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}

			workflowTarget := strings.TrimSpace(remaining[0])
			if workflowTarget == "" {
				return fmt.Errorf("workflow_id is required")
			}

			inputs, err := parseInputFields(opts)
			if err != nil {
				return err
			}

			client, err := newActionsClient(f, token)
			if err != nil {
				return err
			}

			ref := strings.TrimSpace(opts.Ref)
			if ref == "" {
				ref, err = resolveDefaultBranch(f, token, repository.Owner, repository.Name)
				if err != nil {
					return fmt.Errorf("could not determine the repository's default branch; pass --ref explicitly: %w", err)
				}
			}

			workflowID, err := resolveWorkflowID(client, repository.Owner, repository.Name, workflowTarget)
			if err != nil {
				return err
			}

			payload := actions.WorkflowDispatchPayload{
				Ref:    ref,
				Inputs: inputs,
			}

			if err := client.CreateWorkflowDispatch(repository.Owner, repository.Name, workflowID, payload); err != nil {
				return fmt.Errorf("failed to trigger workflow %q for %s/%s: %w", workflowTarget, repository.Owner, repository.Name, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Triggered workflow %q (%s) on ref %q\n", workflowTarget, workflowID, ref)
			fmt.Fprintf(cmd.OutOrStdout(), "View runs with: ag run list %s/%s\n", repository.Owner, repository.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Ref, "ref", "r", "", "The git reference (branch or tag) to run the workflow on")
	cmd.Flags().StringArrayVarP(&opts.RawFields, "raw-field", "f", nil, "Add a string parameter in key=value format")
	cmd.Flags().StringArrayVarP(&opts.Fields, "field", "F", nil, "Add a string parameter in key=value format")

	return cmd
}

func parseInputFields(opts *runOptions) (map[string]string, error) {
	inputs := make(map[string]string)
	allFields := append(opts.RawFields, opts.Fields...)
	for _, field := range allFields {
		key, val, ok := strings.Cut(field, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid field format %q (expected key=value)", field)
		}
		inputs[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return inputs, nil
}

func resolveWorkflowID(client *actions.Client, owner, repo, target string) (string, error) {
	workflows, err := listAllWorkflows(client, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to list workflows for %s/%s: %w", owner, repo, err)
	}

	for _, wf := range workflows {
		if wf.ID == target {
			return wf.ID, nil
		}
	}

	var matches []actions.Workflow
	for _, wf := range workflows {
		if wf.Name == target || wf.Path == target || strings.HasSuffix(wf.Path, "/"+target) {
			matches = append(matches, wf)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0].ID, nil
	case 0:
		return target, nil
	default:
		descriptions := make([]string, 0, len(matches))
		for _, wf := range matches {
			descriptions = append(descriptions, fmt.Sprintf("%q (%s)", wf.Name, wf.Path))
		}
		return "", fmt.Errorf("workflow target %q is ambiguous; it matches multiple workflows: %s (use the exact workflow ID or full path)", target, strings.Join(descriptions, ", "))
	}
}
