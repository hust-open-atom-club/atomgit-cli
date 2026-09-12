package runner

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdRunner creates the read-only host runner command tree.
func NewCmdRunner(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Inspect AtomGit Actions host runners",
		Long:  "List repository-specific and shared AtomGit Actions host runners. These commands are read-only.",
		Example: `  ag runner list owner/repo
  ag runner shared owner/repo --json`,
	}
	cmd.AddCommand(newCmdList(f, false))
	cmd.AddCommand(newCmdList(f, true))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

const maxRunnersPerPage = 100

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
		return "", cmdutil.AuthenticationError(err)
	}
	return token, nil
}
