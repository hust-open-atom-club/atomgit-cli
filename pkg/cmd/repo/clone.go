package repo

import (
	"fmt"
	"os"
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
			cloneURL, repoName := parseRepoArg(repoURL)

			// Determine target directory
			targetDir := repoName
			if len(args) == 2 {
				targetDir = args[1]
			}
			opts.Directory = targetDir

			return runClone(cloneURL, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Clone specific branch")

	return cmd
}

func parseRepoArg(arg string) (cloneURL, repoName string) {
	// Full URL
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		cloneURL = arg
		// Extract repo name from URL
		parts := strings.Split(arg, "/")
		if len(parts) >= 2 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return
	}

	// SSH format: git@atomgit.com:owner/repo.git
	if strings.HasPrefix(arg, "git@") {
		cloneURL = arg
		parts := strings.Split(arg, "/")
		if len(parts) >= 2 {
			repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return
	}

	// Owner/repo format
	parts := strings.Split(arg, "/")
	if len(parts) == 2 {
		cloneURL = fmt.Sprintf("https://atomgit.com/%s/%s.git", parts[0], parts[1])
		repoName = parts[1]
	} else {
		// Just repo name - will need current user
		cloneURL = fmt.Sprintf("https://atomgit.com/%s", arg)
		repoName = arg
	}

	return
}

func runClone(cloneURL string, opts *CloneOptions) error {
	return runCloneWithCommand(cloneURL, opts, exec.Command)
}

func runCloneWithCommand(cloneURL string, opts *CloneOptions, command func(string, ...string) *exec.Cmd) error {
	args := []string{"clone"}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	args = append(args, cloneURL)

	if opts.Directory != "" {
		args = append(args, opts.Directory)
	}

	cmd := command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Cloning into '%s'...\n", opts.Directory)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Printf("✓ Cloned repository to %s\n", opts.Directory)
	return nil
}
