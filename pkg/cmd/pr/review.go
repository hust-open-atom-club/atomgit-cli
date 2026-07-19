package pr

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type reviewOptions struct {
	Approve        bool
	RequestChanges bool
	Comment        bool
	Body           string
	BodyFile       string
	Editor         bool
}

type reviewEditor func(*cobra.Command) (string, error)

func newCmdPRReview(f *cmdutil.Factory) *cobra.Command {
	return newCmdPRReviewWithEditor(f, func(cmd *cobra.Command) (string, error) {
		return cmdutil.EditText(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
	})
}

func newCmdPRReviewWithEditor(f *cmdutil.Factory, editor reviewEditor) *cobra.Command {
	opts := &reviewOptions{}

	cmd := &cobra.Command{
		Use:   "review <owner>/<repo> <number>",
		Short: "Submit a pull request review",
		Long: `Submit a formal review that approves a pull request, requests
changes, or leaves a review comment. This is separate from ag pr comment.

Approval bodies are optional. Request-changes and comment reviews require a
non-empty body from --body, --body-file, or --editor.`,
		Example: `  ag pr review owner/repo 42 --approve
  ag pr review owner/repo 42 --request-changes --body "Please add tests."
  ag pr review owner/repo 42 --comment --body-file review.md
  ag pr review owner/repo 42 --comment --editor`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := reviewEvent(opts)
			if err != nil {
				return err
			}

			body, err := resolveReviewBody(cmd, opts, editor)
			if err != nil {
				return err
			}
			if reviewBodyRequired(event) && strings.TrimSpace(body) == "" {
				return fmt.Errorf("a review body is required for --%s", reviewModeFlag(event))
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
			if author := strings.TrimSpace(pullRequest.User.Login); author != "" && strings.EqualFold(strings.TrimSpace(currentUser), author) {
				return fmt.Errorf("cannot review your own pull request")
			}

			request := api.PullRequestReviewRequest{Body: body, Event: event}
			var review api.PullRequestReview
			if err := client.Post(path+"/review", request, &review); err != nil {
				return fmt.Errorf("failed to submit review: %w", err)
			}

			url := strings.TrimSpace(review.HTMLURL)
			if url == "" {
				url = pullRequest.HTMLURL
			}
			cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), reviewSummary(event, number), url)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.Approve, "approve", false, "Approve the pull request")
	cmd.Flags().BoolVar(&opts.RequestChanges, "request-changes", false, "Request changes to the pull request")
	cmd.Flags().BoolVar(&opts.Comment, "comment", false, "Submit a review comment without approval")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Review body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read the review body from a file (use - for standard input)")
	cmd.Flags().BoolVar(&opts.Editor, "editor", false, "Open an editor to write the review body")

	return cmd
}

func reviewEvent(opts *reviewOptions) (api.PullRequestReviewEvent, error) {
	selected := 0
	if opts.Approve {
		selected++
	}
	if opts.RequestChanges {
		selected++
	}
	if opts.Comment {
		selected++
	}
	if selected != 1 {
		return "", fmt.Errorf("exactly one of --approve, --request-changes, or --comment must be specified")
	}

	switch {
	case opts.Approve:
		return api.PullRequestReviewApprove, nil
	case opts.RequestChanges:
		return api.PullRequestReviewRequestChanges, nil
	default:
		return api.PullRequestReviewComment, nil
	}
}

func resolveReviewBody(cmd *cobra.Command, opts *reviewOptions, editor reviewEditor) (string, error) {
	bodyChanged := cmd.Flags().Changed("body")
	bodyFileChanged := cmd.Flags().Changed("body-file")
	sources := 0
	if bodyChanged {
		sources++
	}
	if bodyFileChanged {
		sources++
	}
	if opts.Editor {
		sources++
	}
	if sources > 1 {
		return "", fmt.Errorf("only one of --body, --body-file, or --editor may be used")
	}

	switch {
	case bodyChanged:
		return opts.Body, nil
	case bodyFileChanged && opts.BodyFile == "-":
		content, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("failed to read review body from standard input: %w", err)
		}
		return string(content), nil
	case bodyFileChanged:
		content, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read review body file: %w", err)
		}
		return string(content), nil
	case opts.Editor:
		if editor == nil {
			return "", fmt.Errorf("editor is unavailable")
		}
		body, err := editor(cmd)
		if err != nil {
			return "", err
		}
		return body, nil
	default:
		return "", nil
	}
}

func reviewBodyRequired(event api.PullRequestReviewEvent) bool {
	return event == api.PullRequestReviewRequestChanges || event == api.PullRequestReviewComment
}

func reviewModeFlag(event api.PullRequestReviewEvent) string {
	switch event {
	case api.PullRequestReviewRequestChanges:
		return "request-changes"
	case api.PullRequestReviewComment:
		return "comment"
	default:
		return "approve"
	}
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

func reviewSummary(event api.PullRequestReviewEvent, number int) string {
	switch event {
	case api.PullRequestReviewApprove:
		return fmt.Sprintf("Approved PR #%d", number)
	case api.PullRequestReviewRequestChanges:
		return fmt.Sprintf("Requested changes on PR #%d", number)
	default:
		return fmt.Sprintf("Submitted review comment on PR #%d", number)
	}
}
