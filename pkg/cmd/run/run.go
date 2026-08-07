package run

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRun(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Inspect AtomGit Actions workflow runs",
		Long: `List and inspect AtomGit Actions workflow runs, jobs, logs, and artifacts.

This command is read-only. Workflow dispatch, rerun, cancel, and delete
operations are not supported.`,
	}

	cmd.AddCommand(newCmdRunList(f))
	cmd.AddCommand(newCmdRunView(f))
	return cmd
}

func newActionsClient(f *cmdutil.Factory, token string) (*actions.Client, error) {
	if f.HttpClient == nil {
		return actions.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return actions.NewClientWithHTTPClient(token, httpClient), nil
}

func parseRepository(value string) (string, string, error) {
	parsed, err := cmdutil.ParseRepository(value)
	if err != nil {
		return "", "", err
	}
	return parsed.Owner, parsed.Name, nil
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
