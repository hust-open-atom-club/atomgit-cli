package milestone

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const milestoneDateLayout = "2006-01-02"

// NewCmdMilestone creates the milestone command group.
func NewCmdMilestone(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milestone",
		Short: "Manage repository milestones",
		Long: `List, view, create, update, close, reopen, and delete repository milestones.

AtomGit requires title and due_on on every milestone PATCH. Edit, close, and
reopen read the current milestone first and preserve those required fields.`,
	}
	cmd.AddCommand(newCmdMilestoneList(f))
	cmd.AddCommand(newCmdMilestoneView(f))
	cmd.AddCommand(newCmdMilestoneCreate(f))
	cmd.AddCommand(newCmdMilestoneEdit(f))
	cmd.AddCommand(newCmdMilestoneState(f, "close", "closed"))
	cmd.AddCommand(newCmdMilestoneState(f, "reopen", "open"))
	cmd.AddCommand(newCmdMilestoneDelete(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newAPIClient(f *cmdutil.Factory, token string) (*api.Client, error) {
	if f.HttpClient == nil {
		return api.NewClient(token), nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), nil
}

func newCmdMilestoneList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		State     string
		Sort      string
		Direction string
		Limit     int
		JSON      bool
	}
	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List repository milestones",
		Example: "  ag milestone list owner/repo --state all --limit 50",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMilestoneListOptions(opts.State, opts.Sort, opts.Direction, opts.Limit); err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			milestones, err := api.GetPaginated[api.Milestone](client, opts.Limit, func(page, perPage int) string {
				query := url.Values{
					"state":     {opts.State},
					"sort":      {opts.Sort},
					"direction": {opts.Direction},
					"page":      {strconv.Itoa(page)},
					"per_page":  {strconv.Itoa(perPage)},
				}
				return milestoneCollectionPath(repository) + "?" + query.Encode()
			})
			if err != nil {
				return fmt.Errorf("failed to list milestones: %w", err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), milestonesJSON(milestones))
			}
			out := cmd.OutOrStdout()
			if len(milestones) == 0 {
				fmt.Fprintln(out, "No milestones found.")
				return nil
			}
			for _, item := range milestones {
				fmt.Fprintf(out, "#%s %s [%s]", item.GetNumber(), item.Title, item.State)
				if item.DueOn != "" {
					fmt.Fprintf(out, " due %s", item.DueOn)
				}
				fmt.Fprintln(out)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().StringVar(&opts.Sort, "sort", "due_on", "Sort milestones by due_on")
	cmd.Flags().StringVar(&opts.Direction, "direction", "asc", "Sort direction: asc, desc")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of milestones to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output milestones as JSON")
	return cmd
}

func newCmdMilestoneView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <number>",
		Short: "View a repository milestone",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parseMilestoneNumber(remaining[0])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			var item api.Milestone
			if err := client.Get(milestonePath(repository, number), &item); err != nil {
				return fmt.Errorf("failed to get milestone #%s: %w", number, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newMilestoneJSON(item))
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Milestone: #%s %s\n", item.GetNumber(), item.Title)
			fmt.Fprintf(out, "State: %s\n", item.State)
			fmt.Fprintf(out, "Due: %s\n", displayMilestoneDueDate(item.DueOn))
			fmt.Fprintf(out, "Issues: %d open, %d closed\n", item.OpenIssues, item.ClosedIssues)
			fmt.Fprintf(out, "URL: %s\n", milestoneResultURL(item, f.Config.GetHost(), repository, number))
			if item.Description != "" {
				fmt.Fprintf(out, "\n%s\n", item.Description)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output milestone as JSON")
	return cmd
}

func newCmdMilestoneCreate(f *cmdutil.Factory) *cobra.Command {
	var title, description, dueOn string
	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>]",
		Short: "Create a repository milestone",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title = strings.TrimSpace(title)
			if title == "" {
				return fmt.Errorf("milestone title is required")
			}
			if err := validateMilestoneDate(dueOn); err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			body := map[string]string{"title": title, "description": description, "due_on": dueOn}
			var created api.Milestone
			if err := client.Post(milestoneCollectionPath(repository), body, &created); err != nil {
				return fmt.Errorf("failed to create milestone: %w", err)
			}
			number := created.GetNumber()
			if number == "" {
				return fmt.Errorf("created milestone response did not include a number")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created milestone #%s: %s\n", number, milestoneResultURL(created, f.Config.GetHost(), repository, number))
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Milestone title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Milestone description")
	cmd.Flags().StringVar(&dueOn, "due-on", "", "Due date in YYYY-MM-DD format")
	return cmd
}

func newCmdMilestoneEdit(f *cmdutil.Factory) *cobra.Command {
	var title, description, dueOn string
	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <number>",
		Short: "Edit a repository milestone",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			changes := make(map[string]string)
			if cmd.Flags().Changed("title") {
				title = strings.TrimSpace(title)
				if title == "" {
					return fmt.Errorf("milestone title must not be empty")
				}
				changes["title"] = title
			}
			if cmd.Flags().Changed("description") {
				changes["description"] = description
			}
			if cmd.Flags().Changed("due-on") {
				if err := validateMilestoneDate(dueOn); err != nil {
					return err
				}
				changes["due_on"] = dueOn
			}
			if len(changes) == 0 {
				return fmt.Errorf("at least one of --title, --description, or --due-on must be provided")
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parseMilestoneNumber(remaining[0])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			body, err := milestoneUpdateBody(client, repository, number, changes)
			if err != nil {
				return err
			}
			var updated api.Milestone
			if err := client.Patch(milestonePath(repository, number), body, &updated); err != nil {
				return fmt.Errorf("failed to edit milestone #%s: %w", number, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated milestone #%s: %s\n", number, milestoneResultURL(updated, f.Config.GetHost(), repository, number))
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "New milestone title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New milestone description; pass an empty value to clear")
	cmd.Flags().StringVar(&dueOn, "due-on", "", "New due date in YYYY-MM-DD format")
	return cmd
}

func newCmdMilestoneState(f *cmdutil.Factory, commandName, state string) *cobra.Command {
	pastTense := "Closed"
	commandTitle := "Close"
	if commandName == "reopen" {
		pastTense = "Reopened"
		commandTitle = "Reopen"
	}
	return &cobra.Command{
		Use:   commandName + " [<owner>/<repo>] <number>",
		Short: commandTitle + " a repository milestone",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parseMilestoneNumber(remaining[0])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			body, err := milestoneUpdateBody(client, repository, number, map[string]string{"state": state})
			if err != nil {
				return err
			}
			var updated api.Milestone
			if err := client.Patch(milestonePath(repository, number), body, &updated); err != nil {
				return fmt.Errorf("failed to %s milestone #%s: %w", commandName, number, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s milestone #%s: %s\n", pastTense, number, milestoneResultURL(updated, f.Config.GetHost(), repository, number))
			return nil
		},
	}
}

func newCmdMilestoneDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <number>",
		Short: "Delete a repository milestone",
		Long:  "Permanently delete a milestone. This differs from closing it and requires confirmation unless --yes is used.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parseMilestoneNumber(remaining[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !yes {
				confirmed, err := confirmMilestoneDelete(cmd.InOrStdin(), out, repository, number)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Deletion cancelled.")
					return nil
				}
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			if err := client.Delete(milestonePath(repository, number)); err != nil {
				return fmt.Errorf("failed to delete milestone #%s: %w", number, err)
			}
			fmt.Fprintf(out, "Deleted milestone #%s\n", number)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func authenticatedClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated: %w", err)
	}
	return newAPIClient(f, token)
}

func milestoneCollectionPath(repository cmdutil.Repository) string {
	return fmt.Sprintf("/repos/%s/%s/milestones", repository.Owner, repository.Name)
}

func milestonePath(repository cmdutil.Repository, number string) string {
	return milestoneCollectionPath(repository) + "/" + number
}

func milestoneUpdateBody(client *api.Client, repository cmdutil.Repository, number string, changes map[string]string) (map[string]string, error) {
	var current api.Milestone
	if err := client.Get(milestonePath(repository, number), &current); err != nil {
		return nil, fmt.Errorf("failed to get milestone #%s before update: %w", number, err)
	}
	title := strings.TrimSpace(current.Title)
	dueOn := strings.TrimSpace(current.DueOn)
	if title == "" || dueOn == "" {
		return nil, fmt.Errorf("milestone #%s is missing API-required title or due date", number)
	}
	body := map[string]string{"title": current.Title, "due_on": current.DueOn}
	for key, value := range changes {
		body[key] = value
	}
	return body, nil
}

func parseMilestoneNumber(value string) (string, error) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return "", fmt.Errorf("invalid milestone number %q (expected a positive integer)", value)
	}
	return strconv.Itoa(number), nil
}

func validateMilestoneDate(value string) error {
	if value == "" {
		return fmt.Errorf("milestone due date is required in YYYY-MM-DD format")
	}
	parsed, err := time.Parse(milestoneDateLayout, value)
	if err != nil || parsed.Format(milestoneDateLayout) != value {
		return fmt.Errorf("invalid milestone due date %q (expected YYYY-MM-DD)", value)
	}
	return nil
}

func validateMilestoneListOptions(state, sortBy, direction string, limit int) error {
	if state != "open" && state != "closed" && state != "all" {
		return fmt.Errorf("invalid milestone state %q (expected open, closed, or all)", state)
	}
	if sortBy != "due_on" {
		return fmt.Errorf("invalid milestone sort %q (expected due_on)", sortBy)
	}
	if direction != "asc" && direction != "desc" {
		return fmt.Errorf("invalid milestone direction %q (expected asc or desc)", direction)
	}
	if limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	return nil
}

func confirmMilestoneDelete(in io.Reader, out io.Writer, repository cmdutil.Repository, number string) (bool, error) {
	fmt.Fprintf(out, "Permanently delete milestone #%s from %s? [y/N] ", number, repository.String())
	var response string
	if _, err := fmt.Fscan(in, &response); err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes"), nil
}

func milestoneResultURL(item api.Milestone, host string, repository cmdutil.Repository, fallbackNumber string) string {
	number := item.GetNumber()
	if number == "" {
		number = fallbackNumber
	}
	return cmdutil.ResolveWebURL(item.URL, host, repository.Owner, repository.Name, "milestones", number)
}

func displayMilestoneDueDate(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

type milestoneJSON struct {
	Number       string `json:"number"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	State        string `json:"state"`
	DueOn        string `json:"dueOn"`
	OpenIssues   int    `json:"openIssues"`
	ClosedIssues int    `json:"closedIssues"`
	URL          string `json:"url"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func milestonesJSON(items []api.Milestone) []milestoneJSON {
	result := make([]milestoneJSON, len(items))
	for index, item := range items {
		result[index] = newMilestoneJSON(item)
	}
	return result
}

func newMilestoneJSON(item api.Milestone) milestoneJSON {
	return milestoneJSON{
		Number: item.GetNumber(), Title: item.Title, Description: item.Description, State: item.State,
		DueOn: item.DueOn, OpenIssues: item.OpenIssues, ClosedIssues: item.ClosedIssues,
		URL: item.URL, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
