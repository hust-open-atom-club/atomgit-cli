package auth

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const atomGitCredentialHost = "atomgit.com"

func newCmdAuthSetupGit(f *cmdutil.Factory) *cobra.Command {
	return newCmdAuthSetupGitWithExecutable(f, resolveExecutablePath)
}

func newCmdAuthSetupGitWithExecutable(f *cmdutil.Factory, executable func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup-git",
		Short: "Configure Git to use ag as a credential helper",
		Long: `Configure Git to use AtomGit CLI as the HTTPS credential helper for
atomgit.com. Git requests credentials from the active account selected by
ag auth switch; access tokens are not written to Git configuration.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if f == nil || f.Config == nil {
				return fmt.Errorf("configuration is unavailable")
			}
			if _, err := f.Config.GetToken(); err != nil {
				return err
			}
			path, err := executable()
			if err != nil {
				return fmt.Errorf("resolve ag executable: %w", err)
			}
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("resolve ag executable: path is empty")
			}

			run := gitConfigRunner(f)
			key := "credential.https://" + atomGitCredentialHost + ".helper"
			// An empty host-specific helper severs Git's inherited helper chain,
			// ensuring credentials for AtomGit come from ag rather than a generic
			// credential manager configured for every host.
			if _, err := run("config", "--global", "--replace-all", key, ""); err != nil {
				return fmt.Errorf("reset Git credential helper for %s: %w", atomGitCredentialHost, err)
			}
			helper := "!" + shellQuote(path) + " auth git-credential"
			if _, err := run("config", "--global", "--add", key, helper); err != nil {
				return fmt.Errorf("configure Git credential helper for %s: %w", atomGitCredentialHost, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Configured Git to use ag as a credential helper for %s.\n", atomGitCredentialHost)
			return nil
		},
	}
	return cmd
}

func newCmdAuthGitCredential(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-credential <operation>",
		Short:  "Implement the Git credential helper protocol",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitCredentialHelper(f, args[0], cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	return cmd
}

func runGitCredentialHelper(f *cmdutil.Factory, operation string, in io.Reader, out io.Writer) error {
	switch operation {
	case "store", "erase":
		// ag's credential store remains the source of truth. Git must not log
		// the user out or persist another copy of the token.
		return nil
	case "get":
	default:
		return fmt.Errorf("ag auth git-credential: %q operation not supported", operation)
	}

	wants, err := readGitCredentialRequest(in)
	if err != nil {
		return fmt.Errorf("read Git credential request: %w", err)
	}
	if wants["protocol"] != "https" || !isAtomGitCredentialHost(wants["host"]) {
		return nil
	}
	if f == nil || f.Config == nil {
		return nil
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return nil
	}
	user, err := f.Config.GetUser()
	if err != nil {
		return nil
	}
	if token == "" || user == "" || containsCredentialLineBreak(token) || containsCredentialLineBreak(user) {
		return nil
	}
	if requestedUser := wants["username"]; requestedUser != "" && !strings.EqualFold(requestedUser, user) {
		return nil
	}

	fmt.Fprintln(out, "protocol=https")
	fmt.Fprintf(out, "host=%s\n", wants["host"])
	fmt.Fprintf(out, "username=%s\n", user)
	fmt.Fprintf(out, "password=%s\n", token)
	return nil
}

func readGitCredentialRequest(in io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == "url" {
			parsed, err := url.Parse(parts[1])
			if err != nil {
				// url.Parse errors can include the complete input URL, including
				// credentials supplied by Git. Keep the error credential-free.
				return nil, fmt.Errorf("invalid URL field")
			}
			values["protocol"] = parsed.Scheme
			values["host"] = parsed.Host
			values["path"] = parsed.Path
			if parsed.User != nil {
				values["username"] = parsed.User.Username()
			}
			continue
		}
		values[parts[0]] = parts[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func containsCredentialLineBreak(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}

func isAtomGitCredentialHost(host string) bool {
	if strings.EqualFold(host, atomGitCredentialHost) {
		return true
	}
	hostname, port, err := net.SplitHostPort(host)
	return err == nil && port == "443" && strings.EqualFold(hostname, atomGitCredentialHost)
}

func shellQuote(value string) string {
	// Git executes helpers prefixed with ! through a shell. Always quote the
	// executable path so every shell metacharacter is treated as path data.
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func resolveExecutablePath() (string, error) {
	if path, err := exec.LookPath(os.Args[0]); err == nil {
		return filepath.Abs(path)
	}
	return os.Executable()
}

func gitConfigRunner(f *cmdutil.Factory) func(args ...string) (string, error) {
	if f != nil && f.GitConfig != nil {
		return f.GitConfig
	}
	return func(args ...string) (string, error) {
		output, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
		return strings.TrimSpace(string(output)), nil
	}
}
