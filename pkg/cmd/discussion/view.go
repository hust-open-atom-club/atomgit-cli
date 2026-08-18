package discussion

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// commentThread pairs a comment with the replies collected for it. Replies
// live on a separate endpoint, so the API type itself stays transport-only.
type commentThread struct {
	Comment api.DiscussionComment
	Replies []api.DiscussionComment
}

func newCmdDiscussionView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		comments bool
		jsonOut  bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <number>",
		Short: "View a repository discussion",
		Long: `View a single repository discussion with its Markdown body.

With --comments the comment thread is fetched as well; comments that have
replies include their nested replies in server order. Hidden, deleted, and
empty bodies are shown as explicit placeholders instead of blank text.`,
		Example: `  ag discussion view owner/repo 1
  ag discussion view owner/repo 1 --comments
  ag discussion view owner/repo 1 --comments --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}

			// The discussion number is validated before any authentication or
			// HTTP client initialization (issue #49 ordering).
			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid discussion number %q: must be a positive integer", remaining[0])
			}

			// Discussions are public content, so a missing token falls back to
			// an unauthenticated request (same as discussion list).
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

			detail, err := api.GetDiscussion(client, repository.Owner, repository.Name, number)
			if err != nil {
				return fmt.Errorf("failed to view discussion #%d for %s: %w", number, repository, err)
			}

			var threads []commentThread
			if opts.comments {
				comments, err := api.ListDiscussionComments(client, repository.Owner, repository.Name, number, detail.CommentTotal)
				if err != nil {
					return fmt.Errorf("failed to list comments of discussion #%d for %s: %w", number, repository, err)
				}
				threads = make([]commentThread, 0, len(comments))
				for _, comment := range comments {
					thread := commentThread{Comment: comment}
					if comment.ReplyTotal > 0 {
						replies, err := api.ListDiscussionReplies(client, repository.Owner, repository.Name, number, comment.ID, comment.ReplyTotal)
						if err != nil {
							return fmt.Errorf("failed to list replies of comment %s on discussion #%d: %w", comment.ID, number, err)
						}
						thread.Replies = replies
					}
					threads = append(threads, thread)
				}
			}

			if opts.jsonOut {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), buildDiscussionViewJSON(detail, threads))
			}
			printDiscussionView(cmd.OutOrStdout(), detail, threads)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.comments, "comments", false, "Also fetch and show the comment thread")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output the discussion as JSON")

	return cmd
}

// discussionBody picks the Markdown body when present and falls back to the
// plain-text content; hidden and deleted entries keep explicit placeholders so
// missing content is never presented as a normal body.
func discussionBody(comment api.DiscussionComment) string {
	if bool(comment.IsDeleted) {
		return "[deleted]"
	}
	if bool(comment.IsHidden) {
		return "[hidden]"
	}
	if strings.TrimSpace(comment.MDContent) != "" {
		return comment.MDContent
	}
	if strings.TrimSpace(comment.Content) != "" {
		return comment.Content
	}
	return "[no content]"
}

func printDiscussionView(out io.Writer, detail api.DiscussionDetail, threads []commentThread) {
	fmt.Fprintf(out, "#%d\t%s\n", detail.Number, detail.Title)
	fmt.Fprintf(out, "Author:\t%s\n", discussionAuthor(detail.Discussion))
	fmt.Fprintf(out, "Category:\t%s\n", displayDiscussionValue(detail.Category.Icon+" "+detail.Category.Name))
	fmt.Fprintf(out, "Status:\t%s\n", discussionStatus(detail.Discussion))
	fmt.Fprintf(out, "Created:\t%s\n", displayDiscussionValue(detail.CreatedAt))
	fmt.Fprintf(out, "Updated:\t%s\n", displayDiscussionValue(detail.UpdatedAt))
	fmt.Fprintf(out, "Comments:\t%d\n", detail.CommentTotal)
	fmt.Fprintln(out, strings.Repeat("-", 40))
	body := strings.TrimSpace(detail.MDContent)
	if body == "" {
		body = "[no content]"
	}
	fmt.Fprintln(out, body)

	for _, thread := range threads {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "* %s\t%s\n", authorLabel(thread.Comment.Author), displayDiscussionValue(thread.Comment.CreatedAt))
		for _, line := range strings.Split(discussionBody(thread.Comment), "\n") {
			fmt.Fprintf(out, "  %s\n", line)
		}
		for _, reply := range thread.Replies {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  * %s\t%s\n", authorLabel(reply.Author), displayDiscussionValue(reply.CreatedAt))
			for _, line := range strings.Split(discussionBody(reply), "\n") {
				fmt.Fprintf(out, "    %s\n", line)
			}
		}
	}
}

// authorLabel renders a discussion author, preferring the login over the
// display name.
func authorLabel(author api.DiscussionAuthor) string {
	if login := displayDiscussionValue(author.Login); login != "-" {
		return login
	}
	return displayDiscussionValue(author.Name)
}

type discussionViewJSON struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Author       string `json:"author"`
	Category     string `json:"category"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	CommentTotal int    `json:"commentTotal"`
	Body         string `json:"body"`
	// Comments is nil (key omitted) when the thread was not requested, and a
	// pointer to an empty slice (rendered as []) when it was requested but
	// empty — omitempty alone cannot tell those apart for slices.
	Comments *[]discussionThread `json:"comments,omitempty"`
}

type discussionThread struct {
	ID        string                `json:"id"`
	Author    string                `json:"author"`
	CreatedAt string                `json:"createdAt"`
	Hidden    bool                  `json:"hidden"`
	Deleted   bool                  `json:"deleted"`
	LikeTotal int                   `json:"likeTotal"`
	Body      string                `json:"body"`
	Replies   []discussionReplyJSON `json:"replies,omitempty"`
}

type discussionReplyJSON struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	Hidden    bool   `json:"hidden"`
	Deleted   bool   `json:"deleted"`
	LikeTotal int    `json:"likeTotal"`
	Body      string `json:"body"`
}

func buildDiscussionViewJSON(detail api.DiscussionDetail, threads []commentThread) discussionViewJSON {
	view := discussionViewJSON{
		Number:       detail.Number,
		Title:        detail.Title,
		Author:       discussionAuthor(detail.Discussion),
		Category:     strings.TrimSpace(detail.Category.Icon + " " + detail.Category.Name),
		Status:       discussionStatus(detail.Discussion),
		CreatedAt:    detail.CreatedAt,
		UpdatedAt:    detail.UpdatedAt,
		CommentTotal: detail.CommentTotal,
		Body:         detail.MDContent,
	}
	// A nil threads slice means comments were not requested; an empty slice
	// means they were requested and there are none.
	if threads != nil {
		entries := make([]discussionThread, 0, len(threads))
		for _, thread := range threads {
			entry := discussionThread{
				ID:        thread.Comment.ID,
				Author:    authorLabel(thread.Comment.Author),
				CreatedAt: thread.Comment.CreatedAt,
				Hidden:    bool(thread.Comment.IsHidden),
				Deleted:   bool(thread.Comment.IsDeleted),
				LikeTotal: thread.Comment.LikeTotal,
				Body:      discussionBody(thread.Comment),
			}
			for _, reply := range thread.Replies {
				entry.Replies = append(entry.Replies, discussionReplyJSON{
					ID:        reply.ID,
					Author:    authorLabel(reply.Author),
					CreatedAt: reply.CreatedAt,
					Hidden:    bool(reply.IsHidden),
					Deleted:   bool(reply.IsDeleted),
					LikeTotal: reply.LikeTotal,
					Body:      discussionBody(reply),
				})
			}
			entries = append(entries, entry)
		}
		view.Comments = &entries
	}
	return view
}
