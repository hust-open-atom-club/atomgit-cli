package key

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const defaultSSHKeyListLimit = 1000

type ListOptions struct {
	Limit int
}

func newCmdSSHKeyList(f *cmdutil.Factory) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys registered with your AtomGit account",
		Example: `  ag ssh-key list
  ag ssh-key list --limit 200`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.OutOrStdout(), f, opts)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", defaultSSHKeyListLimit, "Maximum number of SSH keys to list")
	return cmd
}

func runList(out io.Writer, f *cmdutil.Factory, opts *ListOptions) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	client, err := authenticatedAPIClient(f)
	if err != nil {
		return err
	}

	keys, err := api.GetPaginated[api.SSHKey](client, opts.Limit, func(page, perPage int) string {
		return fmt.Sprintf("/user/keys?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Fprintln(out, "No SSH keys found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tFINGERPRINT\tCREATED")
	for _, sshKey := range keys {
		fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\n",
			sshKey.ID,
			displaySSHKeyValue(sshKey.Title),
			displaySSHKeyValue(sshKeyFingerprint(sshKey)),
			displaySSHKeyValue(sshKey.CreatedAt),
		)
	}
	return w.Flush()
}

func authenticatedAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return newAPIClient(f, token)
}

func sshKeyFingerprint(sshKey api.SSHKey) string {
	if fingerprint := strings.TrimSpace(sshKey.Fingerprint); fingerprint != "" {
		return fingerprint
	}

	fields := strings.Fields(sshKey.Key)
	if len(fields) < 2 {
		return "-"
	}
	keyBytes, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "-"
	}

	digest := sha256.Sum256(keyBytes)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])
}

func displaySSHKeyValue(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	return value
}
