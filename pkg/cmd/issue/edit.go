package issue

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdIssueEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title          string
		Body           string
		BodyFile       string
		Assignee       string
		RemoveAssignee bool
		Yes            bool
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <number>",
		Short: "Edit an issue",
		Example: `  ag issue edit owner/repo 42 --title "new title" --body "new body"
  ag issue edit owner/repo 42 --assignee alice
  ag issue edit owner/repo 42 --remove-assignee --yes
  ag issue edit owner/repo 42 --body-file description.md`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			titleChanged := cmd.Flags().Changed("title")
			bodyChanged := cmd.Flags().Changed("body")
			bodyFileChanged := cmd.Flags().Changed("body-file")
			assigneeChanged := cmd.Flags().Changed("assignee")
			removeAssigneeChanged := cmd.Flags().Changed("remove-assignee")

			if assigneeChanged && removeAssigneeChanged {
				return fmt.Errorf("--assignee and --remove-assignee cannot be used together")
			}
			if !titleChanged && !bodyChanged && !bodyFileChanged && !assigneeChanged && !removeAssigneeChanged {
				return fmt.Errorf("at least one of --title, --body, --body-file, --assignee, or --remove-assignee must be provided")
			}
			if bodyChanged && bodyFileChanged {
				return fmt.Errorf("--body and --body-file cannot be used together")
			}
			if titleChanged && strings.TrimSpace(opts.Title) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			assignee := strings.TrimSpace(opts.Assignee)
			if assigneeChanged && assignee == "" {
				return fmt.Errorf("assignee cannot be empty")
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]
			if !isPositiveIssueNumber(number) {
				return fmt.Errorf("invalid issue number %q (must be a positive integer)", number)
			}

			if (assigneeChanged || removeAssigneeChanged) && !opts.Yes {
				action := "set assignee to " + assignee
				if removeAssigneeChanged {
					action = "remove assignee"
				}
				prompt := fmt.Sprintf("Change assignee on issue #%s in %s/%s to %s? (y/N) ", number, owner, repo, action)
				confirmed, promptErr := confirmPrompt(cmd, prompt)
				if promptErr != nil {
					return promptErr
				}
				if !confirmed {
					return nil
				}
			}

			body := opts.Body
			if bodyFileChanged {
				content, err := os.ReadFile(opts.BodyFile)
				if err != nil {
					return fmt.Errorf("failed to read body file: %w", err)
				}
				body = string(content)
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}
			client, err := f.NewAPIClient(token)
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
			if assigneeChanged {
				fields["assignee"] = assignee
			}
			if removeAssigneeChanged {
				fields["assignee"] = ""
			}

			if assigneeChanged || removeAssigneeChanged {
				var bodyUpdate *string
				if bodyChanged || bodyFileChanged {
					bodyUpdate = &body
				}
				if err := api.EditIssueAssignee(client, owner, repo, number, title, bodyUpdate, assignee, removeAssigneeChanged); err != nil {
					return fmt.Errorf("failed to edit issue: %w", err)
				}
			} else {
				updatePath := fmt.Sprintf("/repos/%s/issues/%s", owner, number)
				if err := client.PatchForm(updatePath, fields, nil); err != nil {
					return fmt.Errorf("failed to edit issue: %w", err)
				}
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
	cmd.Flags().StringVar(&opts.Assignee, "assignee", "", "Set the issue assignee (login)")
	cmd.Flags().BoolVar(&opts.RemoveAssignee, "remove-assignee", false, "Clear the issue assignee")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func confirmPrompt(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprint(cmd.ErrOrStderr(), prompt)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, nil
	}
	response := strings.TrimSpace(strings.ToLower(line))
	return response == "y" || response == "yes", nil
}
