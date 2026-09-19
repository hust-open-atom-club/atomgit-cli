package comment

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <sha> (--body <text> | --body-file <path-or->)",
		Short: "Create a comment on a commit",
		Long:  "Create a comment on a commit, identified by SHA (full or short) or branch name.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			sha, err := cmdutil.ValidateRef(remaining[0], "commit SHA")
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

			comment, err := api.CreateCommitComment(client, owner, repo, sha, body)
			if err != nil {
				return err
			}

			commentID := comment.GetID()
			if commentID == "" {
				return fmt.Errorf("created comment response did not include a comment ID")
			}
			commentURL := cmdutil.ResolveWebURL(comment.GetURL(), f.Config.GetHost(), owner, repo, "commits", "detail", sha)
			summary := fmt.Sprintf("Created comment #%s", commentID)
			cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), summary, commentURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use - for stdin)")

	return cmd
}
