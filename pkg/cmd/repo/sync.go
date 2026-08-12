package repo

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type SyncOptions struct {
	Branch string
	Force  bool
	Yes    bool
}

type syncRepository struct {
	Owner string
	Name  string
}

func (r syncRepository) String() string {
	return r.Owner + "/" + r.Name
}

func newCmdRepoSync(f *cmdutil.Factory) *cobra.Command {
	opts := &SyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync [<owner>/<repo>]",
		Short: "Synchronize a fork with its upstream repository",
		Long: `Synchronize a remote AtomGit fork branch with its upstream repository.

The repository must be a fork with an available upstream repository. The
repository default branch is used unless --branch is specified. This command
updates only the remote fork and does not modify the local Git working tree.

By default, AtomGit performs a non-forced synchronization and reports a
conflict instead of overwriting divergent commits. --force requires an
interactive confirmation unless --yes is also supplied.`,
		Example: `  # Synchronize the current repository's default branch
  ag repo sync

  # Synchronize an explicit branch of a fork
  ag repo sync owner/fork --branch develop

  # Force synchronization after interactive confirmation
  ag repo sync owner/fork --branch develop --force

  # Force synchronization non-interactively
  ag repo sync owner/fork --branch develop --force --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			return runRepoSync(cmd, f, opts, syncRepository{Owner: repository.Owner, Name: repository.Name})
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Branch to synchronize (defaults to the repository default branch)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Overwrite divergent commits after confirmation")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation when --force is used")
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func runRepoSync(cmd *cobra.Command, f *cmdutil.Factory, opts *SyncOptions, repository syncRepository) error {
	token, err := f.Config.GetToken()
	if err != nil {
		return err
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return fmt.Errorf("failed to synchronize %s: %w", repository, err)
	}

	var metadata api.Repository
	if err := client.Get(repoAPIPath(repository), &metadata); err != nil {
		return fmt.Errorf("failed to read repository %s: %w", repository, err)
	}
	if !metadata.Fork {
		return fmt.Errorf("repository %s is not a fork", repository)
	}

	upstream, err := parseSyncUpstream(metadata.ParentFullName)
	if err != nil {
		return fmt.Errorf("repository %s has no usable upstream: %w", repository, err)
	}
	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		branch = strings.TrimSpace(metadata.DefaultBranch)
	}
	if branch == "" {
		return fmt.Errorf("repository %s has no default branch; specify --branch", repository)
	}

	forkSHA, err := readSyncBranch(client, repository, branch)
	if err != nil {
		return fmt.Errorf("failed to read branch %q in fork %s: %w", branch, repository, err)
	}
	upstreamSHA, err := readSyncBranch(client, upstream, branch)
	if err != nil {
		return fmt.Errorf("failed to read branch %q in upstream %s: %w", branch, upstream, err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Repository: %s\n", repository)
	fmt.Fprintf(out, "Upstream: %s\n", upstream)
	fmt.Fprintf(out, "Branch: %s\n", branch)
	if forkSHA == upstreamSHA {
		fmt.Fprintln(out, "Already up to date.")
		return nil
	}

	if opts.Force && !opts.Yes {
		confirmed, err := confirmRepoSync(cmd.InOrStdin(), out, repository, upstream, branch)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(out, "Repository synchronization cancelled.")
			return nil
		}
	}

	request := api.RepositorySyncRequest{Branch: branch, Force: opts.Force}
	var response api.RepositorySyncResponse
	if err := client.Put(repoAPIPath(repository)+"/sync_repo", request, &response); err != nil {
		return fmt.Errorf("failed to synchronize branch %q in %s from %s: %w", branch, repository, upstream, err)
	}
	if !response.Result {
		return fmt.Errorf("AtomGit did not synchronize branch %q in %s", branch, repository)
	}

	fmt.Fprintf(out, "Synchronized %s:%s from %s", repository, branch, upstream)
	if opts.Force {
		fmt.Fprint(out, " (forced)")
	}
	fmt.Fprintln(out)
	return nil
}

func parseSyncUpstream(value string) (syncRepository, error) {
	parsed, err := cmdutil.ParseRepository(value)
	if err != nil {
		return syncRepository{}, err
	}
	return syncRepository{Owner: parsed.Owner, Name: parsed.Name}, nil
}

func repoAPIPath(repository syncRepository) string {
	return "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name)
}

func readSyncBranch(client *api.Client, repository syncRepository, branch string) (string, error) {
	var result api.Branch
	path := repoAPIPath(repository) + "/branches/" + url.PathEscape(branch)
	if err := client.Get(path, &result); err != nil {
		return "", err
	}
	sha := strings.TrimSpace(result.Commit.SHA)
	if sha == "" {
		sha = strings.TrimSpace(result.Commit.ID)
	}
	if sha == "" {
		return "", fmt.Errorf("branch response has no commit SHA")
	}
	return sha, nil
}

func confirmRepoSync(in io.Reader, out io.Writer, repository, upstream syncRepository, branch string) (bool, error) {
	fmt.Fprintf(out, "Force-sync %s:%s from %s and overwrite divergent commits? [y/N] ", repository, branch, upstream)
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
