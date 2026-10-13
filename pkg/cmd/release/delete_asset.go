package release

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type deleteAssetOptions struct{ Yes bool }

func newCmdReleaseDeleteAsset(f *cmdutil.Factory) *cobra.Command {
	opts := deleteAssetOptions{}
	cmd := &cobra.Command{
		Use:   "delete-asset [<owner>/<repo>] <tag> <asset-id>",
		Short: "Delete an attachment from a release",
		Long: `Delete one uploaded attachment by its ID after checking that it belongs to the specified release.
Use release view --json to find attachment IDs. Source archives cannot be deleted.
By default, confirmation is required; use --yes to skip the prompt.`,
		Example: `  ag-cli release delete-asset owner/repo v1.0.0 42
  ag-cli release delete-asset v1.0.0 attachment-id --yes`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error { return runReleaseDeleteAsset(cmd, f, opts, args) },
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 3, Description: "Optional owner/repo, release tag, and attachment ID"},
		Output:      "text", Effects: "write",
		Notes: []string{"Checks release membership before deletion; confirmation required unless --yes is supplied", "Deletes only the selected uploaded attachment; DELETE is not retried"},
	})
	return cmd
}

func runReleaseDeleteAsset(cmd *cobra.Command, f *cmdutil.Factory, opts deleteAssetOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(remaining[0])
	id := remaining[1]
	if tag == "" {
		return fmt.Errorf("release tag is required")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("attachment id is required")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}
	release, err := api.GetReleaseByTag(client, repository.Owner, repository.Name, tag)
	if err != nil {
		return fmt.Errorf("failed to get release %q before deleting attachment: %w", tag, err)
	}
	if release.TagName != tag {
		return fmt.Errorf("release tag mismatch: requested %q, received %q", tag, release.TagName)
	}
	var target *api.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.AttachmentID() == id {
			if target != nil {
				return fmt.Errorf("multiple assets have id %q on release %q; cannot delete", id, tag)
			}
			target = &asset
		}
	}
	if target == nil {
		return fmt.Errorf("attachment %q does not belong to release %q in %s/%s", id, tag, repository.Owner, repository.Name)
	}
	if target.Type != assetTypeAttach {
		return fmt.Errorf("asset %q is type %q, not an uploaded attachment; cannot delete", id, target.Type)
	}
	identity := fmt.Sprintf("attachment %q (ID %q) from release %q in %s/%s", target.Name, id, tag, repository.Owner, repository.Name)
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Target: %s\n", identity); err != nil {
		return fmt.Errorf("show attachment deletion target: %w", err)
	}
	if !opts.Yes {
		confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Delete %s? This action cannot be undone. [y/N] ", identity))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
			return nil
		}
	}
	if err := api.DeleteReleaseAttachmentByID(client, repository.Owner, repository.Name, tag, id); err != nil {
		return fmt.Errorf("failed to delete %s: %w", identity, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s\n", identity)
	return nil
}
