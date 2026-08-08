package release

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdRelease creates the `ag release` command tree for managing repository
// releases.
func NewCmdRelease(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage repository releases",
		Long: `List, view, create, and edit repository releases on AtomGit, and upload or download release attachments.

The repository's automated release pipeline validates tags and artifacts, then
uses these primitives through "make publish VERSION=vX.Y.Z NOTES_FILE=notes.md".`,
		Example: `  ag release list owner/repo
  ag release view owner/repo v1.0.0
  ag release create owner/repo v1.0.0 --name "Version 1.0.0" --body "Release notes"
  ag release upload owner/repo v1.0.0 ./dist/app.tar.gz`,
	}

	cmd.AddCommand(newCmdReleaseList(f))
	cmd.AddCommand(newCmdReleaseView(f))
	cmd.AddCommand(newCmdReleaseCreate(f))
	cmd.AddCommand(newCmdReleaseEdit(f))
	cmd.AddCommand(newCmdReleaseUpload(f))
	cmd.AddCommand(newCmdReleaseDownload(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}
