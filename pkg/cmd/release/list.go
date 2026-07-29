package release

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdReleaseList(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List repository releases",
		Long:    `List releases for a repository, ordered most recent first.`,
		Example: `  ag release list owner/repo --limit 50`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			releases, err := api.ListReleases(client, repository.Owner, repository.Name, limit)
			if err != nil {
				return fmt.Errorf("failed to list releases: %w", err)
			}

			out := cmd.OutOrStdout()
			if len(releases) == 0 {
				fmt.Fprintln(out, "No releases found")
				return nil
			}
			for _, r := range releases {
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", r.TagName, r.Name, releaseStatus(r), r.CreatedAt)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of releases to list")
	return cmd
}

// releaseStatus derives a displayable status from the prerelease flag and the
// server-reported release_status field, following the release contract:
//   - Prerelease=true or ReleaseStatus==ReleaseStatusPre  => "prerelease"
//   - ReleaseStatus==ReleaseStatusLatest                  => "latest"
//   - any other value (empty, "none", unknown)            => "release"
func releaseStatus(r api.Release) string {
	if r.Prerelease || r.ReleaseStatus == api.ReleaseStatusPre {
		return "prerelease"
	}
	if r.ReleaseStatus == api.ReleaseStatusLatest {
		return "latest"
	}
	return "release"
}
