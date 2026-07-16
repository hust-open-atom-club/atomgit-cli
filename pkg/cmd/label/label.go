package label

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdLabel(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage repository labels",
		Long:  `List and manage repository labels.`,
	}

	cmd.AddCommand(newCmdLabelList(f))
	return cmd
}

func newCmdLabelList(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list <owner>/<repo>",
		Short:   "List repository labels",
		Example: `  ag label list owner/repo --limit 50`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}
			if len(args) == 0 {
				return fmt.Errorf("repository required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo := parts[0], parts[1]

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}
			labels, err := api.GetPaginated[api.Label](client, limit, func(page, perPage int) string {
				return fmt.Sprintf("/repos/%s/%s/labels?page=%d&per_page=%d", owner, repo, page, perPage)
			})
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}

			out := cmd.OutOrStdout()
			for _, label := range labels {
				fmt.Fprintf(out, "%s [%s]", label.Name, label.Color)
				if description := strings.TrimSpace(label.Description); description != "" {
					fmt.Fprintf(out, " %s", description)
				}
				fmt.Fprintln(out)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of labels to list")
	return cmd
}
