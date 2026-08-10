package discussion

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const notAuthenticatedError = "not authenticated: run `ag auth login`"

func NewCmdDiscussion(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "discussion",
		Short:        "View repository discussions",
		Long:         "List AtomGit discussions for a repository",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCmdDiscussionList(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdDiscussionList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List repository discussions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			// respository parse shoule be second execute after limit identity
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil && err.Error() != notAuthenticatedError {
				return err
			}
			if err != nil {
				token = ""
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			items, err := api.GetPaginated[api.Discussion](client, opts.Limit,
				func(page, perPage int) string {
					return fmt.Sprintf("/repos/%s/%s/discuss?page=%d&per_page=%d",
						repository.Owner, repository.Name, page, perPage)
				})

			if err != nil {
				return fmt.Errorf("failed to list discussions for %s: %w", repository, err)
			}
			out := cmd.OutOrStdout()
			if opts.JSON {
				return cmdutil.WriteJSON(out, items)
			}

			if len(items) == 0 {
				fmt.Fprintln(out, "No discussions found.")
				return nil
			}

			w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NUMBER\tTITLE\tCATEGORY\tAUTHOR\tSTATUS\tCOMMENTS\tUPDATED")
			for _, item := range items {
				fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%d\t%s\n",
					item.Number,
					displayDiscussionValue(item.Title),
					displayDiscussionValue(item.Category.Name),
					discussionAuthor(item),
					discussionStatus(item),
					item.CommentTotal,
					displayDiscussionValue(item.UpdatedAt),
				)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of discussions to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output discussions as JSON")
	return cmd
}

func displayDiscussionValue(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	return value
}

func discussionAuthor(item api.Discussion) string {
	if login := displayDiscussionValue(item.Author.Login); login != "-" {
		return login
	}
	return displayDiscussionValue(item.Author.Name)
}

func discussionStatus(item api.Discussion) string {
	status := "OPEN"
	if bool(item.IsClosed) {
		status = "CLOSED"
	} else if bool(item.IsAnswered) {
		status = "ANSWERED"
	}
	if bool(item.IsLocked) {
		status += ",LOCKED"
	}
	if bool(item.IsPinned) {
		status += ",PINNED"
	}
	return status
}
