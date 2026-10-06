package comment

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreatePostsMultilineBodyToCommit(t *testing.T) {
	var method, path, body, auth string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		method, path, auth = req.Method, req.URL.Path, req.Header.Get("Authorization")
		raw, _ := io.ReadAll(req.Body)
		body = string(raw)
		return jsonResponse(req, http.StatusCreated, `{"id":"9001","body":"多行\n评论","created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`), nil
	})

	var out bytes.Buffer
	cmd := newCmdCreate(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "abc1234", "--body", "多行\n评论"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/api/v5/repos/alice/demo/commits/abc1234/comments" {
		t.Fatalf("request = %s %s", method, path)
	}
	if auth != "Bearer token" {
		t.Fatalf("authorization = %q", auth)
	}
	if body != `{"body":"多行\n评论"}` {
		t.Fatalf("body = %q", body)
	}
	if out.String() != "Created comment #9001: https://atomgit.com/alice/demo/commits/detail/abc1234\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCreatePrefersResponseWebURL(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusCreated, `{"id":9002,"web_url":"https://atomgit.com/alice/demo/-/commit_comment/9002"}`), nil
	})

	var out bytes.Buffer
	cmd := newCmdCreate(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "abc1234", "--body", "hi"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Created comment #9002: https://atomgit.com/alice/demo/-/commit_comment/9002\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCreateReadsBodyFileAndStdin(t *testing.T) {
	t.Run("body file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "comment.md")
		if err := os.WriteFile(path, []byte("from file\nsecond line"), 0600); err != nil {
			t.Fatal(err)
		}
		var body string
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(req.Body)
			body = string(raw)
			return jsonResponse(req, http.StatusCreated, `{"id":"9001"}`), nil
		})
		cmd := newCmdCreate(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"alice/demo", "abc1234", "--body-file", path})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if body != `{"body":"from file\nsecond line"}` {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("body file dash reads stdin", func(t *testing.T) {
		var body string
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(req.Body)
			body = string(raw)
			return jsonResponse(req, http.StatusCreated, `{"id":"9001"}`), nil
		})
		cmd := newCmdCreate(newFactory(t, &testConfig{}, transport))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader("piped body"))
		cmd.SetArgs([]string{"alice/demo", "abc1234", "--body-file", "-"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if body != `{"body":"piped body"}` {
			t.Fatalf("body = %q", body)
		}
	})
}

func TestCreateRejectsBadBodiesBeforeNetwork(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(emptyFile, []byte(" \n\t"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no body flags", args: []string{"alice/demo", "abc1234"}, want: "comment body is required"},
		{name: "empty body", args: []string{"alice/demo", "abc1234", "--body", "  "}, want: "comment body cannot be empty"},
		{name: "empty body file", args: []string{"alice/demo", "abc1234", "--body-file", emptyFile}, want: "comment body cannot be empty"},
		{name: "missing body file", args: []string{"alice/demo", "abc1234", "--body-file", filepath.Join(dir, "missing.md")}, want: "failed to read body file"},
		{name: "body and body file", args: []string{"alice/demo", "abc1234", "--body", "a", "--body-file", emptyFile}, want: "mutually exclusive"},
		{name: "invalid sha", args: []string{"alice/demo", "bad~sha", "--body", "a"}, want: "invalid commit SHA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &testConfig{}
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errNoRequests
			})
			cmd := newCmdCreate(newFactory(t, cfg, transport))
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

func TestCreateSurfacesAPIFailureWithContext(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusNotFound, `{"error_code":404,"error_message":"commit not found"}`), nil
	})

	cmd := newCmdCreate(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"alice/demo", "deadbeef", "--body", "hi"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "create commit comment on deadbeef") || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v", err)
	}
}
