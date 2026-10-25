package org

import (
	"fmt"
	"strconv"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <org> <number>",
		Short: "Delete an organization discussion",
		Long: `Permanently delete an organization discussion by its positive number.

Requires authentication. Confirm with y or yes at the prompt, or pass --yes.
Empty input, EOF, or a negative response cancels without sending a request.
Use --yes for non-interactive deletion. DELETE requests are not automatically
retried or redirected; inspect uncertain results before retrying.`,
		Example: `  ag-cli org discussion delete my-org 7
  ag-cli org discussion delete my-org 7 --yes`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(args[1])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid discussion number %q: must be a positive integer", args[1])
			}
			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Permanently delete discussion #%d from organization %s? This action cannot be undone. [y/N] ", number, organization))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
					return nil
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if err := api.DeleteOrganizationDiscussion(client, organization, number); err != nil {
				return fmt.Errorf("failed to delete discussion #%d for organization %s (not retried; inspect the discussion before retrying): %w", number, organization, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted discussion #%d for organization %s\n", number, organization)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "Explicit organization path and positive discussion number."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and explicit confirmation (y/yes or --yes). Cancellation sends no request.", "Accepts HTTP 200 with an empty body. DELETE is not automatically retried or redirected; inspect uncertain results before retrying."},
	})
	return cmd
}
