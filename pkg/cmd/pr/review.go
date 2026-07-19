package pr

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type reviewOptions struct {
	Approve bool
	Force   bool
}

func newCmdPRReview(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewOptions{}

	cmd := &cobra.Command{
		Use:   "review <owner>/<repo> <number>",
		Short: "Approve a pull request review",
		Long: `Approve a pull request using AtomGit's formal review API.

AtomGit currently exposes approval as the only review action. Use ag pr comment
create to leave an ordinary comment; request-changes reviews are not supported
by the public API.`,
		Example: `  ag pr review owner/repo 42 --approve
  ag pr review owner/repo 42 --approve --force`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.Approve {
				return fmt.Errorf("--approve is required; AtomGit does not support other review actions")
			}

			owner, repo, number, err := parseReviewArgs(args)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}
			currentUser, err := f.Config.GetUser()
			if err != nil {
				return fmt.Errorf("failed to determine current user: %w", err)
			}
			currentUser = strings.TrimSpace(currentUser)
			if currentUser == "" {
				return fmt.Errorf("current user is empty in stored credentials")
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
			var pullRequest api.PullRequest
			if err := client.Get(path, &pullRequest); err != nil {
				return fmt.Errorf("failed to get PR #%d: %w", number, err)
			}
			if !strings.EqualFold(strings.TrimSpace(pullRequest.State), "open") {
				state := strings.TrimSpace(pullRequest.State)
				if state == "" {
					state = "unknown"
				}
				return fmt.Errorf("cannot review PR #%d because it is %s", number, state)
			}
			if author := strings.TrimSpace(pullRequest.User.Login); author != "" && strings.EqualFold(currentUser, author) {
				return fmt.Errorf("cannot review your own pull request")
			}

			request := api.PullRequestReviewRequest{Force: opts.Force}
			if err := client.Post(path+"/review", request, nil); err != nil {
				return fmt.Errorf("failed to approve PR: %w", err)
			}

			summary := fmt.Sprintf("Approved PR #%d", number)
			if opts.Force {
				summary = fmt.Sprintf("Force-approved PR #%d", number)
			}
			cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), summary, pullRequest.HTMLURL)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.Approve, "approve", false, "Approve the pull request")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force approval as a repository administrator")

	return cmd
}

func parseReviewArgs(args []string) (string, string, int, error) {
	parts := strings.Split(args[0], "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", 0, fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
	}

	number, err := strconv.Atoi(args[1])
	if err != nil || number <= 0 {
		return "", "", 0, fmt.Errorf("invalid PR number: %s (must be positive)", args[1])
	}
	return parts[0], parts[1], number, nil
}
