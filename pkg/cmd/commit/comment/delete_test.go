package comment

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type countingReader struct {
	reader io.Reader
	reads  int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.reader.Read(p)
}

func TestDeleteConfirmation(t *testing.T) {
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
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.Method {
				case http.MethodGet:
					if req.URL.Path != "/api/v5/repos/alice/demo/comments/7" {
						t.Fatalf("GET path = %q", req.URL.Path)
					}
					return jsonResponse(req, http.StatusOK, `{"id":7,"body":"old","user":{"id":1001,"login":"alice"}}`), nil
				case http.MethodDelete:
					deletes++
					return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Header: make(http.Header), Body: http.NoBody, Request: req}, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
					return nil, nil
				}
			})

			cmd := newCmdDelete(newFactory(t, &testConfig{}, transport))
			if tt.yes {
				if err := cmd.Flags().Set("yes", "true"); err != nil {
					t.Fatal(err)
				}
			}
			input := &countingReader{reader: strings.NewReader(tt.input)}
			var stdout, stderr bytes.Buffer
			cmd.SetIn(input)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
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

func TestDeleteRejectsForeignAndWrongTypeComments(t *testing.T) {
	t.Run("not owner", func(t *testing.T) {
		deletes := 0
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodDelete {
				deletes++
			}
			return jsonResponse(req, http.StatusOK, `{"id":7,"body":"old","user":{"id":1002,"login":"bob"}}`), nil
		})
		cmd := newCmdDelete(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		_ = cmd.Flags().Set("yes", "true")
		cmd.SetArgs([]string{"alice/demo", "7"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "只能删除自己的评论") {
			t.Fatalf("err = %v", err)
		}
		if deletes != 0 {
			t.Fatalf("sent %d DELETE requests for a foreign comment", deletes)
		}
	})

	t.Run("issue comment id", func(t *testing.T) {
		deletes := 0
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodDelete {
				deletes++
			}
			return jsonResponse(req, http.StatusBadRequest, `{"error_code":400,"error_message":"Note type is not correct."}`), nil
		})
		cmd := newCmdDelete(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		_ = cmd.Flags().Set("yes", "true")
		cmd.SetArgs([]string{"alice/demo", "7"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not a commit comment") {
			t.Fatalf("err = %v", err)
		}
		if deletes != 0 {
			t.Fatalf("sent %d DELETE requests after type rejection", deletes)
		}
	})
}

func TestDeleteSurfacesMissingCommentAndValidatesID(t *testing.T) {
	t.Run("missing comment", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, http.StatusNotFound, `{"error_code":404,"error_message":"note not found by noteId"}`), nil
		})
		cmd := newCmdDelete(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		_ = cmd.Flags().Set("yes", "true")
		cmd.SetArgs([]string{"alice/demo", "999"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to verify commit comment") || !strings.Contains(err.Error(), "404") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		cfg := &testConfig{}
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errNoRequests
		})
		cmd := newCmdDelete(newFactory(t, cfg, transport))
		cmd.SetOut(&bytes.Buffer{})
		_ = cmd.Flags().Set("yes", "true")
		cmd.SetArgs([]string{"alice/demo", "x"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid comment ID") || cfg.tokenCalls != 0 {
			t.Fatalf("err = %v tokenCalls = %d", err, cfg.tokenCalls)
		}
	})
}
