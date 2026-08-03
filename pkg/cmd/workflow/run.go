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

			client, err := newActionsClient(f, token)
			if err != nil {
				return err
			}

			ref := strings.TrimSpace(opts.Ref)
			if ref == "" {
				ref = "main"
			}

			workflowID := workflowTarget
			workflowsRes, err := client.ListWorkflows(repository.Owner, repository.Name)
			if err == nil && len(workflowsRes.Workflows) > 0 {
				for _, wf := range workflowsRes.Workflows {
					if wf.ID == workflowTarget || wf.Name == workflowTarget || wf.Path == workflowTarget || strings.HasSuffix(wf.Path, "/"+workflowTarget) {
						workflowID = wf.ID
						break
					}
				}
			}

			inputs := make(map[string]string)
			allFields := append(opts.RawFields, opts.Fields...)
			for _, field := range allFields {
				key, val, ok := strings.Cut(field, "=")
				if !ok || strings.TrimSpace(key) == "" {
					return fmt.Errorf("invalid field format %q (expected key=value)", field)
				}
				inputs[strings.TrimSpace(key)] = strings.TrimSpace(val)
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
