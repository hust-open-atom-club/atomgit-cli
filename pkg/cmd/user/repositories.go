package user

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const defaultUserRepositoryListLimit = 30

type userRepositoryOptions struct {
	Limit int
	JSON  bool
}

type userRepositoryCollection struct {
	command      string
	endpoint     string
	description  string
	emptyMessage string
}

type userRepositoryJSON struct {
	ID       int64  `json:"id"`
	FullName string `json:"fullName"`
	URL      string `json:"url"`
}

var (
	starredRepositories = userRepositoryCollection{
		command:      "starred",
		endpoint:     "starred",
		description:  "starred repositories",
		emptyMessage: "No starred repositories found.",
	}
	watchedRepositories = userRepositoryCollection{
		command:      "watching",
		endpoint:     "subscriptions",
		description:  "watched repositories",
		emptyMessage: "No watched repositories found.",
	}
)

func newCmdUserStarred(f *cmdutil.Factory) *cobra.Command {
	return newCmdUserRepositoryCollection(f, starredRepositories)
}

func newCmdUserWatching(f *cmdutil.Factory) *cobra.Command {
	return newCmdUserRepositoryCollection(f, watchedRepositories)
}

func newCmdUserRepositoryCollection(f *cmdutil.Factory, collection userRepositoryCollection) *cobra.Command {
	opts := &userRepositoryOptions{Limit: defaultUserRepositoryListLimit}
	cmd := &cobra.Command{
		Use:   collection.command + " [<username>]",
		Short: "List " + collection.description + " for a user",
		Long: fmt.Sprintf(
			"List %s for a user. Without a username, the authenticated-user endpoint is used; an explicit username selects the public-user endpoint. Authentication is required for both endpoints.",
			collection.description,
		),
		Example: fmt.Sprintf("  ag user %s\n  ag user %s alice --limit 100\n  ag user %s alice --json", collection.command, collection.command, collection.command),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserRepositoryCollection(cmd.OutOrStdout(), f, opts, args, collection)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultUserRepositoryListLimit, "Maximum number of repositories to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output repositories as JSON")
	return cmd
}

func runUserRepositoryCollection(out io.Writer, f *cmdutil.Factory, opts *userRepositoryOptions, args []string, collection userRepositoryCollection) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}

	username := ""
	if len(args) == 1 {
		username = strings.TrimSpace(args[0])
		if !validLogin(username) {
			return fmt.Errorf("invalid login %q: expected a non-empty user name without path or query separators", args[0])
		}
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	endpoint := "/user/" + collection.endpoint
	target := "authenticated user"
	if username != "" {
		endpoint = "/users/" + url.PathEscape(username) + "/" + collection.endpoint
		target = fmt.Sprintf("user %q", username)
	}

	repositories, err := api.GetPaginated[api.Repository](client, opts.Limit, func(page, perPage int) string {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(perPage))
		return endpoint + "?" + query.Encode()
	})
	if err != nil {
		return fmt.Errorf("list %s for %s: %w", collection.description, target, err)
	}

	items, err := userRepositoriesJSON(repositories, f.Config.GetHost())
	if err != nil {
		return fmt.Errorf("list %s for %s: %w", collection.description, target, err)
	}
	if opts.JSON {
		return cmdutil.WriteJSON(out, items)
	}
	if len(items) == 0 {
		fmt.Fprintln(out, collection.emptyMessage)
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tURL")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\n", cmdutil.EscapeTSVField(item.FullName), cmdutil.EscapeTSVField(item.URL))
	}
	return w.Flush()
}

func userRepositoriesJSON(repositories []api.Repository, host string) ([]userRepositoryJSON, error) {
	items := make([]userRepositoryJSON, len(repositories))
	for index, repository := range repositories {
		fullName := userRepositoryFullName(repository)
		if fullName == "" {
			return nil, fmt.Errorf("repository at index %d did not include an unambiguous full name", index)
		}
		webURL := strings.TrimSpace(repository.HTMLURL)
		if webURL == "" {
			webURL = strings.TrimSpace(repository.AlternateHTMLURL)
		}
		if webURL == "" {
			webURL = cmdutil.ResolveWebURL("", host, strings.Split(fullName, "/")...)
		}
		items[index] = userRepositoryJSON{ID: repository.ID, FullName: fullName, URL: webURL}
	}
	return items, nil
}

func userRepositoryFullName(repository api.Repository) string {
	if fullName := strings.Trim(strings.TrimSpace(repository.FullName), "/"); fullName != "" {
		return fullName
	}
	owner := strings.TrimSpace(repository.Namespace.Path)
	if owner == "" {
		owner = strings.TrimSpace(repository.Owner.Login)
	}
	name := strings.TrimSpace(repository.Name)
	if owner == "" || name == "" {
		return ""
	}
	return owner + "/" + name
}
