package cmdutil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Confirm writes a confirmation prompt to promptOut and reads exactly one
// response from in. EOF, empty input, and non-affirmative responses safely
// cancel the operation. Only y and yes (case-insensitive) confirm.
func Confirm(in io.Reader, promptOut io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprint(promptOut, prompt); err != nil {
		return false, fmt.Errorf("write confirmation prompt: %w", err)
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation response: %w", err)
	}
	response := strings.ToLower(strings.TrimSpace(line))
	return response == "y" || response == "yes", nil
}
