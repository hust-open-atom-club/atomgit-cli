package workflow

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
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

const maxWorkflowsPerPage = 100

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

// listAllWorkflows fetches every workflow in a repository, following pages
// until the API returns a short or empty page (or the advertised total count
// is reached).
func listAllWorkflows(client *actions.Client, owner, repo string) ([]actions.Workflow, error) {
	workflows := make([]actions.Workflow, 0, maxWorkflowsPerPage)
	for page := 1; ; page++ {
		resp, err := client.ListWorkflows(owner, repo, actions.ListWorkflowsOptions{
			Page:    page,
			PerPage: maxWorkflowsPerPage,
		})
		if err != nil {
			return nil, err
		}
		if len(resp.Workflows) == 0 {
			break
		}
		workflows = append(workflows, resp.Workflows...)
		if len(resp.Workflows) < maxWorkflowsPerPage || (resp.TotalCount > 0 && len(workflows) >= resp.TotalCount) {
			break
		}
	}
	return workflows, nil
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

func newAPIClient(f *cmdutil.Factory, token string) (*api.Client, error) {
	if f == nil || f.HttpClient == nil {
		return api.NewClient(token), nil
	}

	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return api.NewClientWithHTTPClient(token, httpClient), nil
}

func resolveDefaultBranch(f *cmdutil.Factory, token, owner, repo string) (string, error) {
	client, err := newAPIClient(f, token)
	if err != nil {
		return "", err
	}
	var repoInfo api.Repository
	if err := client.Get(fmt.Sprintf("/repos/%s/%s", owner, repo), &repoInfo); err != nil {
		return "", err
	}
	if strings.TrimSpace(repoInfo.DefaultBranch) == "" {
		return "", fmt.Errorf("empty default branch for %s/%s", owner, repo)
	}
	return repoInfo.DefaultBranch, nil
}
