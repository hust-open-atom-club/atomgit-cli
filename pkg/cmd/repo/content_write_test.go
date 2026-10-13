package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const contentBlobSHA = "0123456789abcdef0123456789abcdef01234567"

func TestContentWritesRejectInvalidInputBeforeAuth(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	factory := &cmdutil.Factory{Config: repoCommandConfig{tokenErr: config.ErrNotAuthenticated}}
	cases := []struct {
		name string
		new  func(*cmdutil.Factory) interface{ Execute() error }
		args []string
		want string
	}{
		{"missing branch", func(f *cmdutil.Factory) interface{ Execute() error } { return newCmdRepoContentCreate(f) }, []string{"alice/demo", "README.md", "--message", "add", "--file", file}, "required flag"},
		{"blank message", func(f *cmdutil.Factory) interface{ Execute() error } { return newCmdRepoContentCreate(f) }, []string{"alice/demo", "README.md", "--branch", " ", "--message", " ", "--file", file}, "--branch cannot be empty"},
		{"bad path", func(f *cmdutil.Factory) interface{ Execute() error } { return newCmdRepoContentUpdate(f) }, []string{"alice/demo", "../README.md", "--branch", "main", "--message", "edit", "--sha", contentBlobSHA, "--file", file}, "'.' or '..'"},
		{"short sha", func(f *cmdutil.Factory) interface{ Execute() error } { return newCmdRepoContentDelete(f) }, []string{"alice/demo", "README.md", "--branch", "main", "--message", "remove", "--sha", "abc", "--yes"}, "40-character"},
		{"missing file", func(f *cmdutil.Factory) interface{ Execute() error } { return newCmdRepoContentCreate(f) }, []string{"alice/demo", "README.md", "--branch", "main", "--message", "add", "--file", filepath.Join(dir, "missing")}, "read content file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.new(factory)
			root, ok := cmd.(interface {
				SetArgs([]string)
				SetOut(io.Writer)
				SetErr(io.Writer)
			})
			if !ok {
				t.Fatal("command does not expose test setters")
			}
			root.SetArgs(tc.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), config.ErrNotAuthenticated.Error()) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestContentCreateSendsRawBytes(t *testing.T) {
	raw := []byte{0, 255, '\n'}
	file := filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/api/v5/repos/alice/demo/contents/docs/%E6%96%87%E4%BB%B6.txt" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return forkResponse(http.StatusOK, `{"commit":{"sha":"commit-sha","message":"add"}}`), nil
	})
	cmd := newCmdRepoContentCreate(repoFactory(repoCommandConfig{token: "token"}, transport))
	cmd.SetArgs([]string{"alice/demo", "docs/文件.txt", "--branch", "feature/x", "--message", "add file", "--file", file})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got["content"] != "AP8K" || got["message"] != "add file" || got["branch"] != "feature/x" || got["sha"] != "" {
	}
	if out.String() != "Created docs/文件.txt on feature/x\nCommit: commit-sha\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestContentUpdateReadsStdinAndReportsConflict(t *testing.T) {
	var calls int
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var got map[string]string
		if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got["content"] != "" || got["sha"] != contentBlobSHA || got["branch"] != "main" {
			t.Fatalf("request = %#v", got)
		}
		return forkResponse(http.StatusConflict, `{"message":"sha mismatch"}`), nil
	})
	cmd := newCmdRepoContentUpdate(repoFactory(repoCommandConfig{token: "token"}, transport))
	cmd.SetArgs([]string{"alice/demo", "README.md", "--branch", "main", "--message", "edit", "--sha", contentBlobSHA, "--file", "-"})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "sha mismatch") || calls != 1 {
		t.Fatalf("calls = %d err = %v", calls, err)
	}
}

func TestContentDeleteConfirmationAndSuccess(t *testing.T) {
	t.Run("cancel before auth", func(t *testing.T) {
		cmd := newCmdRepoContentDelete(&cmdutil.Factory{Config: repoCommandConfig{tokenErr: errors.New("token failed")}})
		cmd.SetArgs([]string{"alice/demo", "README.md", "--branch", "main", "--message", "remove", "--sha", contentBlobSHA})
		cmd.SetIn(strings.NewReader("n\n"))
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "Deletion cancelled.\n" {
			t.Fatalf("output = %q", out.String())
		}
	})

	t.Run("yes deletes without content object", func(t *testing.T) {
		var method string
		transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			method = req.Method
			if req.URL.Path != "/api/v5/repos/alice/demo/contents/README.md" {
				t.Fatalf("path = %s", req.URL.Path)
			}
			return forkResponse(http.StatusOK, `{"commit":{"message":"remove"}}`), nil
		})
		cmd := newCmdRepoContentDelete(repoFactory(repoCommandConfig{token: "token"}, transport))
		cmd.SetArgs([]string{"alice/demo", "README.md", "--branch", "main", "--message", "remove", "--sha", contentBlobSHA, "--yes"})
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if method != http.MethodDelete || out.String() != "Deleted README.md on main\n" {
			t.Fatalf("method = %s output = %q", method, out.String())
		}
	})
}

func TestContentUpdatePrintsOptionalBlob(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Fatalf("method = %s", req.Method)
		}
		return forkResponse(http.StatusOK, `{"commit":{"sha":"commit-sha"},"content":{"sha":"new-blob","type":"file"}}`), nil
	})
	cmd := newCmdRepoContentUpdate(repoFactory(repoCommandConfig{token: "token"}, transport))
	cmd.SetArgs([]string{"alice/demo", "README.md", "--branch", "main", "--message", "edit", "--sha", contentBlobSHA, "--file", "-"})
	cmd.SetIn(bytes.NewReader([]byte("plain\n")))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Updated README.md on main\nCommit: commit-sha\nBlob: new-blob\n" {
		t.Fatalf("output = %q", out.String())
	}
}
