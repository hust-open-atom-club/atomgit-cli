package pr

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdViewIssues(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "issues [<owner>/]<repo> <pr_number>",
		Short: "View linked issues of a pull request",
		Long:  `View all issues linked to a pull request.`,
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

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			// Get linked issues using GET method
			var issues []api.Issue
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s/issues", owner, repo, prNumber)
			if err := client.Get(path, &issues); err != nil {
				return fmt.Errorf("failed to get linked issues: %w", err)
			}

			out := cmd.OutOrStdout()
			if len(issues) == 0 {
				fmt.Fprintf(out, "PR #%s has no linked issues\n", prNumber)
				return nil
			}

			fmt.Fprintf(out, "PR #%s linked issues:\n", prNumber)
			for _, issue := range issues {
				fmt.Fprintf(out, "  #%s %s [%s]\n", issue.GetNumber(), issue.Title, issue.State)
			}

			return nil
		},
	}
}
