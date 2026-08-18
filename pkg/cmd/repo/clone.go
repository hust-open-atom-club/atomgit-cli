package repo

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type CloneOptions struct {
	Directory string
	Branch    string
}

func newCmdRepoClone(f *cmdutil.Factory) *cobra.Command {
	opts := &CloneOptions{}

	cmd := &cobra.Command{
		Use:   "clone <repository> [<directory>]",
		Short: "Clone a repository",
		Long: `Clone a repository from AtomGit.

The repository argument can be:
- Full URL: https://atomgit.com/owner/repo
- Owner/repo format: owner/repo
- Just repo name (uses current user as owner)`,
		Example: `  # Clone using full URL
  ag repo clone https://atomgit.com/shinwell_hu/my-project

  # Clone using owner/repo format
  ag repo clone shinwell_hu/my-project

  # Clone to specific directory
  ag repo clone shinwell_hu/my-project my-project-local

  # Clone specific branch
  ag repo clone shinwell_hu/my-project --branch develop`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]

			// Parse repository argument
			cloneURL, repoName, err := resolveCloneRepoArg(f, repoURL)
			if err != nil {
				return err
			}

			// Determine target directory
			targetDir := repoName
			if len(args) == 2 {
				targetDir = args[1]
			}
			opts.Directory = targetDir

			return runClone(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cloneURL, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Clone specific branch")

	return cmd
}

func resolveCloneRepoArg(f *cmdutil.Factory, arg string) (string, string, error) {
	defaultOwner := ""
	if !strings.Contains(arg, "/") && !strings.HasPrefix(arg, "git@") {
		user, err := f.Config.GetUser()
		if err != nil {
			return "", "", cmdutil.AuthenticationError(err)
		}
		defaultOwner = user
	}
	return parseRepoArg(arg, defaultOwner)
}

func parseRepoArg(arg, defaultOwner string) (cloneURL, repoName string, err error) {
	// Full URL
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		cloneURL = arg
		// Extract repo name from URL
		parts := strings.Split(arg, "/")
		if len(parts) >= 2 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return cloneURL, repoName, nil
	}

	// SSH format: git@atomgit.com:owner/repo.git
	if strings.HasPrefix(arg, "git@") {
		cloneURL = arg
		parts := strings.Split(arg, "/")
		if len(parts) >= 2 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return cloneURL, repoName, nil
	}

	// Owner/repo format
	parts := strings.Split(arg, "/")
	switch len(parts) {
	case 2:
		owner := strings.TrimSpace(parts[0])
		repoName = strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git")
		if owner == "" || repoName == "" {
			return "", "", fmt.Errorf("invalid repository format: %s (expected repo or owner/repo)", arg)
		}
		cloneURL = fmt.Sprintf("https://atomgit.com/%s/%s.git", owner, repoName)
	case 1:
		repoName = strings.TrimSuffix(strings.TrimSpace(arg), ".git")
		owner := strings.TrimSpace(defaultOwner)
		if owner == "" || repoName == "" {
			return "", "", fmt.Errorf("invalid repository format: %s (expected repo or owner/repo)", arg)
		}
		cloneURL = fmt.Sprintf("https://atomgit.com/%s/%s.git", owner, repoName)
	default:
		return "", "", fmt.Errorf("invalid repository format: %s (expected repo or owner/repo)", arg)
	}

	return cloneURL, repoName, nil
}

func runClone(in io.Reader, out, errOut io.Writer, cloneURL string, opts *CloneOptions) error {
	return runCloneWithCommand(in, out, errOut, cloneURL, opts, exec.Command)
}

func runCloneWithCommand(in io.Reader, out, errOut io.Writer, cloneURL string, opts *CloneOptions, command func(string, ...string) *exec.Cmd) error {
	args := []string{"clone"}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	// Terminate option parsing before user-controlled positional arguments.
	// Without this separator, a target directory such as --config=... can be
	// interpreted by Git as an option and lead to local command execution.
	args = append(args, "--", cloneURL)

	if opts.Directory != "" {
		args = append(args, opts.Directory)
	}

	cmd := command("git", args...)
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Stdin = in

	fmt.Fprintf(out, "Cloning into '%s'...\n", opts.Directory)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Fprintf(out, "✓ Cloned repository to %s\n", opts.Directory)
	return nil
}
