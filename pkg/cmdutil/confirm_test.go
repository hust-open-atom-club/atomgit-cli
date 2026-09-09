package cmdutil

import (
	"errors"
	"strings"
	"testing"
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
