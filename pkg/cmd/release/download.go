package release

import (
	"fmt"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// Auto-generated source archives have non-positive IDs and are not downloadable.
const downloadableAssetIDFloor = 1

type downloadOptions struct {
	Output    string
	Overwrite bool
	Timeout   time.Duration
}

func newCmdReleaseDownload(f *cmdutil.Factory) *cobra.Command {
	opts := downloadOptions{}

	cmd := &cobra.Command{
		Use:   "download [<owner>/<repo>] <tag> <asset>",
		Short: "Download an attachment from a release",
		Long: `Download a release attachment identified by its exact asset name to a local file.

The required -o/--output flag names the local destination path. When the
destination already exists the command fails unless --overwrite is given, in
which case the existing file is replaced. The attachment is streamed to a
temporary file in the destination directory and only installed at the target
path after the transfer completes; a transfer interruption leaves any existing
destination untouched.`,
		Example: `  ag release download owner/repo v1.0.0 app.tar.gz -o ./dist/app.tar.gz
  ag release download owner/repo v1.0.0 app.tar.gz --output ./existing.tar.gz --overwrite`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReleaseDownload(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Local file path to write the attachment to (required)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Replace an existing local file at --output")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", defaultReleaseTransferTimeout, "Maximum attachment transfer time (0 disables the limit)")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func runReleaseDownload(cmd *cobra.Command, f *cmdutil.Factory, opts downloadOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(remaining[0])
	assetName := strings.TrimSpace(remaining[1])
	if tag == "" {
		return fmt.Errorf("release tag is required")
	}
	if assetName == "" {
		return fmt.Errorf("attachment name is required")
	}
	if err := validateReleaseTransferTimeout(opts.Timeout); err != nil {
		return err
	}

	output, err := cmdutil.PreflightDownloadDestination(opts.Output, opts.Overwrite)
	if err != nil {
		return err
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
		return fmt.Errorf("failed to get release before downloading: %w", err)
	}

	var matches []api.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			matches = append(matches, asset)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("attachment %q not found on release %q", assetName, tag)
	}
	if len(matches) != 1 {
		return fmt.Errorf("found %d attachments named %q on release %q; need exactly one",
			len(matches), assetName, tag)
	}
	asset := matches[0]
	if asset.Type != assetTypeAttach {
		return fmt.Errorf("attachment %q on release %q is type %q, not %q; cannot download",
			assetName, tag, asset.Type, assetTypeAttach)
	}
	if asset.ID < downloadableAssetIDFloor {
		return fmt.Errorf("attachment %q on release %q has invalid id %d; cannot download",
			assetName, tag, asset.ID)
	}

	ctx, cancel := releaseTransferContext(cmd.Context(), opts.Timeout)
	defer cancel()
	body, err := api.DownloadReleaseAttachment(ctx, client, repository.Owner, repository.Name, tag, assetName)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}
	defer body.Close()

	if _, err := cmdutil.WriteDownload(output, body, opts.Overwrite); err != nil {
		return fmt.Errorf("failed to write download: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Downloaded attachment %s from release %s to %s\n", assetName, tag, output)
	return nil
}
