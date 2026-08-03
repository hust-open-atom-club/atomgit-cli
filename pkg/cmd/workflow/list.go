package workflow

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type listOptions struct {
	JSON bool
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List workflows in a repository",
		Example: `  ag workflow list owner/repo`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := requireToken(f)
			if err != nil {
				return err
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}

			client, err := newActionsClient(f, token)
			if err != nil {
				return err
			}

			res, err := client.ListWorkflows(repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("failed to list workflows for %s/%s: %w", repository.Owner, repository.Name, err)
			}

			if opts.JSON {
				data, err := json.MarshalIndent(res, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal json output: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if len(res.Workflows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No workflows found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATE\tPATH")
			for _, wf := range res.Workflows {
				id := wf.ID
				if id == "" {
					id = "-"
				}
				name := wf.Name
				if name == "" {
					name = "-"
				}
				state := wf.State
				if state == "" {
					state = "-"
				}
				path := wf.Path
				if path == "" {
					path = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, name, state, path)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output workflows as JSON")

	return cmd
}
