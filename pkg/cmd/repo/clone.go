package repo

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
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

			// Attach the stored account credential only when the HTTPS clone
			// targets the configured AtomGit host; anonymous, SSH, foreign-host
			// and insecure clones keep their previous behavior.
			creds := resolveCloneAuth(f.Config, cloneURL)

			return runClone(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cloneURL, opts, creds)
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

// cloneCredentials authenticates an HTTPS clone of the configured AtomGit
// host with the current account's stored access token.
type cloneCredentials struct {
	host     string
	username string
	token    string
}

// resolveCloneAuth returns credentials for an HTTPS clone that targets the
// configured AtomGit host when the current account is authenticated. It
// returns nil for SSH, foreign-host, insecure (plain HTTP) and anonymous
// clones so those keep working exactly as before. The token is deliberately
// not embedded in the clone URL: the remote URL git persists stays clean.
func resolveCloneAuth(cfg config.Config, cloneURL string) *cloneCredentials {
	u, err := url.Parse(cloneURL)
	if err != nil || u.Scheme != "https" || u.Host != cfg.GetHost() {
		return nil
	}
	token, err := cfg.GetToken()
	if err != nil || token == "" {
		return nil
	}
	username, err := cfg.GetUser()
	if err != nil || username == "" {
		return nil
	}
	return &cloneCredentials{host: u.Host, username: username, token: token}
}

// gitCloneAuthEnv returns the environment entries that attach a one-shot
// Basic Authorization header to the git clone process. AtomGit's Git server
// accepts the account login with the access token as the password; passing
// the header through environment-driven Git configuration keeps the token out
// of the process argument list. Nothing is written to the cloned repository's
// configuration, so the token is not persisted. GIT_TERMINAL_PROMPT is
// disabled so an invalid token fails fast instead of hanging on a prompt.
func gitCloneAuthEnv(creds *cloneCredentials) []string {
	basic := base64.StdEncoding.EncodeToString([]byte(creds.username + ":" + creds.token))
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.https://" + creds.host + "/.extraheader",
		"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + basic,
	}
}

// withCloneAuthEnv merges the clone auth environment into cmd.Env (or the
// process environment when cmd.Env is unset), replacing any pre-existing
// GIT_CONFIG/GIT_TERMINAL_PROMPT values so the injected header wins.
func withCloneAuthEnv(env []string, creds *cloneCredentials) []string {
	base := env
	if len(base) == 0 {
		base = os.Environ()
	}
	filtered := make([]string, 0, len(base)+4)
	for _, kv := range base {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		if name == "GIT_TERMINAL_PROMPT" || name == "GIT_CONFIG_COUNT" ||
			strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_") {
			continue
		}
		filtered = append(filtered, kv)
	}
	return append(filtered, gitCloneAuthEnv(creds)...)
}

func runClone(in io.Reader, out, errOut io.Writer, cloneURL string, opts *CloneOptions, creds *cloneCredentials) error {
	return runCloneWithCommand(in, out, errOut, cloneURL, opts, creds, exec.Command)
}

func runCloneWithCommand(in io.Reader, out, errOut io.Writer, cloneURL string, opts *CloneOptions, creds *cloneCredentials, command func(string, ...string) *exec.Cmd) error {
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

	if creds != nil {
		cmd.Env = withCloneAuthEnv(cmd.Env, creds)
	}

	fmt.Fprintf(out, "Cloning into '%s'...\n", opts.Directory)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Fprintf(out, "✓ Cloned repository to %s\n", opts.Directory)
	return nil
}
