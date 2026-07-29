package cmdutil

import (
	"fmt"
	"io"
	"os"
)

// ReadBody resolves body text from a flag value, file, or standard input.
func ReadBody(body, bodyFile string, bodyChanged, bodyFileChanged bool, stdin io.Reader) (string, error) {
	if bodyChanged && bodyFileChanged {
		return "", fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	if !bodyFileChanged {
		return body, nil
	}

	var (
		content []byte
		err     error
	)
	if bodyFile == "-" {
		content, err = io.ReadAll(stdin)
	} else {
		content, err = os.ReadFile(bodyFile)
	}
	if err != nil {
		return "", fmt.Errorf("failed to read body file: %w", err)
	}
	return string(content), nil
}
