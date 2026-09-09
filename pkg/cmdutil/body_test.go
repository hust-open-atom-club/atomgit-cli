package cmdutil

import (
	"errors"
	"strings"
	"testing"
)

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestReadBody(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		bodyFile        string
		bodyChanged     bool
		bodyFileChanged bool
		stdin           interface{ Read([]byte) (int, error) }
		want            string
		wantError       string
	}{
		{name: "body flag", body: "inline\n", bodyChanged: true, stdin: strings.NewReader("unused"), want: "inline\n"},
		{name: "stdin", bodyFile: "-", bodyFileChanged: true, stdin: strings.NewReader("UTF-8 正文\n\n"), want: "UTF-8 正文\n\n"},
		{name: "empty stdin", bodyFile: "-", bodyFileChanged: true, stdin: strings.NewReader(""), want: ""},
		{name: "conflicting flags", body: "inline", bodyFile: "-", bodyChanged: true, bodyFileChanged: true, stdin: strings.NewReader("unused"), wantError: "mutually exclusive"},
		{name: "stdin error", bodyFile: "-", bodyFileChanged: true, stdin: errorReader{err: errors.New("read failed")}, wantError: "failed to read body file: read failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadBody(tt.body, tt.bodyFile, tt.bodyChanged, tt.bodyFileChanged, tt.stdin)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}
