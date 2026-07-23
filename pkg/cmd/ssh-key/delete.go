package key

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdSSHKeyDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an SSH key from your AtomGit account",
		Long: `Delete an SSH key from your AtomGit account.

The target key is retrieved before deletion. By default, you will be prompted
to confirm the deletion. Use --yes to skip the confirmation prompt.`,
		Example: `  ag ssh-key delete 123
  ag ssh-key delete 123 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID, err := parseSSHKeyID(args[0])
			if err != nil {
				return err
			}

			client, err := authenticatedAPIClient(f)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/user/keys/%d", keyID)
			var sshKey api.SSHKey
			if err := client.Get(path, &sshKey); err != nil {
				return fmt.Errorf("failed to get SSH key %d: %w", keyID, err)
			}
			sshKey.ID = keyID

			if !yes {
				confirmed, err := confirmSSHKeyDelete(cmd, sshKey)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "SSH key deletion cancelled.")
					return nil
				}
			}

			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete SSH key %d: %w", keyID, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted SSH key %d (%s).\n", keyID, displaySSHKeyValue(sshKey.Title))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func parseSSHKeyID(value string) (int64, error) {
	value = strings.TrimSpace(value)
	keyID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || keyID <= 0 {
		return 0, fmt.Errorf("invalid SSH key ID %q: must be a positive integer", value)
	}
	return keyID, nil
}

func confirmSSHKeyDelete(cmd *cobra.Command, sshKey api.SSHKey) (bool, error) {
	fmt.Fprintf(
		cmd.OutOrStdout(),
		"Delete SSH key %d (%s, %s)? [y/N] ",
		sshKey.ID,
		displaySSHKeyValue(sshKey.Title),
		displaySSHKeyValue(sshKeyFingerprint(sshKey)),
	)

	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
