package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>] <ref>",
		Short: "List comments on a commit",
		Long:  "List comments on a commit, identified by SHA (full or short) or branch name.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			ref, err := cmdutil.ValidateRef(remaining[0], "commit ref")
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			comments, err := api.ListCommitComments(client, owner, repo, ref, opts.Limit)
			if err != nil {
				return err
			}

			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), commentsJSON(comments))
			}

			out := cmd.OutOrStdout()
			if len(comments) == 0 {
				fmt.Fprintln(out, "No comments found.")
				return nil
			}

			currentUser, _ := f.Config.GetUser()
			for _, comment := range comments {
				fmt.Fprintf(out, "%d\t%s\t%s\t%s\n",
					comment.ID,
					cmdutil.EscapeTSVField(comment.User.Login),
					cmdutil.EscapeTSVField(formatCommentTime(comment.CreatedAt)),
					cmdutil.EscapeTSVField(commentBody(comment, currentUser)),
				)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of comments to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output comments as JSON")

	return cmd
}
