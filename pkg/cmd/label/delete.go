package label

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdLabelDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <name>",
		Short: "Delete a repository label",
		Long: `Delete a repository label from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.`,
		Example: `  ag label delete owner/repo obsolete
  ag label delete owner/repo obsolete --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(remaining[0])
			if err := validateLabelName(name); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if !yes {
				fmt.Fprintf(out, "Delete label %q from %s? [y/N] ", name, repository.String())
				var response string
				if _, err := fmt.Fscan(cmd.InOrStdin(), &response); err != nil && err != io.EOF {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
					fmt.Fprintln(out, "Deletion cancelled.")
					return nil
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

			path := fmt.Sprintf("/repos/%s/%s/labels/%s", repository.Owner, repository.Name, url.PathEscape(name))
			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete label: %w", err)
			}

			fmt.Fprintf(out, "Deleted label %q\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
