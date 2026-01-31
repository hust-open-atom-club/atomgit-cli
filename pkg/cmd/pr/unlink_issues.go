package pr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shinwell/ag-cli/internal/api"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdUnlinkIssues(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Issues []string
	}

	cmd := &cobra.Command{
		Use:   "unlink-issues [<owner>/]<repo> <pr_number>",
		Short: "Unlink issues from a pull request",
		Long:  `Unlink one or more issues from a pull request.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var prNumber string

			if len(args) == 1 {
				return fmt.Errorf("repository and PR number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			prNumber = args[1]

			if len(opts.Issues) == 0 {
				return fmt.Errorf("at least one issue number is required (--issue)")
			}

			client := api.NewClient(token)

			// Unlink each issue
			unlinkedIssues := []string{}
			for _, issueNumber := range opts.Issues {
				// Validate issue number
				if _, err := strconv.Atoi(issueNumber); err != nil {
					return fmt.Errorf("invalid issue number: %s", issueNumber)
				}

				body := map[string]string{
					"issue_number": issueNumber,
				}

				path := fmt.Sprintf("/repos/%s/%s/pulls/%s/issues", owner, repo, prNumber)
				if err := client.DeleteWithBody(path, body); err != nil {
					return fmt.Errorf("failed to unlink issue #%s: %w", issueNumber, err)
				}

				unlinkedIssues = append(unlinkedIssues, issueNumber)
			}

			// Output result
			if len(unlinkedIssues) == 1 {
				fmt.Printf("Unlinked issue #%s from PR #%s\n", unlinkedIssues[0], prNumber)
			} else {
				fmt.Printf("Unlinked issues #%s from PR #%s\n", strings.Join(unlinkedIssues, ", #"), prNumber)
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.Issues, "issue", "i", nil, "Issue number to unlink (can be specified multiple times)")

	return cmd
}
