package repo

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdRepoDelete(f *cmdutil.Factory) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [<owner>/]<repo>",
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
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			currentUser, err := f.Config.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			owner, repoName, err := parseRepositoryName(args[0], currentUser)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			// Confirm deletion unless --yes flag is used
			if !force {
				fmt.Printf("Are you sure you want to delete %s/%s? This action cannot be undone. [y/N] ", owner, repoName)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			// Delete the repository
			path := fmt.Sprintf("/repos/%s/%s", owner, repoName)
			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete repository: %w", err)
			}

			fmt.Printf("✓ Deleted repository %s/%s\n", owner, repoName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
