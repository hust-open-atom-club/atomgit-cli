package discussion

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDiscussionEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title    string
		Body     string
		BodyFile string
		Category string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <number>",
		Short: "Edit a repository discussion",
		Long: `Edit the title, Markdown body, or category of a repository discussion.

At least one edit flag is required. Unspecified fields retain their current
values. Titles and category names cannot be empty or whitespace-only. Use the
category name alone, without its display emoji or ID. --body and --body-file
are mutually exclusive; an empty body, file, or stdin requests clearing the
body, subject to server validation. Body text is sent without trimming.

The command requires authentication and verifies the update response, reading
the discussion back when necessary. PUT requests are never automatically
retried; inspect the discussion before retrying an uncertain result.`,
		Example: `  ag-cli discussion edit 6 --title "Updated title"
  ag-cli discussion edit owner/repo 6 --body "Updated **Markdown**"
  ag-cli discussion edit owner/repo 6 --body-file discussion.md
  ag-cli discussion edit 6 --body-file -
  ag-cli discussion edit owner/repo 6 --title "Roadmap" --category "Ideas"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid discussion number %q: must be a positive integer", remaining[0])
			}

			titleChanged := cmd.Flags().Changed("title")
			categoryChanged := cmd.Flags().Changed("category")
			bodyChanged := cmd.Flags().Changed("body")
			bodyFileChanged := cmd.Flags().Changed("body-file")
			if !titleChanged && !categoryChanged && !bodyChanged && !bodyFileChanged {
				return fmt.Errorf("at least one of --title, --body, --body-file, or --category is required")
			}
			if titleChanged && strings.TrimSpace(opts.Title) == "" {
				return fmt.Errorf("discussion title cannot be empty or whitespace-only")
			}
			if categoryChanged && strings.TrimSpace(opts.Category) == "" {
				return fmt.Errorf("discussion category cannot be empty or whitespace-only")
			}
			body, err := cmdutil.ReadBody(opts.Body, opts.BodyFile, bodyChanged, bodyFileChanged, cmd.InOrStdin())
			if err != nil {
				return err
			}

			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			current, err := api.GetDiscussion(client, repository.Owner, repository.Name, number)
			if err != nil {
				return fmt.Errorf("failed to read discussion #%d for %s before editing: %w", number, repository, err)
			}
			if current.ID == "" || current.Number != number || !current.MDContentPresent ||
				strings.TrimSpace(current.Title) == "" || strings.TrimSpace(current.Category.Name) == "" {
				return fmt.Errorf("cannot edit discussion #%d for %s: incomplete or unexpected current discussion; no update sent", number, repository)
			}
			request := api.DiscussionEditRequest{
				Title: current.Title, MDContent: current.MDContent, CategoryName: current.Category.Name,
			}
			if titleChanged {
				request.Title = opts.Title
			}
			if bodyChanged || bodyFileChanged {
				request.MDContent = body
			}
			if categoryChanged {
				request.CategoryName = opts.Category
			}

			updated, err := api.UpdateDiscussion(client, repository.Owner, repository.Name, number, request)
			if err != nil {
				return fmt.Errorf("failed to update discussion #%d for %s (not retried; inspect the discussion before retrying): %w", number, repository, err)
			}
			if !discussionEditMatches(updated, current, request) {
				updated, err = api.GetDiscussion(client, repository.Owner, repository.Name, number)
				if err != nil {
					return fmt.Errorf("failed to verify discussion #%d for %s; the update may have been applied (inspect before retrying): %w", number, repository, err)
				}
				if !discussionEditMatches(updated, current, request) {
					return fmt.Errorf("could not verify update to discussion #%d for %s: returned fields do not match the requested values; inspect before retrying", number, repository)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated discussion #%d for %s\n", number, repository)
			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New discussion title (cannot be blank)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New Markdown body (empty requests clearing it)")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read the Markdown body from file (use - for stdin)")
	cmd.Flags().StringVar(&opts.Category, "category", "", "New category name without a display emoji or ID (cannot be blank)")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.MarkFlagsOneRequired("title", "body", "body-file", "category")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "A positive discussion number, optionally preceded by owner/repo; otherwise use repository context."},
		Output:      "text", Effects: "write",
		Notes: []string{
			"Requires authentication. Unspecified title, Markdown body, and category name are preserved; other discussion state and comments are not edited.",
			"--body-file reads a file or command stdin with -. Body text is not trimmed; explicit empty bodies request clearing, subject to server validation. Blank titles and categories are rejected.",
			"Verifies returned fields and reads back when necessary. PUT is not automatically retried; inspect uncertain results before retrying.",
		},
	})
	return cmd
}

func discussionEditMatches(updated, current api.DiscussionDetail, request api.DiscussionEditRequest) bool {
	return updated.ID == current.ID && updated.Number == current.Number && updated.MDContentPresent &&
		updated.Title == request.Title && updated.MDContent == request.MDContent && updated.Category.Name == request.CategoryName
}
