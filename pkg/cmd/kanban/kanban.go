package kanban

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const defaultKanbanLimit = 30

type listOptions struct {
	Limit int
	JSON  bool
}

type itemOptions struct {
	Limit int
	JSON  bool
}

type kanbanJSON struct {
	ID          string `json:"id"`
	IID         int    `json:"iid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	Visibility  int    `json:"visibility"`
	UpdatedAt   string `json:"updatedAt"`
}

type kanbanItemJSON struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Column string `json:"column"`
	URL    string `json:"url"`
}

func NewCmdKanban(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "View organization Kanban boards",
		Long:  "List and inspect read-only organization Kanban boards and their Issue/Pull Request items.",
		Example: `  ag kanban list hust-open-atom-club
  ag kanban view hust-open-atom-club 1234567890
  ag kanban items hust-open-atom-club 1234567890 --json`,
	}
	cmd.AddCommand(newCmdKanbanList(f))
	cmd.AddCommand(newCmdKanbanView(f))
	cmd.AddCommand(newCmdKanbanItems(f))
	return cmd
}

func newCmdKanbanList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: defaultKanbanLimit}
	cmd := &cobra.Command{
		Use:   "list <owner>",
		Short: "List organization Kanban boards",
		Args:  cobra.ExactArgs(1),
		Example: `  ag kanban list hust-open-atom-club
  ag kanban list hust-open-atom-club --limit 50 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, err := api.ValidateKanbanOwner(args[0])
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			boards, err := api.GetKanbans(client, owner, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list Kanban boards for %s: %w", owner, err)
			}
			return writeKanbanList(cmd.OutOrStdout(), boards, opts.JSON)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultKanbanLimit, "Maximum number of Kanban boards to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output Kanban boards as JSON")
	return cmd
}

func newCmdKanbanView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "view <owner> <kanban-id>",
		Short:   "View a Kanban board",
		Args:    cobra.ExactArgs(2),
		Example: "  ag kanban view hust-open-atom-club 1234567890 --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, kanbanID, err := validateBoardArgs(args)
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			board, err := api.GetKanban(client, owner, kanbanID)
			if err != nil {
				return fmt.Errorf("failed to view Kanban %s for %s: %w", kanbanID, owner, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), toKanbanJSON(board))
			}
			writeKanbanDetail(cmd.OutOrStdout(), board)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the Kanban board as JSON")
	return cmd
}

func newCmdKanbanItems(f *cmdutil.Factory) *cobra.Command {
	opts := &itemOptions{Limit: defaultKanbanLimit}
	cmd := &cobra.Command{
		Use:   "items <owner> <kanban-id>",
		Short: "List items on a Kanban board",
		Args:  cobra.ExactArgs(2),
		Example: `  ag kanban items hust-open-atom-club 1234567890
  ag kanban items hust-open-atom-club 1234567890 --limit 50 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, kanbanID, err := validateBoardArgs(args)
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			items, err := api.GetKanbanItems(client, owner, kanbanID, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list items for Kanban %s in %s: %w", kanbanID, owner, err)
			}
			return writeKanbanItems(cmd.OutOrStdout(), items, opts.JSON)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultKanbanLimit, "Maximum number of Kanban items to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output Kanban items as JSON")
	return cmd
}

func authenticatedClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return f.NewAPIClient(token)
}

func validateBoardArgs(args []string) (string, string, error) {
	owner, err := api.ValidateKanbanOwner(args[0])
	if err != nil {
		return "", "", err
	}
	kanbanID, err := api.ValidateKanbanID(args[1])
	if err != nil {
		return "", "", err
	}
	return owner, kanbanID, nil
}

func writeKanbanList(out io.Writer, boards []api.Kanban, jsonOutput bool) error {
	if jsonOutput {
		result := make([]kanbanJSON, len(boards))
		for i, board := range boards {
			result[i] = toKanbanJSON(board)
		}
		return cmdutil.WriteJSON(out, result)
	}
	if len(boards) == 0 {
		_, err := fmt.Fprintln(out, "No Kanban boards found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tIID\tSTATUS\tNAME\tUPDATED"); err != nil {
		return err
	}
	for _, board := range boards {
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", board.ID, board.IID, boardStatus(board.Status), displayValue(board.Name), displayValue(board.UpdatedAt)); err != nil {
			return err
		}
	}
	return w.Flush()
}

func writeKanbanDetail(out io.Writer, board api.Kanban) {
	fmt.Fprintf(out, "ID: %s\n", displayValue(board.ID))
	fmt.Fprintf(out, "IID: %d\n", board.IID)
	fmt.Fprintf(out, "Name: %s\n", displayValue(board.Name))
	fmt.Fprintf(out, "Status: %s\n", boardStatus(board.Status))
	fmt.Fprintf(out, "Visibility: %d\n", board.Visibility)
	if strings.TrimSpace(board.Description) != "" {
		fmt.Fprintf(out, "Description: %s\n", strings.ReplaceAll(strings.TrimSpace(board.Description), "\n", " "))
	}
	if strings.TrimSpace(board.UpdatedAt) != "" {
		fmt.Fprintf(out, "Updated: %s\n", board.UpdatedAt)
	}
}

func writeKanbanItems(out io.Writer, items []api.KanbanItem, jsonOutput bool) error {
	if jsonOutput {
		result := make([]kanbanItemJSON, len(items))
		for i, item := range items {
			result[i] = toKanbanItemJSON(item)
		}
		return cmdutil.WriteJSON(out, result)
	}
	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "No Kanban items found.")
		return err
	}
	for _, item := range items {
		kind := kanbanItemType(item.SourceType)
		line := fmt.Sprintf("%s #%s [%s] %s", kind, displayValue(item.GetNumber()), displayValue(item.Status), displayValue(item.Title))
		if column := item.Column(); column != "" {
			line += " column:" + column
		}
		if item.HTMLURL != "" {
			line += " " + item.HTMLURL
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func toKanbanJSON(board api.Kanban) kanbanJSON {
	return kanbanJSON{ID: board.ID, IID: board.IID, Name: board.Name, Description: board.Description, Status: board.Status, Visibility: board.Visibility, UpdatedAt: board.UpdatedAt}
}

func toKanbanItemJSON(item api.KanbanItem) kanbanItemJSON {
	return kanbanItemJSON{ID: item.ID, Number: item.GetNumber(), Type: kanbanItemType(item.SourceType), Title: item.Title, Status: item.Status, Column: item.Column(), URL: firstNonEmpty(item.HTMLURL, item.URL)}
}

func kanbanItemType(sourceType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "pull_request", "pull-request", "pullrequest", "pr":
		return "Pull Request"
	case "issue":
		return "Issue"
	default:
		if strings.TrimSpace(sourceType) == "" {
			return "Item"
		}
		return sourceType
	}
}

func boardStatus(status int) string {
	if status == 0 {
		return "open"
	}
	return "closed"
}

func displayValue(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
