package issue

import (
	"fmt"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdIssueEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title    string
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/]<repo> <number>",
		Short: "Edit an issue",
		Example: `  ag issue edit owner/repo 42 --title "new title" --body "new body"
  ag issue edit owner/repo 42 --body-file description.md`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			titleChanged := cmd.Flags().Changed("title")
			bodyChanged := cmd.Flags().Changed("body")
			bodyFileChanged := cmd.Flags().Changed("body-file")

			if !titleChanged && !bodyChanged && !bodyFileChanged {
				return fmt.Errorf("at least one of --title, --body, or --body-file must be provided")
			}
			if bodyChanged && bodyFileChanged {
				return fmt.Errorf("--body and --body-file cannot be used together")
			}
			if titleChanged && strings.TrimSpace(opts.Title) == "" {
				return fmt.Errorf("title cannot be empty")
			}

			body := opts.Body
			if bodyFileChanged {
				content, err := os.ReadFile(opts.BodyFile)
				if err != nil {
					return fmt.Errorf("failed to read body file: %w", err)
				}
				body = string(content)
			}

			if len(args) == 1 {
				return fmt.Errorf("repository and issue number required")
			}
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo := parts[0], parts[1]
			number := args[1]

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}
			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			title := opts.Title
			if !titleChanged {
				issuePath := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
				var current api.Issue
				if err := client.Get(issuePath, &current); err != nil {
					return fmt.Errorf("failed to get issue before editing: %w", err)
				}
				title = current.Title
			}

			fields := map[string]string{
				"repo":  repo,
				"title": title,
			}
			if bodyChanged || bodyFileChanged {
				fields["body"] = body
			}

			updatePath := fmt.Sprintf("/repos/%s/issues/%s", owner, number)
			if err := client.PatchForm(updatePath, fields, nil); err != nil {
				return fmt.Errorf("failed to edit issue: %w", err)
			}

			host := strings.TrimRight(strings.TrimSpace(f.Config.GetHost()), "/")
			if host == "" {
				host = "atomgit.com"
			}
			if !strings.Contains(host, "://") {
				host = "https://" + host
			}
			issueURL := fmt.Sprintf("%s/%s/%s/issues/%s", host, owner, repo, number)

			fmt.Fprintf(cmd.OutOrStdout(), "Updated issue #%s: %s\n", number, issueURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New issue title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New issue body")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read the new issue body from a file")

	return cmd
}
