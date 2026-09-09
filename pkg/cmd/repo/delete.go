package repo

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdRepoDelete(f *cmdutil.Factory) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [<repository>]",
		Short: "Delete a repository",
		Long: `Delete a repository from AtomGit.

This command permanently deletes a repository. This action cannot be undone.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.`,
		Example: `  # Delete a repository (with confirmation)
  ag repo delete my-project

  # Delete a repository without confirmation
  ag repo delete my-project --yes

  # Delete a repository in an organization
  ag repo delete my-org/my-project --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner, repoName string
			var err error
			if len(args) == 0 {
				repository, err := cmdutil.ResolveRepository(f, "")
				if err != nil {
					return err
				}
				owner, repoName = repository.Owner, repository.Name
			} else {
				if strings.Contains(args[0], "/") {
					owner, repoName, err = parseRepositoryName(args[0], "")
				} else {
					currentUser, userErr := f.Config.GetUser()
					if userErr != nil {
						return cmdutil.AuthenticationError(userErr)
					}
					owner, repoName, err = parseRepositoryName(args[0], currentUser)
				}
				if err != nil {
					return err
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

			out := cmd.OutOrStdout()

			// Confirm deletion unless --yes flag is used
			if !force {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Are you sure you want to delete %s/%s? This action cannot be undone. [y/N] ", owner, repoName))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Deletion cancelled.")
					return nil
				}
			}

			// Delete the repository
			path := fmt.Sprintf("/repos/%s/%s", owner, repoName)
			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete repository: %w", err)
			}

			fmt.Fprintf(out, "✓ Deleted repository %s/%s\n", owner, repoName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "yes", "y", false, "Skip confirmation prompt")
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}
