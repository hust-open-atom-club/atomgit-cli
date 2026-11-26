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
		Short: "Inspect and manage AtomGit Actions workflow runs and artifacts",
		Long: `List and inspect AtomGit Actions workflow runs, jobs, logs, and artifacts.

Cancel (or stop), rerun, and retry workflow runs after confirmation.
Artifacts can be deleted after confirmation.
Workflow run dispatch and deletion operations are not supported.`,
	}

	cmd.AddCommand(newCmdRunList(f))
	cmd.AddCommand(newCmdRunView(f))
	cmd.AddCommand(newCmdRunStepLog(f))
	cmd.AddCommand(newCmdRunArtifact(f))
	cmd.AddCommand(newCmdRunStop(f))
	cmd.AddCommand(newCmdRunCancel(f))
	cmd.AddCommand(newCmdRunRerun(f))
	cmd.AddCommand(newCmdRunRetry(f))
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
