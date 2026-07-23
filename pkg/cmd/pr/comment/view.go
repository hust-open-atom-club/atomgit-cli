package comment

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view [<owner>/<repo>] <number>",
		Short: "View all comments on a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			number, err := strconv.Atoi(remaining[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", remaining[0])
			}

			client := api.NewClient(token)

			var comments []api.Comment
			// view=all returns both 普通评论 (pr_comment) and 检视意见 (diff_comment),
			// with replies nested under each comment's `reply` field.
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments?view=all", owner, repo, number)
			if err := client.Get(path, &comments); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(comments) == 0 {
				fmt.Fprintln(out, "No comments found.")
				return nil
			}

			fmt.Fprintf(out, "PR #%d 的评论 (共 %d 条):\n\n", number, len(comments))

			// Sort top-level comments by creation time.
			sortCommentsByTime(comments)

			currentUser, _ := f.Config.GetUser()
			for i := range comments {
				printComment(out, &comments[i], currentUser)
			}

			return nil
		},
	}
}

// convertHTMLToMarkdown converts HTML content to Markdown, focusing on tables
func convertHTMLToMarkdown(body string) string {
	// Check if body contains HTML table
	if !strings.Contains(body, "<table") {
		return body
	}

	// Use custom table converter for better control
	return convertHTMLTableToMarkdown(body)
}

// convertHTMLTableToMarkdown converts HTML tables to Markdown format
func convertHTMLTableToMarkdown(html string) string {
	// Extract and convert each table
	tableRegex := regexp.MustCompile(`(?s)<table[^>]*>(.*?)</table>`)
	result := tableRegex.ReplaceAllStringFunc(html, func(tableHTML string) string {
		return parseTable(tableHTML)
	})

	// Clean up remaining HTML tags (except links which we'll handle)
	result = cleanHTMLTags(result)

	return strings.TrimSpace(result)
}

// parseTable parses a single HTML table and converts it to Markdown
func parseTable(tableHTML string) string {
	var rows [][]string
	var maxCols int

	// Extract all rows
	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	cellRegex := regexp.MustCompile(`(?s)<t[dh](?:[^>]*)>(.*?)</t[dh]>`)

	rowMatches := rowRegex.FindAllStringSubmatch(tableHTML, -1)
	for _, rowMatch := range rowMatches {
		if len(rowMatch) < 2 {
			continue
		}
		rowContent := rowMatch[1]

		var cells []string
		cellMatches := cellRegex.FindAllStringSubmatch(rowContent, -1)
		for _, cellMatch := range cellMatches {
			if len(cellMatch) >= 2 {
				cell := cleanHTMLTags(cellMatch[1])
				cell = strings.TrimSpace(cell)
				cells = append(cells, cell)
			}
		}

		if len(cells) > 0 {
			rows = append(rows, cells)
			if len(cells) > maxCols {
				maxCols = len(cells)
			}
		}
	}

	if len(rows) == 0 {
		return ""
	}

	// Build Markdown table
	var md strings.Builder

	// Header row
	for i, cell := range rows[0] {
		if i > 0 {
			md.WriteString(" | ")
		}
		md.WriteString(cell)
	}
	md.WriteString("\n")

	// Separator row
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			md.WriteString(" | ")
		}
		md.WriteString("---")
	}
	md.WriteString("\n")

	// Data rows
	for i := 1; i < len(rows); i++ {
		for j, cell := range rows[i] {
			if j > 0 {
				md.WriteString(" | ")
			}
			md.WriteString(cell)
		}
		md.WriteString("\n")
	}

	return md.String()
}

// cleanHTMLTags removes HTML tags but preserves links
func cleanHTMLTags(html string) string {
	// First, convert <a href="...">text</a> to [text](url)
	linkRegex := regexp.MustCompile(`<a\s+href="([^"]*)"[^>]*>(.*?)</a>`)
	result := linkRegex.ReplaceAllString(html, "[$2]($1)")

	// Remove all other HTML tags
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	result = tagRegex.ReplaceAllString(result, "")

	// Decode common HTML entities
	result = strings.ReplaceAll(result, "&#9989;", "✅")
	result = strings.ReplaceAll(result, "&#10060;", "❌")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&quot;", "\"")

	return result
}

// sortCommentsByTime orders comments from oldest to newest by created_at.
func sortCommentsByTime(comments []api.Comment) {
	sort.Slice(comments, func(i, j int) bool {
		t1, _ := time.Parse(time.RFC3339, comments[i].CreatedAt)
		t2, _ := time.Parse(time.RFC3339, comments[j].CreatedAt)
		return t1.Before(t2)
	})
}

// formatTime parses an RFC3339 timestamp and renders it with the given layout,
// falling back to the raw string when parsing fails.
func formatTime(raw, layout string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format(layout)
}

// youMarker returns " (你)" when the comment belongs to the current user.
func youMarker(c *api.Comment, currentUser string) string {
	if currentUser != "" && c.User.Login == currentUser {
		return " (你)"
	}
	return ""
}

// printBody prints a comment body (HTML tables converted to Markdown), prefixing
// every line with indent.
func printBody(w io.Writer, body, indent string) {
	body = convertHTMLToMarkdown(body)
	for _, line := range strings.Split(body, "\n") {
		fmt.Fprintf(w, "%s%s\n", indent, line)
	}
}

// diffLocation renders the file + line range for a 检视意见 (diff_comment).
func diffLocation(c *api.Comment) string {
	dp := c.DiffPosition

	path := c.DiffFile
	if path == "" {
		path = c.Path
	}
	if path == "" && dp != nil {
		switch {
		case dp.NewPath != "":
			path = dp.NewPath
		case dp.Path != "":
			path = dp.Path
		case dp.OldPath != "":
			path = dp.OldPath
		}
	}

	var lines string
	if dp != nil {
		start, end := dp.StartNewLine, dp.EndNewLine
		if start == 0 && end == 0 {
			start, end = dp.StartOldLine, dp.EndOldLine
		}
		switch {
		case start > 0 && end > 0 && start != end:
			lines = fmt.Sprintf("L%d–%d", start, end)
		case start > 0:
			lines = fmt.Sprintf("L%d", start)
		case end > 0:
			lines = fmt.Sprintf("L%d", end)
		}
	}

	switch {
	case path != "" && lines != "":
		return fmt.Sprintf("%s  %s", path, lines)
	case path != "":
		return path
	default:
		return lines
	}
}

// printComment renders a top-level comment, then its nested replies.
//
//	[discussion_id]
//	[id] @user reviewed 2026-06-26 14:32 (你)  [未解决]  [file  L25–27]
//	   body
//	   └─[id]  @user reply 2026-06-26 14:55 (你)
//	      body
func printComment(w io.Writer, comment *api.Comment, currentUser string) {
	if comment == nil {
		return
	}

	// discussion_id on its own line (required to reply to the thread).
	if comment.DiscussionID != "" {
		fmt.Fprintf(w, "[%s]\n", comment.DiscussionID)
	}

	verb := "commented"
	if comment.CommentType == "diff_comment" {
		verb = "reviewed"
	}

	header := fmt.Sprintf("[%d] @%s %s %s%s",
		comment.ID, comment.User.Login, verb,
		formatTime(comment.CreatedAt, "2006-01-02 15:04"),
		youMarker(comment, currentUser))

	// For 检视意见, append status and file/line range as bracketed tags.
	if comment.CommentType == "diff_comment" {
		status := "未解决"
		if comment.Resolved {
			status = "已解决"
		}
		header += "  [" + status + "]"
		if loc := diffLocation(comment); loc != "" {
			header += "  [" + loc + "]"
		}
	}
	fmt.Fprintln(w, header)

	printBody(w, comment.Body, "   ")

	sortCommentsByTime(comment.Reply)
	for i := range comment.Reply {
		printReply(w, &comment.Reply[i], currentUser)
	}

	fmt.Fprintln(w)
}

// printReply renders a nested reply under a top-level comment.
func printReply(w io.Writer, comment *api.Comment, currentUser string) {
	fmt.Fprintf(w, "   └─[%d]  @%s reply %s%s\n",
		comment.ID, comment.User.Login,
		formatTime(comment.CreatedAt, "2006-01-02 15:04"),
		youMarker(comment, currentUser))
	printBody(w, comment.Body, "      ")

	// Flatten any further-nested replies at the same level.
	sortCommentsByTime(comment.Reply)
	for i := range comment.Reply {
		printReply(w, &comment.Reply[i], currentUser)
	}
}
