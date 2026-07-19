package cmdutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

type editorRunner func(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error

// EditText opens a private temporary file in the user's configured editor and
// returns the resulting contents. The editor is executed directly, without a
// shell, so command substitutions and shell metacharacters are not evaluated.
func EditText(stdin io.Reader, stdout, stderr io.Writer, initial string) (string, error) {
	return editText(stdin, stdout, stderr, initial, os.LookupEnv, runEditor)
}

func editText(stdin io.Reader, stdout, stderr io.Writer, initial string, lookupEnv func(string) (string, bool), runner editorRunner) (string, error) {
	editor := ""
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if value, ok := lookupEnv(name); ok && strings.TrimSpace(value) != "" {
			editor = value
			break
		}
	}
	if editor == "" {
		return "", fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}

	command, err := splitEditorCommand(editor)
	if err != nil {
		return "", fmt.Errorf("invalid editor command: %w", err)
	}

	dir, err := os.MkdirTemp("", "ag-editor-*")
	if err != nil {
		return "", fmt.Errorf("create editor directory: %w", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "body.md")
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		return "", fmt.Errorf("write editor file: %w", err)
	}

	if err := runner(command[0], append(command[1:], path), stdin, stdout, stderr); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect editor file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("editor file is not a regular file")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read editor file: %w", err)
	}
	return string(content), nil
}

func runEditor(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func splitEditorCommand(value string) ([]string, error) {
	runes := []rune(strings.TrimSpace(value))
	var args []string
	var current strings.Builder
	var quote rune
	started := false

	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote == '\'' {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			started = true
			continue
		}

		if quote == '"' {
			switch r {
			case '"':
				quote = 0
			case '\\':
				if i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\') {
					i++
					current.WriteRune(runes[i])
				} else {
					current.WriteRune(r)
				}
			default:
				current.WriteRune(r)
			}
			started = true
			continue
		}

		switch {
		case r == '\'' || r == '"':
			quote = r
			started = true
		case r == '\\':
			if i+1 >= len(runes) {
				return nil, fmt.Errorf("trailing escape")
			}
			i++
			current.WriteRune(runes[i])
			started = true
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
			started = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("editor command is empty")
	}
	return args, nil
}
