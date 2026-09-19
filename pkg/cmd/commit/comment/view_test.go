package comment

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestViewPrintsCommentDetail(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/comments/7" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(req, http.StatusOK, `{"id":7,"body":"looks good\nship it","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T11:30:00+08:00","target":{"sha":"abc1234"}}`), nil
	})

	var out bytes.Buffer
	cmd := newCmdView(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "7"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Comment #7\nAuthor: @alice\nCreated: 2026-09-15 10:00\nUpdated: 2026-09-15 11:30\n\nlooks good\nship it\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestViewJSONOutput(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, `{"id":7,"body":"hello","user":{"id":1002,"login":"bob"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`), nil
	})

	var out bytes.Buffer
	cmd := newCmdView(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "7", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := `{
  "id": "7",
  "body": "hello",
  "author": "bob",
  "created_at": "2026-09-15T10:00:00+08:00",
  "updated_at": "2026-09-15T10:00:00+08:00"
}
`
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestViewRejectsNonCommitAndMissingComments(t *testing.T) {
	t.Run("issue comment id rejected", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, http.StatusBadRequest, `{"error_code":400,"error_message":"Note type is not correct."}`), nil
		})
		cmd := newCmdView(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "7"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not a commit comment") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("missing comment surfaces not found", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, http.StatusNotFound, `{"error_code":404,"error_message":"note not found by noteId"}`), nil
		})
		cmd := newCmdView(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "999"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "get commit comment #999") || !strings.Contains(err.Error(), "404") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		cfg := &testConfig{}
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errNoRequests
		})
		cmd := newCmdView(newFactory(t, cfg, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "../7"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid comment ID") || cfg.tokenCalls != 0 {
			t.Fatalf("err = %v tokenCalls = %d", err, cfg.tokenCalls)
		}
	})
}

func TestViewAcceptsCreateResponseStringID(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/comments/12312sadsa" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(req, http.StatusOK, `{"id":"12312sadsa","body":"LGTM","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`), nil
	})

	var out bytes.Buffer
	cmd := newCmdView(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "12312sadsa"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if want := "Comment #12312sadsa\nAuthor: @alice\nCreated: 2026-09-15 10:00\n\nLGTM\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}
