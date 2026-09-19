package comment

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <comment-id> (--body <text> | --body-file <path-or->)",
		Short: "Edit a commit comment",
		Long:  "Edit the body of a commit comment you own, replacing it with the new text.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			commentID, err := validateCommentID(remaining[0])
			if err != nil {
				return err
			}

			bodyChanged := cmd.Flags().Changed("body")
			bodyFileChanged := cmd.Flags().Changed("body-file")
			if !bodyChanged && !bodyFileChanged {
				return fmt.Errorf("comment body is required; use --body or --body-file")
			}

			body, err := cmdutil.ReadBody(opts.Body, opts.BodyFile, bodyChanged, bodyFileChanged, cmd.InOrStdin())
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("comment body cannot be empty")
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			currentUser, err := f.Config.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			existing, err := api.GetCommitComment(client, owner, repo, commentID)
			if err != nil {
				return fmt.Errorf("failed to verify commit comment: %w", err)
			}
			if existing.User.Login != currentUser {
				return fmt.Errorf("只能编辑自己的评论")
			}

			comment, err := api.UpdateCommitComment(client, owner, repo, commentID, body)
			if err != nil {
				return err
			}

			summary := fmt.Sprintf("Updated comment #%s", comment.ID)
			cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), summary, comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read new body text from file (use - for stdin)")

	return cmd
}
