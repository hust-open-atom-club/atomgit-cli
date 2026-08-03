package workflow

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdWorkflow creates the root workflow command tree.
func NewCmdWorkflow(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage AtomGit Actions workflows",
		Long:  `List and run AtomGit Actions workflows.`,
		Example: `  ag workflow list owner/repo
  ag workflow run owner/repo 12345 --ref main
  ag workflow run owner/repo ci.yml -f env=production`,
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdRun(f))
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func newActionsClient(f *cmdutil.Factory, token string) (*actions.Client, error) {
	if f == nil || f.HttpClient == nil {
		return actions.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return actions.NewClientWithHTTPClient(token, httpClient), nil
}

func requireToken(f *cmdutil.Factory) (string, error) {
	if f == nil || f.Config == nil {
		return "", fmt.Errorf("configuration is unavailable")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return "", fmt.Errorf("not authenticated: %w", err)
	}
	return token, nil
}
