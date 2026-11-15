package user

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type followingOptions struct {
	Limit int
	JSON  bool
}

func newCmdUserFollowing(f *cmdutil.Factory) *cobra.Command {
	opts := &followingOptions{}
	cmd := &cobra.Command{
		Use: "following <username>", Short: "List users and organizations followed by a user",
		Long:    "List followed users and organizations for an explicit username. Missing login credentials allow anonymous access; credential-store failures are reported. This command only reads data and does not infer a user from Git. A hidden or nonexistent user can return HTTP 404.",
		Example: "  ag-cli user following alice\n  ag-cli user following alice --limit 100 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimSpace(args[0])
			if !validLogin(username) {
				return fmt.Errorf("invalid login %q: expected a non-empty user name without path or query separators", args[0])
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			client, err := f.OptionalAPIClient()
			if err != nil {
				return err
			}
			items, err := api.ListUserFollowing(client.WithContext(cmd.Context()), username, opts.Limit)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if opts.JSON {
				return cmdutil.WriteJSON(out, items)
			}
			if len(items) == 0 {
				_, err := fmt.Fprintln(out, "No followed users or organizations found.")
				return err
			}
			w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "LOGIN\tNAME\tTYPE\tURL")
			for _, item := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cmdutil.EscapeTSVField(item.Login), cmdutil.EscapeTSVField(item.Name), cmdutil.EscapeTSVField(item.Type), cmdutil.EscapeTSVField(item.HTMLURL))
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum total number of followed accounts to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output followed accounts as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Explicit username; Git repository context is not used."},
		Output:      "text,json", Effects: "read", Notes: []string{"Anonymous access is allowed when not logged in; credential-store errors are not ignored.", "Results include User and Organization objects. URLs are not synthesized."},
	})
	return cmd
}
