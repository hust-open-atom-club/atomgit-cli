package cmdutil

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type confirmErrorReader struct{}

func (confirmErrorReader) Read([]byte) (int, error) { return 0, errors.New("simulated read failure") }

func TestConfirmResponses(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "y", input: "y\n", want: true},
		{name: "yes", input: "YES\n", want: true},
		{name: "no", input: "no\n"},
		{name: "empty", input: "\n"},
		{name: "eof", input: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var prompt strings.Builder
			got, err := Confirm(strings.NewReader(tt.input), &prompt, "Delete? [y/N] ")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("confirmed = %v, want %v", got, tt.want)
			}
			if prompt.String() != "Delete? [y/N] " {
				t.Fatalf("prompt = %q", prompt.String())
			}
		})
	}
}

func TestConfirmContextResponsesAndErrors(t *testing.T) {
	for _, input := range []string{"y\n", "YES\n", "no\n", "\n", ""} {
		t.Run(input, func(t *testing.T) {
			var prompt strings.Builder
			got, err := ConfirmContext(t.Context(), strings.NewReader(input), &prompt, "Apply? ")
			want := input == "y\n" || input == "YES\n"
			if err != nil || got != want || prompt.String() != "Apply? " {
				t.Fatalf("confirmed=%t err=%v prompt=%q", got, err, prompt.String())
			}
		})
	}
	if _, err := ConfirmContext(t.Context(), strings.NewReader("y\n"), errorWriter{}, "Apply? "); err == nil || !strings.Contains(err.Error(), "write confirmation prompt") {
		t.Fatalf("prompt error=%v", err)
	}
	if _, err := ConfirmContext(t.Context(), confirmErrorReader{}, io.Discard, ""); err == nil || !strings.Contains(err.Error(), "read confirmation response") {
		t.Fatalf("read error=%v", err)
	}
}

type startedConfirmationReader struct {
	io.Reader
	started chan struct{}
}

func (r startedConfirmationReader) Read(p []byte) (int, error) {
	close(r.started)
	return r.Reader.Read(p)
}

func TestConfirmContextCancelsPendingRead(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		confirmed, err := ConfirmContext(ctx, startedConfirmationReader{reader, started}, io.Discard, "Apply? ")
		if confirmed {
			done <- errors.New("accepted confirmation after cancellation")
			return
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("confirmation never started reading")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation waited for stdin to close")
	}
}

func TestConfirmContextAlreadyCanceledDoesNotPromptOrRead(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var prompt strings.Builder
	started := make(chan struct{})
	confirmed, err := ConfirmContext(ctx, startedConfirmationReader{strings.NewReader("yes\n"), started}, &prompt, "Apply? ")
	if confirmed || !errors.Is(err, context.Canceled) || prompt.Len() != 0 {
		t.Fatalf("confirmed=%t err=%v prompt=%q", confirmed, err, prompt.String())
	}
	select {
	case <-started:
		t.Fatal("read input after cancellation")
	default:
	}
}

func TestConfirmReturnsPromptAndReadErrors(t *testing.T) {
	var prompt strings.Builder
	if _, err := Confirm(strings.NewReader("y\n"), errorWriter{}, "Delete? "); err == nil || !strings.Contains(err.Error(), "write confirmation prompt") {
		t.Fatalf("write error = %v", err)
	}
	if _, err := Confirm(confirmErrorReader{}, &prompt, "Delete? "); err == nil || !strings.Contains(err.Error(), "read confirmation response") {
		t.Fatalf("read error = %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("simulated write failure") }
