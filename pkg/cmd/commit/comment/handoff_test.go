package comment

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// TestCreatedCommentIDFlowsToFollowUpCommands reproduces the review finding
// that the create endpoint's documented response returns the comment id as an
// opaque string (success example "12312sadsa"): the identifier printed by
// create must be accepted verbatim by view, edit, and delete.
func TestCreatedCommentIDFlowsToFollowUpCommands(t *testing.T) {
	const createResponse = `{"id":"12312sadsa","body":"LGTM","created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`
	commentPath := func(id string) string { return "/api/v5/repos/alice/demo/comments/" + id }

	var out bytes.Buffer
	createTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/commits/abc1234/comments" {
			t.Fatalf("create request = %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(req, http.StatusCreated, createResponse), nil
	})
	createCmd := newCmdCreate(newFactory(t, &testConfig{}, createTransport))
	createCmd.SetOut(&out)
	createCmd.SetArgs([]string{"alice/demo", "abc1234", "--body", "LGTM"})
	if err := createCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	match := regexp.MustCompile(`Created comment #(.+?)(?::\s*\S+)?\n$`).FindStringSubmatch(out.String())
	if match == nil {
		t.Fatalf("create output = %q, want a printed comment ID", out.String())
	}
	commentID := match[1]
	if commentID != "12312sadsa" {
		t.Fatalf("created comment ID = %q, want the create response identifier", commentID)
	}

	followUp := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != commentPath(commentID) {
			t.Fatalf("%s path = %q, want %q", req.Method, req.URL.Path, commentPath(commentID))
		}
		switch req.Method {
		case http.MethodGet:
			return jsonResponse(req, http.StatusOK, fmt.Sprintf(`{"id":%q,"body":"LGTM","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`, commentID)), nil
		case http.MethodPatch:
			return jsonResponse(req, http.StatusOK, fmt.Sprintf(`{"id":%q,"body":"updated","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T11:00:00+08:00","html_url":"https://atomgit.com/alice/demo/-/commit_comment/%s"}`, commentID, commentID)), nil
		case http.MethodDelete:
			return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Header: make(http.Header), Body: http.NoBody, Request: req}, nil
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil, nil
		}
	})

	t.Run("view", func(t *testing.T) {
		var out bytes.Buffer
		cmd := newCmdView(newFactory(t, &testConfig{}, followUp))
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"alice/demo", commentID})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out.String(), "Comment #12312sadsa\n") {
			t.Fatalf("view output = %q", out.String())
		}
	})

	t.Run("edit", func(t *testing.T) {
		var out bytes.Buffer
		cmd := newCmdEdit(newFactory(t, &testConfig{}, followUp))
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"alice/demo", commentID, "--body", "updated"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "Updated comment #12312sadsa: https://atomgit.com/alice/demo/-/commit_comment/12312sadsa\n" {
			t.Fatalf("edit output = %q", out.String())
		}
	})

	t.Run("delete", func(t *testing.T) {
		var out bytes.Buffer
		cmd := newCmdDelete(newFactory(t, &testConfig{}, followUp))
		cmd.SetOut(&out)
		_ = cmd.Flags().Set("yes", "true")
		cmd.SetArgs([]string{"alice/demo", commentID})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "Deleted comment #12312sadsa\n" {
			t.Fatalf("delete output = %q", out.String())
		}
	})
}

// TestFollowUpCommandsValidateIDBeforeNetwork pins the command-side guard:
// identifiers that cannot travel in an API path are rejected before any token
// fetch or request.
func TestFollowUpCommandsValidateIDBeforeNetwork(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{name: "path traversal", id: "../7"},
		{name: "path separator", id: "a/b"},
		{name: "query separator", id: "7?x=1"},
		{name: "fragment", id: "7#note"},
		{name: "space", id: "7 8"},
		{name: "percent escape", id: "%2e%2e"},
		{name: "dot segment", id: "."},
		{name: "dotdot segment", id: ".."},
		{name: "empty", id: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runners := map[string]func(f *cmdutil.Factory) *cobra.Command{
				"view": newCmdView,
				"edit": func(f *cmdutil.Factory) *cobra.Command { return newCmdEdit(f) },
				"delete": func(f *cmdutil.Factory) *cobra.Command {
					cmd := newCmdDelete(f)
					_ = cmd.Flags().Set("yes", "true")
					return cmd
				},
			}
			for name, build := range runners {
				t.Run(name, func(t *testing.T) {
					cfg := &testConfig{}
					transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return nil, errNoRequests
					})
					cmd := build(newFactory(t, cfg, transport))
					cmd.SetOut(&bytes.Buffer{})
					args := []string{"alice/demo", tc.id}
					if name == "edit" {
						args = append(args, "--body", "x")
					}
					cmd.SetArgs(args)
					err := cmd.Execute()
					if err == nil || !strings.Contains(err.Error(), "invalid comment ID") {
						t.Fatalf("err = %v, want invalid comment ID", err)
					}
					if cfg.tokenCalls != 0 {
						t.Fatalf("token fetched %d times before validation failed", cfg.tokenCalls)
					}
				})
			}
		})
	}
}
