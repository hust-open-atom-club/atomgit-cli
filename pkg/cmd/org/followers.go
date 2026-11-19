package org

import (
	"fmt"
	"io"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgFollowers(f *cmdutil.Factory) *cobra.Command {
	opts := &collectionOptions{Limit: defaultOrganizationCollectionLimit}
	cmd := &cobra.Command{
		Use:   "followers <org>",
		Short: "List organization followers",
		Long:  "List users following an explicitly specified organization. Followers are distinct from organization members. Follow timestamps retain the server's timezone.",
		Example: `  ag-cli org followers hust-open-atom-club
  ag-cli org followers hust-open-atom-club --limit 100
  ag-cli org followers hust-open-atom-club --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			return runOrgFollowers(cmd.OutOrStdout(), f, organization, opts)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultOrganizationCollectionLimit, "Maximum number of followers to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output followers as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "One explicit organization path."},
		Output:      "text,json", Effects: "read",
		Notes: []string{"Requires authentication. --limit must be positive and limits the total number of followers across pages. JSON preserves id, login, name, avatar_url, and watch_at; watch_at is the follow time, not a membership date."},
	})
	return cmd
}

func runOrgFollowers(out io.Writer, f *cmdutil.Factory, organization string, opts *collectionOptions) error {
	client, err := organizationAPIClient(f)
	if err != nil {
		return err
	}
	followers, err := api.ListOrganizationFollowers(client, organization, opts.Limit)
	if err != nil {
		return fmt.Errorf("failed to list followers for organization %q: %w", organization, err)
	}
	if opts.JSON {
		return cmdutil.WriteJSON(out, followers)
	}
	if len(followers) == 0 {
		fmt.Fprintf(out, "No followers found for organization %q.\n", organization)
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tWATCH AT")
	for _, follower := range followers {
		fmt.Fprintf(w, "%s\t%s\t%s\n", displayValue(follower.Login), displayValue(follower.Name), displayValue(follower.WatchAt))
	}
	return w.Flush()
}
