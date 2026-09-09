package release

import (
	"fmt"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type createOptions struct {
	Name       string
	Body       string
	BodyFile   string
	Target     string
	Prerelease bool
}

func newCmdReleaseCreate(f *cmdutil.Factory) *cobra.Command {
	opts := createOptions{}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <tag>",
		Short: "Create a release",
		Long:  `Create a release for a repository identified by its tag. A non-empty release body is required by the AtomGit API.`,
		Example: `  ag release create owner/repo v1.0.0 --name "First" --body "Initial release"
  ag release create owner/repo v1.0.0-rc --prerelease --body-file notes.md`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReleaseCreate(cmd, f, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Release name (defaults to tag)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Release body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Path to file containing release body")
	cmd.Flags().StringVar(&opts.Target, "target", "", "Target commitish (branch or SHA)")
	cmd.Flags().BoolVar(&opts.Prerelease, "prerelease", false, "Mark release as a prerelease (release_status=pre)")
	return cmd
}

func runReleaseCreate(cmd *cobra.Command, f *cmdutil.Factory, opts createOptions, args []string) error {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(remaining[0])
	if tag == "" {
		return fmt.Errorf("release tag is required")
	}

	nameChanged := cmd.Flags().Changed("name")
	finalName := opts.Name
	if !nameChanged {
		finalName = tag
	} else if strings.TrimSpace(finalName) == "" {
		return fmt.Errorf("invalid name: %q (must not be empty)", finalName)
	}

	bodyChanged := cmd.Flags().Changed("body")
	bodyFileChanged := cmd.Flags().Changed("body-file")
	if bodyChanged && bodyFileChanged {
		return fmt.Errorf("--body and --body-file are mutually exclusive")
	}

	finalBody := opts.Body
	if bodyFileChanged {
		content, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		finalBody = string(content)
	}
	if !bodyChanged && !bodyFileChanged {
		return fmt.Errorf("release body is required; use --body or --body-file")
	}
	if strings.TrimSpace(finalBody) == "" {
		return fmt.Errorf("release body must not be empty")
	}
	targetChanged := cmd.Flags().Changed("target")
	target := strings.TrimSpace(opts.Target)
	if targetChanged && target == "" {
		return fmt.Errorf("invalid target: %q (must not be empty)", opts.Target)
	}

	request := api.CreateReleaseRequest{
		TagName: tag,
		Name:    finalName,
		Body:    finalBody,
	}
	if targetChanged {
		request.TargetCommitish = target
	}
	if opts.Prerelease {
		request.ReleaseStatus = api.ReleaseStatusPre
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	if _, err := api.CreateRelease(client, repository.Owner, repository.Name, request); err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created release %s\n", tag)
	return nil
}
