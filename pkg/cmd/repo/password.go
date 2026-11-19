package repo

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxRepositoryPasswordBytes = 64 << 10

// readRepositoryPassword shares bounded stdin and hidden terminal input between
// repository operations. Passwords never enter flags, prompts, or errors.
func readRepositoryPassword(cmd *cobra.Command, fromStdin bool, purpose string) (string, error) {
	if fromStdin {
		contents, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxRepositoryPasswordBytes+1))
		if err != nil {
			return "", fmt.Errorf("read %s password from stdin: %w", purpose, err)
		}
		return parseRepositoryPassword(contents, true, purpose)
	}

	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", fmt.Errorf("%s requires an interactive password; use --yes --password-stdin for non-interactive use", purpose)
	}
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "AtomGit password: "); err != nil {
		return "", fmt.Errorf("write password prompt: %w", err)
	}
	contents, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read %s password: %w", purpose, err)
	}
	return parseRepositoryPassword(contents, false, purpose)
}

func parseRepositoryPassword(contents []byte, fromStdin bool, purpose string) (string, error) {
	if len(contents) > maxRepositoryPasswordBytes {
		source := "terminal"
		if fromStdin {
			source = "stdin"
		}
		return "", fmt.Errorf("%s password from %s exceeds %d bytes", purpose, source, maxRepositoryPasswordBytes)
	}
	password := string(contents)
	if fromStdin {
		password = strings.TrimSuffix(strings.TrimSuffix(password, "\n"), "\r")
	}
	if password == "" {
		return "", fmt.Errorf("%s password must not be empty", purpose)
	}
	return password, nil
}
