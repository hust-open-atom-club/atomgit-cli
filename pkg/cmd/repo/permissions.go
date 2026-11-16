package repo

import (
	"encoding/json"
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type permissionsOptions struct {
	JSON bool
}

type permissionsJSON struct {
	Repository  string                    `json:"repository"`
	Permissions api.RepositoryPermissions `json:"permissions"`
}

func newCmdRepoPermissions(f *cmdutil.Factory) *cobra.Command {
	opts := &permissionsOptions{}
	cmd := &cobra.Command{
		Use:     "permissions [<owner>/<repo>]",
		Short:   "View your repository role and permission points",
		Long:    "View the currently authenticated account's role and resource/action permission matrix.\nRequires authentication; anonymous access is not used to determine account permissions.\nPreserves explicit selected=false values and unknown fields. A successful request or a role name does not imply all permissions. No roles are modified.",
		Example: "  ag-cli repo permissions owner/repo\n  ag-cli repo permissions --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			permissions, err := api.GetRepositoryPermissions(client.WithContext(cmd.Context()), repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("failed to view permissions for %s: %w", repository, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), permissionsJSON{Repository: repository.String(), Permissions: permissions})
			}
			// JSON quoting keeps embedded tabs, newlines, and control characters
			// inside remote field names and values from becoming text structure.
			formatted, err := json.MarshalIndent(permissions, "", "  ")
			if err != nil {
				return fmt.Errorf("format repository permissions: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\nRole and permission matrix (selected is reported by the server):\n%s\n", cmdutil.EscapeTSVField(repository.String()), formatted)
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output repository and the complete permission matrix as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; explicit repository takes precedence over Git context."},
		Output:      "text,json", Effects: "read",
		Notes: []string{"Requires authentication to query the current account's permissions; never falls back to anonymous access.", "JSON contains repository and permissions; permissions preserves role_info, resource_trees/actions, false selections, and unknown fields. Role names and request success do not grant or imply other permissions."},
	})
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}
