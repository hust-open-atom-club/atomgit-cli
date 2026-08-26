package comment

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type issueDeleteConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *issueDeleteConfig) GetToken() (string, error) {
	c.tokenCalls++
	return "token", c.tokenErr
}
func (*issueDeleteConfig) GetUser() (string, error) { return "alice", nil }
func (*issueDeleteConfig) GetHost() string          { return "atomgit.com" }

type issueDeleteRoundTripFunc func(*http.Request) (*http.Response, error)

func (f issueDeleteRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type issueDeleteReader struct {
	reader io.Reader
	reads  int
}

func (r *issueDeleteReader) Read(p []byte) (int, error) {
	r.reads++
	return r.reader.Read(p)
}

func TestIssueCommentDeleteConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		yes         bool
		wantDeletes int
		wantOutput  string
		wantPrompt  bool
	}{
		{name: "no", input: "n\n", wantOutput: "取消删除\n", wantPrompt: true},
		{name: "empty", input: "\n", wantOutput: "取消删除\n", wantPrompt: true},
		{name: "EOF", input: "", wantOutput: "取消删除\n", wantPrompt: true},
		{name: "short yes", input: "y\n", wantDeletes: 1, wantOutput: "Deleted comment #7\n", wantPrompt: true},
		{name: "yes", input: "yes\n", wantDeletes: 1, wantOutput: "Deleted comment #7\n", wantPrompt: true},
		{name: "yes flag", input: "n\n", yes: true, wantDeletes: 1, wantOutput: "Deleted comment #7\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletes := 0
			transport := issueDeleteRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.Method {
				case http.MethodGet:
					return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"user":{"login":"alice"}}`)), Request: req}, nil
				case http.MethodDelete:
					deletes++
					return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Header: make(http.Header), Body: http.NoBody, Request: req}, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
					return nil, nil
				}
			})
			cfg := &issueDeleteConfig{}
			factory := &cmdutil.Factory{Config: cfg, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: transport}, nil
			}}
			cmd := newCmdDelete(factory)
			if tt.yes {
				if err := cmd.Flags().Set("yes", "true"); err != nil {
					t.Fatal(err)
				}
			}
			input := &issueDeleteReader{reader: strings.NewReader(tt.input)}
			var stdout, stderr bytes.Buffer
			cmd.SetIn(input)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			if err := cmd.RunE(cmd, []string{"alice/demo", "3", "7"}); err != nil {
				t.Fatal(err)
			}
			if deletes != tt.wantDeletes || stdout.String() != tt.wantOutput {
				t.Fatalf("deletes=%d output=%q, want deletes=%d output=%q", deletes, stdout.String(), tt.wantDeletes, tt.wantOutput)
			}
			if strings.Contains(stdout.String(), "确定要删除") {
				t.Fatalf("prompt leaked to stdout: %q", stdout.String())
			}
			if got := strings.Contains(stderr.String(), "确定要删除评论 #7"); got != tt.wantPrompt {
				t.Fatalf("prompt present=%v stderr=%q, want %v", got, stderr.String(), tt.wantPrompt)
			}
			if tt.yes && input.reads != 0 {
				t.Fatalf("--yes read stdin %d times", input.reads)
			}
		})
	}
}

func TestIssueCommentDeleteYesStillValidatesAndAuthenticates(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		cfg := &issueDeleteConfig{}
		cmd := newCmdDelete(&cmdutil.Factory{Config: cfg})
		_ = cmd.Flags().Set("yes", "true")
		input := &issueDeleteReader{reader: strings.NewReader("yes\n")}
		cmd.SetIn(input)
		err := cmd.RunE(cmd, []string{"alice/demo", "invalid", "7"})
		if err == nil || !strings.Contains(err.Error(), "invalid issue number") || cfg.tokenCalls != 0 || input.reads != 0 {
			t.Fatalf("error=%v tokenCalls=%d reads=%d", err, cfg.tokenCalls, input.reads)
		}
	})

	t.Run("authentication", func(t *testing.T) {
		cfg := &issueDeleteConfig{tokenErr: errors.New("token failed")}
		cmd := newCmdDelete(&cmdutil.Factory{Config: cfg})
		_ = cmd.Flags().Set("yes", "true")
		input := &issueDeleteReader{reader: strings.NewReader("yes\n")}
		cmd.SetIn(input)
		err := cmd.RunE(cmd, []string{"alice/demo", "3", "7"})
		if err == nil || !strings.Contains(err.Error(), "token failed") || cfg.tokenCalls != 1 || input.reads != 0 {
			t.Fatalf("error=%v tokenCalls=%d reads=%d", err, cfg.tokenCalls, input.reads)
		}
	})
}
