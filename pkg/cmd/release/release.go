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
		Long: `List, view, create, and edit repository releases on AtomGit, and upload release attachments.

This command group provides general Release management primitives. It does not
validate tags, build artifacts, or run the atomgit-cli release pipeline tracked
by #18.`,
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
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}
