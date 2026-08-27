package user

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	defaultEventListLimit = 30
	minEventYear          = 1970
	maxEventYear          = 9999
)

type eventOptions struct {
	Year  int
	Limit int
	JSON  bool
}

type userEventWithDate struct {
	Date  string
	Event api.UserEvent
}

// eventJSON is the stable, documented JSON schema emitted by
// `ag user events --json`. Fields are always present so automation can
// distinguish a real zero/empty value from a missing field.
type eventJSON struct {
	Date             string `json:"date"`
	Action           int    `json:"action"`
	ActionName       string `json:"actionName"`
	AuthorID         int64  `json:"authorId"`
	AuthorUsername   string `json:"authorUsername"`
	AuthorName       string `json:"authorName"`
	AuthorURL        string `json:"authorUrl"`
	CreatedAt        string `json:"createdAt"`
	ProjectID        int64  `json:"projectId"`
	ProjectName      string `json:"projectName"`
	TargetID         int64  `json:"targetId"`
	TargetIID        int64  `json:"targetIid"`
	TargetTitle      string `json:"targetTitle"`
	TargetType       string `json:"targetType"`
	TargetTypeFormat string `json:"targetTypeFormat"`
}

func newCmdUserEvents(f *cmdutil.Factory) *cobra.Command {
	opts := &eventOptions{
		Limit: defaultEventListLimit,
	}
	cmd := &cobra.Command{
		Use:   "events [<username>]",
		Short: "List personal activity events for a user",
		Long:  "List personal activity events for a user. Without an explicit username, the authenticated user is used.",
		Example: `  ag user events
  ag user events alice
  ag user events alice --year 2026 --limit 50
  ag user events alice --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserEvents(cmd.OutOrStdout(), f, opts, args)
		},
	}
	cmd.Flags().IntVar(&opts.Year, "year", 0, "Filter events to the specified year (0 disables the filter)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultEventListLimit, "Maximum number of events to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output events as JSON")
	return cmd
}

func runUserEvents(out io.Writer, f *cmdutil.Factory, opts *eventOptions, args []string) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}
	if opts.Year != 0 && (opts.Year < minEventYear || opts.Year > maxEventYear) {
		return fmt.Errorf("invalid year: %d (must be between %d and %d)", opts.Year, minEventYear, maxEventYear)
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	username, err := resolveEventsUsername(f, args)
	if err != nil {
		return err
	}

	events, err := listUserEvents(client, username, opts.Year, opts.Limit)
	if err != nil {
		return err
	}

	if opts.JSON {
		return cmdutil.WriteJSON(out, eventsJSON(events))
	}
	if len(events) == 0 {
		fmt.Fprintln(out, "No events found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tACTION\tPROJECT\tTITLE")
	for _, item := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			cmdutil.EscapeTSVField(item.Date),
			cmdutil.EscapeTSVField(item.Event.ActionName),
			cmdutil.EscapeTSVField(item.Event.ProjectName),
			cmdutil.EscapeTSVField(item.Event.TargetTitle),
		)
	}
	return w.Flush()
}

func resolveEventsUsername(f *cmdutil.Factory, args []string) (string, error) {
	if len(args) == 1 {
		username := strings.TrimSpace(args[0])
		if !validLogin(username) {
			return "", fmt.Errorf("invalid login %q: expected a non-empty user name without path or query separators", args[0])
		}
		return username, nil
	}

	username, err := f.Config.GetUser()
	if err != nil {
		return "", cmdutil.AuthenticationError(err)
	}
	username = strings.TrimSpace(username)
	if !validLogin(username) {
		return "", fmt.Errorf("invalid authenticated login %q: expected a non-empty user name without path or query separators", username)
	}
	return username, nil
}

func listUserEvents(client *api.Client, username string, year, limit int) ([]userEventWithDate, error) {
	events := make([]userEventWithDate, 0)
	seenCursors := make(map[string]struct{})
	cursor := ""

	for {
		query := url.Values{}
		if year != 0 {
			query.Set("year", strconv.Itoa(year))
		}
		if cursor != "" {
			query.Set("next", cursor)
		}

		path := "/users/" + url.PathEscape(username) + "/events"
		if encoded := query.Encode(); encoded != "" {
			path += "?" + encoded
		}

		var page api.UserEventsPage
		if err := client.Get(path, &page); err != nil {
			return nil, fmt.Errorf("list events for user %q: %w", username, err)
		}

		for _, date := range sortedEventDates(page.Events) {
			for _, event := range page.Events[date] {
				events = append(events, userEventWithDate{Date: date, Event: event})
				if len(events) == limit {
					return events, nil
				}
			}
		}

		if page.Next == "" {
			return events, nil
		}
		if _, exists := seenCursors[page.Next]; exists {
			return nil, fmt.Errorf("list events for user %q: pagination made no progress: repeated cursor %q", username, page.Next)
		}
		seenCursors[page.Next] = struct{}{}
		cursor = page.Next
	}
}

func sortedEventDates(events map[string][]api.UserEvent) []string {
	dates := make([]string, 0, len(events))
	for date := range events {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})
	return dates
}

func eventsJSON(events []userEventWithDate) []eventJSON {
	result := make([]eventJSON, 0, len(events))
	for _, item := range events {
		result = append(result, eventJSON{
			Date:             item.Date,
			Action:           item.Event.Action,
			ActionName:       item.Event.ActionName,
			AuthorID:         item.Event.AuthorID,
			AuthorUsername:   item.Event.AuthorUsername,
			AuthorName:       item.Event.Author.Name,
			AuthorURL:        item.Event.Author.WebURL,
			CreatedAt:        item.Event.CreatedAt,
			ProjectID:        item.Event.ProjectID,
			ProjectName:      item.Event.ProjectName,
			TargetID:         item.Event.TargetID,
			TargetIID:        item.Event.TargetIID,
			TargetTitle:      item.Event.TargetTitle,
			TargetType:       item.Event.TargetType,
			TargetTypeFormat: item.Event.TargetTypeFormat,
		})
	}
	return result
}
