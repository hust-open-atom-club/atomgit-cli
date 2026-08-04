package pr

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdLinkIssues(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Issues []string
	}

	cmd := &cobra.Command{
		Use:   "link-issues [<owner>/<repo>] <pr_number>",
		Short: "Link issues to a pull request",
		Long:  `Link one or more issues to a pull request.`,
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

			// Link issues using array format
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s/issues?", owner, repo, prNumber)
			if err := client.Post(path, issueNumbers, nil); err != nil {
				return fmt.Errorf("failed to link issues: %w", err)
			}

			linkedIssues := opts.Issues

			// Output result
			out := cmd.OutOrStdout()
			if len(linkedIssues) == 1 {
				fmt.Fprintf(out, "Linked issue #%s to PR #%s\n", linkedIssues[0], prNumber)
			} else {
				fmt.Fprintf(out, "Linked issues #%s to PR #%s\n", strings.Join(linkedIssues, ", #"), prNumber)
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.Issues, "issue", "i", nil, "Issue number to link (can be specified multiple times)")

	return cmd
}
