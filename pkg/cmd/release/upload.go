package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// assetTypeAttach is the value of ReleaseAsset.Type for a deletable
// uploaded attachment. Source archives generated automatically by the
// API use a different type and cannot be removed.
const assetTypeAttach = "attach"

type uploadOptions struct {
	Name         string
	SkipExisting bool
	Overwrite    bool
	Timeout      time.Duration
}

func newCmdReleaseUpload(f *cmdutil.Factory) *cobra.Command {
	opts := uploadOptions{}

	cmd := &cobra.Command{
		Use:   "upload [<owner>/<repo>] <tag> <file>",
		Short: "Upload an attachment to a release",
		Long:  `Upload a local file as an attachment to an existing release identified by its tag.`,
		Example: `  ag release upload owner/repo v1.0.0 ./dist/app.tar.gz
  ag release upload owner/repo v1.0.0 ./build/app.zip --name app-v1.zip
  ag release upload owner/repo v1.0.0 ./new.tar.gz --overwrite
  ag release upload owner/repo v1.0.0 ./existing.tar.gz --skip-existing`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReleaseUpload(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Remote attachment name (defaults to the local file's base name)")
	cmd.Flags().BoolVar(&opts.SkipExisting, "skip-existing", false, "Do nothing and report success if an attachment with the same name already exists")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Delete an existing attachment with the same name before uploading")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", defaultReleaseTransferTimeout, "Maximum attachment transfer time (0 disables the limit)")
	return cmd
}

func runReleaseUpload(cmd *cobra.Command, f *cmdutil.Factory, opts uploadOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(remaining[0])
	file := remaining[1]

	if opts.SkipExisting && opts.Overwrite {
		return fmt.Errorf("--skip-existing and --overwrite are mutually exclusive")
	}
	if err := validateReleaseTransferTimeout(opts.Timeout); err != nil {
		return err
	}
	if tag == "" {
		return fmt.Errorf("release tag is required")
	}

	remoteName := strings.TrimSpace(opts.Name)
	if !cmd.Flags().Changed("name") {
		remoteName = filepath.Base(file)
	}
	if strings.TrimSpace(remoteName) == "" {
		return fmt.Errorf("invalid attachment name: %q (must not be empty)", remoteName)
	}

	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat upload file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("upload file %q is not a regular file", file)
	}
	handle, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open upload file: %w", err)
	}
	defer handle.Close()

	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}
	client, err := newAPIClient(f, token)
	if err != nil {
		return err
	}

	release, err := api.GetReleaseByTag(client, repository.Owner, repository.Name, tag)
	if err != nil {
		return fmt.Errorf("failed to get release before uploading: %w", err)
	}

	var matches []api.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.Name == remoteName {
			matches = append(matches, asset)
		}
	}

	if len(matches) == 0 {
		upload, err := getUploadTarget(client, repository, tag, remoteName)
		if err != nil {
			return err
		}
		return uploadAndReport(cmd, client, upload, tag, remoteName, handle, opts.Timeout)
	}

	if !opts.SkipExisting && !opts.Overwrite {
		return fmt.Errorf("attachment %q already exists on release %q; pass --skip-existing or --overwrite to proceed",
			remoteName, tag)
	}
	if opts.SkipExisting {
		fmt.Fprintf(cmd.OutOrStdout(), "Skipped existing attachment %s on release %s\n", remoteName, tag)
		return nil
	}

	if len(matches) != 1 {
		return fmt.Errorf("found %d attachments named %q on release %q; --overwrite requires exactly one",
			len(matches), remoteName, tag)
	}
	asset := matches[0]
	if asset.Type != assetTypeAttach {
		return fmt.Errorf("attachment %q on release %q is type %q, not %q; cannot overwrite",
			remoteName, tag, asset.Type, assetTypeAttach)
	}
	if asset.ID <= 0 {
		return fmt.Errorf("attachment %q on release %q has invalid id %d; cannot overwrite",
			remoteName, tag, asset.ID)
	}

	// Resolve the upload target before deleting the old attachment.
	upload, err := getUploadTarget(client, repository, tag, remoteName)
	if err != nil {
		return err
	}

	// DELETE is not retried. Reconcile a lost response before uploading.
	if deleteErr := api.DeleteReleaseAttachment(client, repository.Owner, repository.Name, tag, asset.ID); deleteErr != nil {
		deleted, reconcileErr := reconcileAttachmentDeletion(client, repository, tag, asset)
		if reconcileErr != nil {
			return fmt.Errorf("failed to delete existing attachment %q before overwrite: %w; could not reconcile the release state: %v",
				remoteName, deleteErr, reconcileErr)
		}
		if !deleted {
			return fmt.Errorf("failed to delete existing attachment %q before overwrite: %w", remoteName, deleteErr)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Delete response was unsuccessful, but attachment %s is absent; continuing overwrite\n", remoteName)
	}

	if err := uploadAndReport(cmd, client, upload, tag, remoteName, handle, opts.Timeout); err != nil {
		return fmt.Errorf("upload after overwrite failed (old attachment %q was already deleted): %w", remoteName, err)
	}
	return nil
}

// getUploadTarget resolves the external object-store destination while all
// existing attachments are still untouched.
func getUploadTarget(
	client *api.Client,
	repository cmdutil.Repository,
	tag, remoteName string,
) (api.ReleaseUploadURL, error) {
	upload, err := api.GetReleaseUploadURL(client, repository.Owner, repository.Name, tag, remoteName)
	if err != nil {
		return api.ReleaseUploadURL{}, fmt.Errorf("failed to get upload url: %w", err)
	}
	return upload, nil
}

// reconcileAttachmentDeletion checks an ambiguous DELETE result. It reports a
// confirmed deletion only when the old attachment ID and every same-name asset
// are absent; a concurrent same-name replacement is left untouched.
func reconcileAttachmentDeletion(
	client *api.Client,
	repository cmdutil.Repository,
	tag string,
	asset api.ReleaseAsset,
) (bool, error) {
	release, err := api.GetReleaseByTag(client, repository.Owner, repository.Name, tag)
	if err != nil {
		return false, err
	}

	for _, current := range release.Assets {
		if current.ID == asset.ID {
			return false, nil
		}
	}
	for _, current := range release.Assets {
		if current.Name == asset.Name {
			return false, fmt.Errorf("another attachment named %q appeared after the delete attempt", asset.Name)
		}
	}
	return true, nil
}

// uploadAndReport streams the file body to a previously resolved upload target
// and prints the success message on completion.
func uploadAndReport(
	cmd *cobra.Command,
	client *api.Client,
	upload api.ReleaseUploadURL,
	tag, remoteName string,
	handle *os.File,
	timeout time.Duration,
) error {
	ctx, cancel := releaseTransferContext(cmd.Context(), timeout)
	defer cancel()
	if err := api.UploadReleaseAsset(ctx, client, upload, handle); err != nil {
		return fmt.Errorf("failed to upload attachment: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Uploaded attachment %s to release %s\n", remoteName, tag)
	return nil
}
