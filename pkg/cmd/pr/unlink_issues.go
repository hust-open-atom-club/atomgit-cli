package pr

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdUnlinkIssues(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Issues []string
	}

	cmd := &cobra.Command{
		Use:   "unlink-issues [<owner>/<repo>] <pr_number>",
		Short: "Unlink issues from a pull request",
		Long:  `Unlink one or more issues from a pull request.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			prNumber := remaining[0]

			if len(opts.Issues) == 0 {
				return fmt.Errorf("at least one issue number is required (--issue)")
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			// Convert issue numbers to integers
			issueNumbers := []int{}
			for _, issueNumber := range opts.Issues {
				num, err := strconv.Atoi(issueNumber)
				if err != nil {
					return fmt.Errorf("invalid issue number: %s", issueNumber)
				}
				issueNumbers = append(issueNumbers, num)
			}

			// Unlink issues using array format
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s/issues?", owner, repo, prNumber)
			if err := client.DeleteWithBody(path, issueNumbers); err != nil {
				return fmt.Errorf("failed to unlink issues: %w", err)
			}

			unlinkedIssues := opts.Issues

			// Output result
			out := cmd.OutOrStdout()
			if len(unlinkedIssues) == 1 {
				fmt.Fprintf(out, "Unlinked issue #%s from PR #%s\n", unlinkedIssues[0], prNumber)
			} else {
				fmt.Fprintf(out, "Unlinked issues #%s from PR #%s\n", strings.Join(unlinkedIssues, ", #"), prNumber)
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.Issues, "issue", "i", nil, "Issue number to unlink (can be specified multiple times)")

	return cmd
}
