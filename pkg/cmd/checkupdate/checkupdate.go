package checkupdate

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	projectOwner = "hust-open-atom-club"
	projectRepo  = "atomgit-cli"
	releaseLimit = 100
)

type status string

const (
	statusUpdateAvailable status = "update available"
	statusUpToDate        status = "up to date"
	statusNewer           status = "current version is newer"
)

func NewCmdCheckUpdate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-update",
		Short: "Check for a newer AtomGit CLI release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			current := internalversion.Get().Version
			if err := validateComparableVersion(current); err != nil {
				return err
			}
			client, err := publicAPIClient(f)
			if err != nil {
				return err
			}
			releases, err := api.ListReleases(client, projectOwner, projectRepo, releaseLimit)
			if err != nil {
				return fmt.Errorf("check AtomGit CLI releases: %w", err)
			}
			latest, err := selectLatestStableRelease(releases)
			if err != nil {
				return err
			}
			result, err := compareVersions(current, latest)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"Current version: %s\nLatest release: %s\nStatus: %s\n",
				current,
				latest,
				result,
			)
			return err
		},
	}
	return cmd
}

func publicAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	if f == nil || f.HttpClient == nil {
		return api.NewClient(""), nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("create update HTTP client: %w", err)
	}
	if httpClient == nil {
		return api.NewClient(""), nil
	}
	return api.NewClientWithHTTPClient("", httpClient), nil
}

func selectLatestStableRelease(releases []api.Release) (string, error) {
	latest := ""
	for _, release := range releases {
		candidate := normalizeVersion(release.TagName)
		if release.Draft ||
			release.Prerelease ||
			release.ReleaseStatus == api.ReleaseStatusPre ||
			!semver.IsValid(candidate) ||
			semver.Prerelease(candidate) != "" {
			continue
		}
		if latest == "" || semver.Compare(candidate, latest) > 0 {
			latest = candidate
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no stable AtomGit CLI release found")
	}
	return latest, nil
}

func compareVersions(current, latest string) (status, error) {
	current = normalizeVersion(current)
	if err := validateComparableVersion(current); err != nil {
		return "", err
	}
	switch semver.Compare(current, latest) {
	case -1:
		return statusUpdateAvailable, nil
	case 0:
		return statusUpToDate, nil
	default:
		return statusNewer, nil
	}
}

func validateComparableVersion(value string) error {
	value = normalizeVersion(value)
	if strings.Contains(strings.ToLower(value), "dirty") || !semver.IsValid(value) {
		return fmt.Errorf("current version %q is not a comparable semantic version", value)
	}
	return nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && value[0] != 'v' {
		return "v" + value
	}
	return value
}
