package release

import (
	"fmt"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type editOptions struct {
	Name       string
	Body       string
	BodyFile   string
	Prerelease bool
	Latest     bool
}

func newCmdReleaseEdit(f *cmdutil.Factory) *cobra.Command {
	opts := editOptions{}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <tag>",
		Short: "Edit a release",
		Long:  `Edit an existing release identified by its tag.`,
		Example: `  ag release edit owner/repo v1.0.0 --name "First Release"
  ag release edit owner/repo v1.0.0 --latest --body-file notes.md
  ag release edit owner/repo v1.0.0-rc --prerelease`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReleaseEdit(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "New release name")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New release body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Path to file containing new release body")
	cmd.Flags().BoolVar(&opts.Prerelease, "prerelease", false, "Set release status to prerelease (release_status=pre)")
	cmd.Flags().BoolVar(&opts.Latest, "latest", false, "Set release status to latest (release_status=latest)")
	return cmd
}

func runReleaseEdit(cmd *cobra.Command, f *cmdutil.Factory, opts editOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(remaining[0])
	if tag == "" {
		return fmt.Errorf("release tag is required")
	}

	nameChanged := cmd.Flags().Changed("name")
	bodyChanged := cmd.Flags().Changed("body")
	bodyFileChanged := cmd.Flags().Changed("body-file")
	prereleaseChanged := cmd.Flags().Changed("prerelease")
	latestChanged := cmd.Flags().Changed("latest")

	if !nameChanged && !bodyChanged && !bodyFileChanged && !prereleaseChanged && !latestChanged {
		return fmt.Errorf("at least one of --name, --body, --body-file, --prerelease or --latest must be provided")
	}
	if bodyChanged && bodyFileChanged {
		return fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	if prereleaseChanged && latestChanged {
		return fmt.Errorf("--prerelease and --latest are mutually exclusive")
	}

	// An explicitly false status flag would otherwise produce a no-op PATCH.
	if prereleaseChanged && !opts.Prerelease {
		return fmt.Errorf("--prerelease=false is not supported; omit the flag to leave status unchanged")
	}
	if latestChanged && !opts.Latest {
		return fmt.Errorf("--latest=false is not supported; omit the flag to leave status unchanged")
	}
	if nameChanged && strings.TrimSpace(opts.Name) == "" {
		return fmt.Errorf("invalid name: %q (must not be empty)", opts.Name)
	}
	if bodyChanged && strings.TrimSpace(opts.Body) == "" {
		return fmt.Errorf("release body must not be empty")
	}

	fileBody := ""
	if bodyFileChanged {
		content, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		fileBody = string(content)
		if strings.TrimSpace(fileBody) == "" {
			return fmt.Errorf("release body must not be empty")
		}
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	// The API requires name and body on PATCH, including unchanged values.
	current, err := api.GetReleaseByTag(client, repository.Owner, repository.Name, tag)
	if err != nil {
		return fmt.Errorf("failed to get release before editing: %w", err)
	}

	finalName := current.Name
	if nameChanged {
		finalName = opts.Name
	}
	finalBody := current.Body
	if bodyChanged {
		finalBody = opts.Body
	} else if bodyFileChanged {
		finalBody = fileBody
	}

	request := api.UpdateReleaseRequest{Name: finalName, Body: finalBody}
	if prereleaseChanged {
		request.ReleaseStatus = api.ReleaseStatusPre
	}
	if latestChanged {
		request.ReleaseStatus = api.ReleaseStatusLatest
	}

	if _, err := api.UpdateRelease(client, repository.Owner, repository.Name, tag, request); err != nil {
		return fmt.Errorf("failed to edit release: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated release %s\n", tag)
	return nil
}
