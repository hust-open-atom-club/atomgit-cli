package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIRawInputAndAccept(t *testing.T) {
	tests := []struct {
		name  string
		input string
		body  []byte
	}{
		{name: "stdin", input: "-", body: []byte{'a', 0, 'b'}},
		{name: "file", body: []byte("file body")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if input == "" {
				input = filepath.Join(t.TempDir(), "body.bin")
				if err := os.WriteFile(input, tt.body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(body, tt.body) || req.Header.Get("Content-Type") != "" || req.Header.Get("Accept") != "application/octet-stream" {
					t.Fatalf("body = %q Content-Type = %q Accept = %q", body, req.Header.Get("Content-Type"), req.Header.Get("Accept"))
				}
				return apiTestResponse(req, http.StatusOK, "ok"), nil
			})
			fixture.stdin.Write(tt.body)
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetIn(fixture.stdin)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs([]string{"/items", "-X", "POST", "--input", input, "--accept", "application/octet-stream"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAPIInputValidationPrecedesAuthentication(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "field conflict", args: []string{"/items", "--input", "-", "--field", "a=b"}},
		{name: "missing file", args: []string{"/items", "--input", filepath.Join(t.TempDir(), "missing")}},
		{name: "empty accept", args: []string{"/items", "--accept", ""}},
		{name: "newline accept", args: []string{"/items", "--accept", "ok\r\nInjected: yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newAPITestFixture("secret", nil)
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetIn(strings.NewReader("body"))
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
			if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
				t.Fatalf("side effects = token %d client %d requests %d", fixture.config.tokenReads, fixture.clientReads, fixture.requests)
			}
		})
	}
}

func TestAPIStdinReadFailurePrecedesAuthentication(t *testing.T) {
	fixture := newAPITestFixture("secret", nil)
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetIn(failingReader{err: errors.New("stdin failed")})
	cmd.SetArgs([]string{"/items", "--input", "-"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdin failed") {
		t.Fatalf("error = %v", err)
	}
	if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
		t.Fatal("stdin read failure caused side effects")
	}
}
