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
		Short: "Inspect AtomGit Actions workflow runs and artifacts",
		Long: `List and inspect AtomGit Actions workflow runs, jobs, logs, and artifacts.

Artifacts can also be deleted after confirmation. Workflow run dispatch,
rerun, cancel, and deletion operations are not supported.`,
	}

	cmd.AddCommand(newCmdRunList(f))
	cmd.AddCommand(newCmdRunView(f))
	cmd.AddCommand(newCmdRunStepLog(f))
	cmd.AddCommand(newCmdRunArtifact(f))
	return cmd
}

func newActionsClient(f *cmdutil.Factory, token string) (*actions.Client, error) {
	return f.NewActionsClient(token)
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
		return "", cmdutil.AuthenticationError(err)
	}
	return token, nil
}
