package repo

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
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
			// targets a first-party AtomGit host (atomgit.com or its
			// gitcode.com mirror); anonymous, SSH, foreign-host and insecure
			// clones keep their previous behavior.
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

// resolveCloneAuth returns credentials for an HTTPS clone that targets a
// first-party AtomGit host (atomgit.com or its gitcode.com mirror) when the
// current account is authenticated. It returns nil for SSH, foreign-host,
// insecure (plain HTTP) and anonymous clones so those keep working exactly as
// before. The token is deliberately not embedded in the clone URL: the remote
// URL git persists stays clean.
func resolveCloneAuth(cfg config.Config, cloneURL string) *cloneCredentials {
	u, err := url.Parse(cloneURL)
	if err != nil || u.Scheme != "https" || !cmdutil.IsAtomGitHost(u.Host) {
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

// gitCloneAuthHeader returns the Basic Authorization header value used to
// authenticate the clone. AtomGit's Git server accepts the account login with
// the access token as the password; passing the header through
// environment-driven Git configuration keeps the token out of the process
// argument list. Nothing is written to the cloned repository's configuration,
// so the token is not persisted.
func gitCloneAuthHeader(creds *cloneCredentials) string {
	basic := base64.StdEncoding.EncodeToString([]byte(creds.username + ":" + creds.token))
	return "AUTHORIZATION: basic " + basic
}

// withCloneAuthEnv merges the clone auth configuration into cmd.Env (or the
// process environment when cmd.Env is unset). Unrelated environment-driven Git
// configuration such as http.proxy or http.sslCAInfo is preserved: the auth
// extraheader is appended at the next GIT_CONFIG_* index, or replaces an
// existing extraheader for the same URL when one is already present.
// GIT_TERMINAL_PROMPT is disabled so an invalid token fails fast instead of
// hanging on a prompt.
func withCloneAuthEnv(env []string, creds *cloneCredentials) []string {
	base := env
	if len(base) == 0 {
		base = os.Environ()
	}

	authKey := "http.https://" + creds.host + "/.extraheader"
	authValue := gitCloneAuthHeader(creds)

	keys := make(map[int]string)
	values := make(map[int]string)
	active := 0
	countSet := false
	kept := make([]string, 0, len(base)+4)

	for _, kv := range base {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			kept = append(kept, kv)
			continue
		}
		switch {
		case name == "GIT_CONFIG_COUNT":
			if n, err := strconv.Atoi(value); err == nil && n >= 0 {
				active = n
				countSet = true
			}
		case strings.HasPrefix(name, "GIT_CONFIG_KEY_"):
			idx, err := strconv.Atoi(strings.TrimPrefix(name, "GIT_CONFIG_KEY_"))
			if err != nil || idx < 0 {
				kept = append(kept, kv)
				continue
			}
			keys[idx] = value
		case strings.HasPrefix(name, "GIT_CONFIG_VALUE_"):
			idx, err := strconv.Atoi(strings.TrimPrefix(name, "GIT_CONFIG_VALUE_"))
			if err != nil || idx < 0 {
				kept = append(kept, kv)
				continue
			}
			values[idx] = value
		default:
			// GIT_TERMINAL_PROMPT is deliberately dropped here: the auth
			// configuration forces prompt-free operation for the clone.
			if name != "GIT_TERMINAL_PROMPT" {
				kept = append(kept, kv)
			}
		}
	}

	// Without GIT_CONFIG_COUNT, inherited GIT_CONFIG_KEY_*/VALUE_* variables
	// are inert because git reads them only when the count is positive, so
	// there is nothing to preserve and the auth header becomes index 0.
	authIndex := -1
	if countSet {
		for i := 0; i < active; i++ {
			if keys[i] == authKey {
				authIndex = i
				break
			}
		}
	}

	for i := 0; i < active; i++ {
		if i == authIndex {
			kept = append(kept,
				"GIT_CONFIG_KEY_"+strconv.Itoa(i)+"="+authKey,
				"GIT_CONFIG_VALUE_"+strconv.Itoa(i)+"="+authValue)
			continue
		}
		if k, ok := keys[i]; ok {
			kept = append(kept, "GIT_CONFIG_KEY_"+strconv.Itoa(i)+"="+k)
		}
		if v, ok := values[i]; ok {
			kept = append(kept, "GIT_CONFIG_VALUE_"+strconv.Itoa(i)+"="+v)
		}
	}

	finalCount := active
	if authIndex == -1 {
		kept = append(kept,
			"GIT_CONFIG_KEY_"+strconv.Itoa(active)+"="+authKey,
			"GIT_CONFIG_VALUE_"+strconv.Itoa(active)+"="+authValue)
		finalCount = active + 1
	}

	return append(kept, "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_COUNT="+strconv.Itoa(finalCount))
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
