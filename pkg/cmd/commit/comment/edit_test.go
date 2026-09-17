package comment

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEditPatchesOnlyBody(t *testing.T) {
	var methods []string
	var patchBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method)
		switch req.Method {
		case http.MethodGet:
			if req.URL.Path != "/api/v5/repos/alice/demo/comments/7" {
				t.Fatalf("GET path = %q", req.URL.Path)
			}
			return jsonResponse(req, http.StatusOK, `{"id":7,"body":"old","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`), nil
		case http.MethodPatch:
			if req.URL.Path != "/api/v5/repos/alice/demo/comments/7" {
				t.Fatalf("PATCH path = %q", req.URL.Path)
			}
			raw, _ := io.ReadAll(req.Body)
			patchBody = string(raw)
			return jsonResponse(req, http.StatusOK, `{"id":7,"body":"new body","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T12:00:00+08:00","html_url":"https://atomgit.com/alice/demo/-/commit_comment/7"}`), nil
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil, nil
		}
	})

	var out bytes.Buffer
	cmd := newCmdEdit(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "7", "--body", "new body"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != http.MethodGet || methods[1] != http.MethodPatch {
		t.Fatalf("methods = %v", methods)
	}
	if patchBody != `{"body":"new body"}` {
		t.Fatalf("patch body = %q", patchBody)
	}
	if out.String() != "Updated comment #7: https://atomgit.com/alice/demo/-/commit_comment/7\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestEditRejectsForeignAndWrongTypeComments(t *testing.T) {
	t.Run("not owner", func(t *testing.T) {
		patches := 0
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPatch {
				patches++
			}
			return jsonResponse(req, http.StatusOK, `{"id":7,"body":"old","user":{"id":1002,"login":"bob"}}`), nil
		})
		cmd := newCmdEdit(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "7", "--body", "new"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "只能编辑自己的评论") {
			t.Fatalf("err = %v", err)
		}
		if patches != 0 {
			t.Fatalf("sent %d PATCH requests for a foreign comment", patches)
		}
	})

	t.Run("issue comment id", func(t *testing.T) {
		patches := 0
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPatch {
				patches++
			}
			return jsonResponse(req, http.StatusBadRequest, `{"error_code":400,"error_message":"Note type is not correct."}`), nil
		})
		cmd := newCmdEdit(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "7", "--body", "new"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not a commit comment") {
			t.Fatalf("err = %v", err)
		}
		if patches != 0 {
			t.Fatalf("sent %d PATCH requests after type rejection", patches)
		}
	})

	t.Run("missing comment", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, http.StatusNotFound, `{"error_code":404,"error_message":"note not found by noteId"}`), nil
		})
		cmd := newCmdEdit(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "999", "--body", "new"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to verify commit comment") || !strings.Contains(err.Error(), "404") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestEditValidatesBodyBeforeNetwork(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no body flags", args: []string{"alice/demo", "7"}, want: "comment body is required"},
		{name: "empty body", args: []string{"alice/demo", "7", "--body", " "}, want: "comment body cannot be empty"},
		{name: "invalid id", args: []string{"alice/demo", "abc", "--body", "x"}, want: "invalid comment ID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &testConfig{}
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errNoRequests
			})
			cmd := newCmdEdit(newFactory(t, cfg, transport))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if cfg.tokenCalls != 0 {
				t.Fatalf("token fetched %d times", cfg.tokenCalls)
			}
		})
	}
}
