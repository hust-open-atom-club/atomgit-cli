package comment

import (
	"fmt"
	"strconv"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdResolve(f *cmdutil.Factory, resolved bool) *cobra.Command {
	action, summary, short := "resolve", "Resolved", "Resolve a PR review discussion"
	if !resolved {
		action, summary, short = "unresolve", "Unresolved", "Unresolve a PR review discussion"
	}
	var yes bool
	cmd := &cobra.Command{
		Use:          action + " [<owner>/<repo>] <number> <discussion-id>",
		Short:        short,
		SilenceUsage: true,
		Long:         "Set the resolved state of a PR review discussion using its discussion_id, not its numeric comment/note ID. Get the discussion_id from pr comment view. Requires authentication and confirmation unless --yes is supplied. The PR and review discussion are verified before writing; success is reported only after reading back the requested state. Writes are never automatically retried or redirected.",
		Example:      "  ag-cli pr comment " + action + " owner/repo 123 discussion-id --yes\n  ag-cli pr comment " + action + " 123 discussion-id",
		Args:         cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid PR number: must be a positive integer")
			}
			id := remaining[1]
			if err := api.ValidatePullRequestDiscussionID(id); err != nil {
				return err
			}
			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Set resolved=%t for discussion %s on %s PR #%d? [y/N]: ", resolved, id, repository, number))
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Discussion state update canceled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if _, err := api.GetPullRequestDiscussionState(client, repository.Owner, repository.Name, number, id); err != nil {
				return fmt.Errorf("failed to verify review discussion: %w", err)
			}
			if err := api.SetPullRequestDiscussionResolved(client, repository.Owner, repository.Name, number, id, resolved); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Discussion state update was not confirmed; check AtomGit before retrying.")
				return err
			}
			state, err := api.GetPullRequestDiscussionState(client, repository.Owner, repository.Name, number, id)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "The update was accepted, but its final state could not be verified; check AtomGit before retrying.")
				return fmt.Errorf("failed to read back review discussion: %w", err)
			}
			if *state.Resolved != resolved {
				return fmt.Errorf("discussion state update was accepted, but readback differs from resolved=%t; check AtomGit before retrying", resolved)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s discussion %s on %s PR #%d (verified).\n", summary, id, repository, number)
			return err
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmdutil.AddRepositoryContextHelp(cmd)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 3, Description: "PR number and discussion_id, optionally preceded by owner/repo; repository can be inferred from Git context."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and confirmation unless --yes is supplied. discussion_id is a review thread identifier, not a comment/note ID. The explicit resolved boolean is sent, including false for unresolve. The PR and thread are checked before a single non-retried write, followed by state readback."},
	})
	return cmd
}
