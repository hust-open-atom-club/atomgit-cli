package comment

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type refreshPositionOptions struct {
	Paths     []string
	CommentID string
	All, JSON bool
}

func newCmdRefreshPosition(f *cmdutil.Factory) *cobra.Command {
	opts := &refreshPositionOptions{}
	cmd := &cobra.Command{
		Use: "refresh-position [<owner>/<repo>] <number>", Short: "Refresh PR comment positions and outdated status",
		Long:    "Refresh comment positions and outdated status for one pull request. Select exactly one scope: repeat --path for files, use --comment-id for one comment, or explicitly use --all. Scopes cannot be combined. File paths are sent literally, without glob expansion. Requests are never automatically retried or redirected.",
		Example: "  ag-cli pr comment refresh-position owner/repo 123 --path src/main.go --path docs/usage.md\n  ag-cli pr comment refresh-position 123 --comment-id 456\n  ag-cli pr comment refresh-position owner/repo 123 --all --json",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			request := api.PullRequestCommentRefreshRequest{Paths: opts.Paths, NoteID: opts.CommentID, All: opts.All}
			if cmd.Flags().Changed("comment-id") && strings.TrimSpace(opts.CommentID) == "" {
				return fmt.Errorf("comment ID must not be empty")
			}
			if err := request.Validate(); err != nil {
				return err
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid PR number: %s", remaining[0])
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			result, err := api.RefreshPullRequestCommentPositions(client.WithContext(cmd.Context()), repository.Owner, repository.Name, number, request)
			if err != nil {
				return fmt.Errorf("failed to refresh comments for PR #%d: %w", number, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "Refreshed comment positions for PR #%d (%d comments).\n", number, len(result.NoteIDs)); err != nil {
				return err
			}
			for _, id := range result.NoteIDs {
				if _, err := fmt.Fprintf(out, "%d\n", id); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&opts.Paths, "path", nil, "File path to refresh (repeat for multiple files; literal paths)")
	cmd.Flags().StringVar(&opts.CommentID, "comment-id", "", "Comment ID to refresh")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Explicitly refresh all comments on the PR")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output refreshed note IDs as JSON")
	cmd.MarkFlagsMutuallyExclusive("path", "comment-id", "all")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Optional owner/repo followed by a positive PR number."},
		Output:      "text,json", Effects: "write", Notes: []string{"Requires exactly one of --path, --comment-id, or --all; scopes cannot be combined.", "No automatic POST retries or redirects. Returned note_ids are integer IDs."},
	})
	return cmd
}
