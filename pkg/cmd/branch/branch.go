package branch

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type repositoryRef struct {
	Owner string
	Repo  string
}

func NewCmdBranch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage remote branches",
		Long:  `List, view, create, and delete AtomGit remote branches.`,
		Example: `  ag branch list owner/repo
  ag branch view owner/repo main
  ag branch create owner/repo feature/foo --ref main
  ag branch delete owner/repo feature/foo`,
	}

	cmd.AddCommand(newCmdBranchList(f))
	cmd.AddCommand(newCmdBranchView(f))
	cmd.AddCommand(newCmdBranchCreate(f))
	cmd.AddCommand(newCmdBranchDelete(f))

	return cmd
}

func authenticatedClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated: %w", err)
	}
	return newAPIClient(f, token)
}

func parseRepositoryArg(repository string) (repositoryRef, error) {
	owner, repo, err := parseRepository(repository)
	if err != nil {
		return repositoryRef{}, err
	}
	return repositoryRef{Owner: owner, Repo: repo}, nil
}

func branchPath(repository repositoryRef, branchName string) string {
	return fmt.Sprintf("/repos/%s/%s/branches/%s", repository.Owner, repository.Repo, escapePathSegment(branchName))
}

func newCmdBranchList(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list <owner>/<repo>",
		Short:   "List remote branches",
		Example: `  ag branch list owner/repo --limit 50`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}

			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}

			branches, err := api.GetPaginated[api.Branch](client, limit, func(page, perPage int) string {
				return fmt.Sprintf("/repos/%s/%s/branches?page=%d&per_page=%d", repository.Owner, repository.Repo, page, perPage)
			})
			if err != nil {
				return fmt.Errorf("failed to list branches for %s/%s: %w", repository.Owner, repository.Repo, err)
			}

			for _, branch := range branches {
				printBranchSummary(cmd.OutOrStdout(), branch)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of branches to list")
	return cmd
}

func newCmdBranchView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "view <owner>/<repo> <branch>",
		Short:   "View a remote branch",
		Example: `  ag branch view owner/repo main`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}

			branchName := args[1]
			var branch api.Branch
			if err := client.Get(branchPath(repository, branchName), &branch); err != nil {
				return fmt.Errorf("failed to view branch %q in %s/%s: %w", branchName, repository.Owner, repository.Repo, err)
			}

			printBranchDetail(cmd.OutOrStdout(), branch)
			return nil
		},
	}
	return cmd
}

func newCmdBranchCreate(f *cmdutil.Factory) *cobra.Command {
	var sourceRef string

	cmd := &cobra.Command{
		Use:     "create <owner>/<repo> <branch> --ref <ref>",
		Short:   "Create a remote branch",
		Example: `  ag branch create owner/repo feature/foo --ref main`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[1]) == "" {
				return fmt.Errorf("branch name is required")
			}
			if strings.TrimSpace(sourceRef) == "" {
				return fmt.Errorf("source ref is required")
			}

			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}

			branchName := args[1]
			request := api.BranchRequest{
				BranchName: branchName,
				Refs:       sourceRef,
			}
			var created api.Branch
			if err := client.Post(fmt.Sprintf("/repos/%s/%s/branches", repository.Owner, repository.Repo), request, &created); err != nil {
				return fmt.Errorf("failed to create branch %q in %s/%s from %q: %w", branchName, repository.Owner, repository.Repo, sourceRef, err)
			}
			if created.Name == "" {
				created.Name = branchName
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created branch %s\n", created.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceRef, "ref", "", "Source ref to create the branch from")
	return cmd
}

func newCmdBranchDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <owner>/<repo> <branch>",
		Short: "Delete a remote branch",
		Example: `  ag branch delete owner/repo feature/foo
  ag branch delete owner/repo feature/foo --yes`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}
			branchName := args[1]
			if strings.TrimSpace(branchName) == "" {
				return fmt.Errorf("branch name is required")
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}

			var repo api.Repository
			repoPath := fmt.Sprintf("/repos/%s/%s", repository.Owner, repository.Repo)
			if err := client.Get(repoPath, &repo); err != nil {
				return fmt.Errorf("failed to read repository %s/%s: %w", repository.Owner, repository.Repo, err)
			}

			var branch api.Branch
			if err := client.Get(branchPath(repository, branchName), &branch); err != nil {
				return fmt.Errorf("failed to view branch %q in %s/%s: %w", branchName, repository.Owner, repository.Repo, err)
			}

			if repo.DefaultBranch == branchName || branch.Default.Bool() {
				return fmt.Errorf("cannot delete default branch %q", branchName)
			}
			if branch.Protected.Bool() {
				return fmt.Errorf("cannot delete protected branch %q", branchName)
			}

			if !yes {
				confirmed, err := confirmDelete(cmd.InOrStdin(), cmd.OutOrStdout(), repository, branchName)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Branch deletion cancelled")
					return nil
				}
			}

			if err := client.Delete(branchPath(repository, branchName)); err != nil {
				return fmt.Errorf("failed to delete branch %q in %s/%s: %w", branchName, repository.Owner, repository.Repo, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted branch %s\n", branchName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm branch deletion without prompting")
	return cmd
}

func confirmDelete(in io.Reader, out io.Writer, repository repositoryRef, branchName string) (bool, error) {
	fmt.Fprintf(out, "Delete branch %s from %s/%s? [y/N] ", branchName, repository.Owner, repository.Repo)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

func printBranchSummary(out io.Writer, branch api.Branch) {
	fmt.Fprintf(out, "%s", displayBranchName(branch))
	if commit := displayCommit(branch.Commit); commit != "" {
		fmt.Fprintf(out, " %s", commit)
	}
	fmt.Fprintf(out, " protected:%t", branch.Protected.Bool())
	if branch.CreatedAt != "" {
		fmt.Fprintf(out, " created:%s", branch.CreatedAt)
	}
	if branch.Creator.Login != "" {
		fmt.Fprintf(out, " creator:%s", branch.Creator.Login)
	}
	fmt.Fprintln(out)
}

func printBranchDetail(out io.Writer, branch api.Branch) {
	fmt.Fprintf(out, "Name: %s\n", displayBranchName(branch))
	if commit := displayCommit(branch.Commit); commit != "" {
		fmt.Fprintf(out, "Commit: %s\n", commit)
	}
	fmt.Fprintf(out, "Protected: %t\n", branch.Protected.Bool())
	if branch.Default.Bool() {
		fmt.Fprintln(out, "Default: true")
	}
	if branch.Merged.Bool() {
		fmt.Fprintln(out, "Merged: true")
	}
	if branch.CanPush.Bool() {
		fmt.Fprintln(out, "Can Push: true")
	}
	if branch.DevelopersCanPush.Bool() {
		fmt.Fprintln(out, "Developers Can Push: true")
	}
	if branch.DevelopersCanMerge.Bool() {
		fmt.Fprintln(out, "Developers Can Merge: true")
	}
	if branch.CreatedAt != "" {
		fmt.Fprintf(out, "Created: %s\n", branch.CreatedAt)
	}
	if branch.Creator.Login != "" {
		fmt.Fprintf(out, "Creator: %s\n", branch.Creator.Login)
	}
}

func displayBranchName(branch api.Branch) string {
	if branch.Name != "" {
		return branch.Name
	}
	return branch.Ref
}

func displayCommit(commit api.BranchCommit) string {
	sha := commit.SHA
	if sha == "" {
		sha = commit.ID
	}
	if sha == "" {
		sha = commit.ShortID
	}
	if sha == "" {
		return ""
	}
	if len(sha) > 12 {
		sha = sha[:12]
	}

	message := strings.TrimSpace(commit.Title)
	if message == "" {
		message = strings.TrimSpace(commit.Message)
	}
	if message == "" {
		message = strings.TrimSpace(commit.Commit.Message)
	}
	if message == "" {
		return sha
	}
	message = strings.ReplaceAll(message, "\n", " ")
	return fmt.Sprintf("%s %s", sha, message)
}
