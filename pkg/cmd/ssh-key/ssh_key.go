package key

import (
	"fmt"
	"io"
	"os"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type AddOptions struct {
	KeyFile string
	Title   string
}

func NewCmdSSHKey(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key <command>",
		Short: "Manage SSH keys",
		Long:  `Manage SSH keys registered with your AtomGit account.`,
	}

	cmd.AddCommand(newCmdSSHKeyAdd(f))
	cmd.AddCommand(newCmdSSHKeyList(f))
	cmd.AddCommand(newCmdSSHKeyDelete(f))

	return cmd
}

func newCmdSSHKeyAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &AddOptions{}

	cmd := &cobra.Command{
		Use:   "add [<key-file>]",
		Short: "Add an SSH key to your AtomGit account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// Read from stdin
				opts.KeyFile = "-"
			} else {
				opts.KeyFile = args[0]
			}

			return runAdd(cmd.OutOrStdout(), f, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Title for the new key")

	return cmd
}

func runAdd(out io.Writer, f *cmdutil.Factory, opts *AddOptions) error {
	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	var keyReader io.Reader
	if opts.KeyFile == "-" {
		keyReader = os.Stdin
	} else {
		file, err := os.Open(opts.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to open key file: %w", err)
		}
		defer file.Close()
		keyReader = file
	}

	// Read key content
	keyBytes, err := io.ReadAll(keyReader)
	if err != nil {
		return fmt.Errorf("failed to read key: %w", err)
	}

	client, err := newAPIClient(f, token)
	if err != nil {
		return err
	}

	// AtomGit API endpoint for adding SSH keys
	// POST /api/v5/user/keys
	body := map[string]interface{}{
		"title": opts.Title,
		"key":   string(keyBytes),
	}

	var result map[string]interface{}
	if err := client.Post("/user/keys", body, &result); err != nil {
		return fmt.Errorf("failed to add SSH key: %w", err)
	}

	fmt.Fprintln(out, "✓ SSH key added to your account")
	return nil
}
