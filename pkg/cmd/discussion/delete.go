package discussion

import (
	"fmt"
	"strconv"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDiscussionDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <number>",
		Short: "Delete a repository discussion",
		Long: `Permanently delete a repository discussion by its positive number.

Requires authentication. Confirm with y or yes at the prompt, or pass --yes.
Empty input, EOF, or a negative response cancels without sending a request.
Use --yes for non-interactive deletion. DELETE requests are not automatically
retried or redirected; inspect uncertain results before retrying.`,
		Example: `  ag-cli discussion delete owner/repo 7
  ag-cli discussion delete 7 --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid discussion number %q: must be a positive integer", remaining[0])
			}
			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Permanently delete discussion #%d from %s? This action cannot be undone. [y/N] ", number, repository))
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
			if err := api.DeleteDiscussion(client, repository.Owner, repository.Name, number); err != nil {
				return fmt.Errorf("failed to delete discussion #%d for %s (not retried; inspect the discussion before retrying): %w", number, repository, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted discussion #%d for %s\n", number, repository)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "A positive discussion number, optionally preceded by owner/repo; otherwise use repository context."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and explicit confirmation (y/yes or --yes). Cancellation sends no request.", "Accepts HTTP 200 with an empty body. DELETE is not automatically retried or redirected; inspect uncertain results before retrying."},
	})
	return cmd
}
