package comment

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View all comments on a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number int

			if len(args) == 1 {
				return fmt.Errorf("repository and PR number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number, err = strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[1])
			}

			client := api.NewClient(token)

			var comments []api.Comment
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
			if err := client.Get(path, &comments); err != nil {
				return err
			}

			if len(comments) == 0 {
				fmt.Println("No comments found.")
				return nil
			}

			fmt.Printf("PR #%d 的评论 (共 %d 条):\n\n", number, len(comments))

			// Sort comments by creation time
			sort.Slice(comments, func(i, j int) bool {
				t1, _ := time.Parse(time.RFC3339, comments[i].CreatedAt)
				t2, _ := time.Parse(time.RFC3339, comments[j].CreatedAt)
				return t1.Before(t2)
			})

			// Build comment tree
			commentMap := make(map[int64]*api.Comment)
			children := make(map[int64][]int64)
			var roots []int64

			for i := range comments {
				commentMap[comments[i].ID] = &comments[i]
				if comments[i].ParentID != nil {
					parentID, _ := strconv.ParseInt(*comments[i].ParentID, 10, 64)
					children[parentID] = append(children[parentID], comments[i].ID)
				} else {
					roots = append(roots, comments[i].ID)
				}
			}

			// Print comment tree
			currentUser, _ := f.Config.GetUser()
			for _, rootID := range roots {
				printCommentTree(commentMap, children, rootID, 0, currentUser)
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

func printCommentTree(commentMap map[int64]*api.Comment, children map[int64][]int64, id int64, depth int, currentUser string) {
	comment := commentMap[id]
	if comment == nil {
		return
	}

	indent := strings.Repeat("    ", depth)

	// Format timestamp
	t, _ := time.Parse(time.RFC3339, comment.CreatedAt)
	timeStr := t.Format("2006-01-02 15:04")

	// Mark current user's comments
	userMarker := ""
	if comment.User.Login == currentUser {
		userMarker = " (你)"
	}

	fmt.Printf("%s[%d] @%s %s%s\n", indent, comment.ID, comment.User.Login, timeStr, userMarker)

	// Convert HTML tables to Markdown and print body with indentation
	body := convertHTMLToMarkdown(comment.Body)
	bodyLines := strings.Split(body, "\n")
	for _, line := range bodyLines {
		fmt.Printf("%s    %s\n", indent, line)
	}
	fmt.Println()

	// Print children
	for _, childID := range children[id] {
		printCommentTree(commentMap, children, childID, depth+1, currentUser)
	}
}
