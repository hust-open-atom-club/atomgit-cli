package release

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdReleaseView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "view [<owner>/<repo>] <tag>",
		Short:   "View a release by tag",
		Long:    `Show details of a single release identified by its tag.`,
		Example: `  ag release view owner/repo v1.0.0`,
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			tag := strings.TrimSpace(remaining[0])
			if tag == "" {
				return fmt.Errorf("release tag is required")
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			release, err := api.GetReleaseByTag(client, repository.Owner, repository.Name, tag)
			if err != nil {
				return fmt.Errorf("failed to view release %q: %w", tag, err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name: %s\n", release.Name)
			fmt.Fprintf(out, "Tag: %s\n", release.TagName)
			fmt.Fprintf(out, "Target: %s\n", release.TargetCommitish)
			fmt.Fprintf(out, "Status: %s\n", releaseStatus(release))
			fmt.Fprintf(out, "Created: %s\n", release.CreatedAt)
			fmt.Fprintf(out, "Author: %s\n", releaseAuthorDisplay(release.Author))
			fmt.Fprintf(out, "Body: %s\n", release.Body)
			fmt.Fprintln(out, "Assets:")
			if len(release.Assets) == 0 {
				fmt.Fprintln(out, "  None")
			} else {
				for _, a := range release.Assets {
					fmt.Fprintf(out, "  %s\t%s\t%s\n", a.Name, a.Type, a.BrowserDownloadURL)
				}
			}
			return nil
		},
	}

	return cmd
}

// releaseAuthorDisplay renders the author line; login is the canonical handle
// and name, when present, adds a human-readable label.
func releaseAuthorDisplay(a api.ReleaseAuthor) string {
	if a.Login == "" && a.Name == "" {
		return ""
	}
	if a.Name != "" && a.Login != "" && a.Name != a.Login {
		return a.Name + " (" + a.Login + ")"
	}
	if a.Login != "" {
		return a.Login
	}
	return a.Name
}
