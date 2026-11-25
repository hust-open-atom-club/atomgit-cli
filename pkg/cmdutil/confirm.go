package cmdutil

import (
	"bufio"
	"context"
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
	return readConfirmation(in)
}

// ConfirmContext is Confirm with cancellation while waiting for input.
// It does not close caller-owned input. After cancellation, an outstanding
// read can finish later; the caller must not reuse that input for another read.
func ConfirmContext(ctx context.Context, in io.Reader, promptOut io.Writer, prompt string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("wait for confirmation: %w", err)
	}
	if _, err := fmt.Fprint(promptOut, prompt); err != nil {
		return false, fmt.Errorf("write confirmation prompt: %w", err)
	}
	type result struct {
		confirmed bool
		err       error
	}
	ready := make(chan result, 1)
	go func() {
		confirmed, err := readConfirmation(in)
		ready <- result{confirmed, err}
	}()
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("wait for confirmation: %w", ctx.Err())
	case response := <-ready:
		// Never accept a response if cancellation raced with the input read.
		if err := ctx.Err(); err != nil {
			return false, fmt.Errorf("wait for confirmation: %w", err)
		}
		return response.confirmed, response.err
	}
}

func readConfirmation(in io.Reader) (bool, error) {
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation response: %w", err)
	}
	response := strings.ToLower(strings.TrimSpace(line))
	return response == "y" || response == "yes", nil
}
