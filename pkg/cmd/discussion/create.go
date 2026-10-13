package discussion

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDiscussionCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct{ Title, Category, Body, BodyFile string }
	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>]",
		Short: "Create a repository discussion",
		Long: `Create a repository discussion with a title, category name, and Markdown body.

Title, category, and body must not be empty or whitespace-only. Use the category
name without its display emoji or ID. Provide exactly one of --body or --body-file;
use --body-file - to read stdin. Body text is sent without trimming.

Requires authentication. POST requests are not automatically retried or redirected.
If the result is uncertain, inspect the discussion list before retrying.`,
		Example: `  ag-cli discussion create owner/repo --title "Roadmap" --category "Ideas" --body "Next steps"
  ag-cli discussion create --title "Roadmap" --category "Ideas" --body-file discussion.md
  ag-cli discussion create --title "Roadmap" --category "Ideas" --body-file -`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.Title) == "" {
				return fmt.Errorf("discussion title cannot be empty or whitespace-only")
			}
			if strings.TrimSpace(opts.Category) == "" {
				return fmt.Errorf("discussion category cannot be empty or whitespace-only")
			}
			body, err := cmdutil.ReadBody(opts.Body, opts.BodyFile, cmd.Flags().Changed("body"), cmd.Flags().Changed("body-file"), cmd.InOrStdin())
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("discussion body cannot be empty or whitespace-only")
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			created, err := api.CreateDiscussion(client, repository.Owner, repository.Name, api.DiscussionCreateRequest{Title: opts.Title, MDContent: body, CategoryName: opts.Category})
			if err != nil {
				return fmt.Errorf("failed to create discussion for %s (not retried; inspect the discussion list before retrying): %w", repository, err)
			}
			if created.Number <= 0 {
				return fmt.Errorf("cannot confirm created discussion for %s: response has no positive discussion number; creation may have succeeded, inspect the discussion list before retrying", repository)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created discussion #%d for %s\n", created.Number, repository)
			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Discussion title (required, cannot be blank)")
	cmd.Flags().StringVar(&opts.Category, "category", "", "Category name without a display emoji or ID (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Markdown body (cannot be blank)")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read the Markdown body from file (use - for stdin)")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.MarkFlagsOneRequired("body", "body-file")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals:   &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; otherwise use repository context."},
		RequiredFlags: []string{"title", "category"}, Output: "text", Effects: "write",
		Notes: []string{"Requires authentication and nonblank title, category, and body. Body text from a flag, file, or stdin is preserved without trimming.", "POST is not automatically retried or redirected. Success requires a positive discussion number from the response; inspect uncertain results before retrying."},
	})
	return cmd
}
