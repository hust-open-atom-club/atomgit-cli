package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepoMirrorListTextAndCredentialRedaction(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/team/demo/push_remote_mirrors" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return forkResponse(http.StatusOK, `[{
  "id": 1,
  "project_id": 2,
  "update_status": "pending",
  "url": "https://alice:secret@example.com/team/demo.git?token=hidden#fragment",
  "number_of_failures": 0,
  "is_private": false,
  "message": "waiting for https://bot:password@example.net/queue?key=hidden",
  "force": false,
  "created_at": "2026-08-29T01:02:03Z"
}]`), nil
	})

	cmd := newCmdRepoMirrorList(repoFactory(repoCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Push remote mirror",
		"Repository: team/demo",
		"Destination: https://example.com/team/demo.git",
		"Status: pending",
		"Private: false",
		"Force: false",
		"Failures: 0",
		"Message: waiting for https://example.net/queue",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	for _, secret := range []string{"alice", "secret", "token", "hidden", "bot", "password", "key"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("output leaked %q:\n%s", secret, out.String())
		}
	}
}

func TestRepoMirrorListEmptyTextAndJSON(t *testing.T) {
	factory := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[]`), nil
	}))

	textCommand := newCmdRepoMirrorList(factory)
	var out bytes.Buffer
	textCommand.SetOut(&out)
	textCommand.SetArgs([]string{"team/demo"})
	if err := textCommand.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "No push remote mirrors configured for team/demo." {
		t.Fatalf("text output = %q", got)
	}

	jsonCommand := newCmdRepoMirrorList(factory)
	out.Reset()
	jsonCommand.SetOut(&out)
	jsonCommand.SetArgs([]string{"team/demo", "--json"})
	if err := jsonCommand.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Fatalf("JSON output = %q", got)
	}
}

func TestRepoMirrorViewJSONPreservesReturnedZeroValuesAndOmitsAbsentFields(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/team/demo/repo_remote_mirror" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{
  "id": 7,
  "update_status": "success",
  "url": "https://user:secret@example.com/team/demo.git?token=hidden",
  "number_of_failures": 0,
  "mirroring_enabled": false,
  "force": false,
  "message": "request rejected for Authorization: Bearer mirror-bearer-789",
  "last_successful_update_at": "2026-08-29T02:03:04Z"
}`), nil
	})

	cmd := newCmdRepoMirrorView(repoFactory(repoCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"type":         "repository",
		"repository":   "team/demo",
		"destination":  "https://example.com/team/demo.git",
		"status":       "success",
		"failureCount": float64(0),
		"enabled":      false,
		"force":        false,
		"message":      "request rejected for Authorization: <redacted>",
	} {
		if got := result[key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
	for _, absent := range []string{"lastError", "createdAt", "updatedAt", "private"} {
		if _, exists := result[absent]; exists {
			t.Errorf("absent field %q was emitted: %s", absent, out.String())
		}
	}
	if strings.Contains(out.String(), "secret") || strings.Contains(out.String(), "token") || strings.Contains(out.String(), "hidden") || strings.Contains(out.String(), "mirror-bearer-789") {
		t.Fatalf("JSON leaked credentials: %s", out.String())
	}
}

func TestRepoMirrorViewTextRendersFailedState(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{
  "update_status": "failed",
  "mirroring_enabled": true,
  "last_update_at": "2026-08-29T03:04:05Z",
  "last_error": "authentication failed: password=mirror-password-123; token=mirror-token-456; push to https://user:secret@example.com/team/demo.git?token=hidden"
}`), nil
	})

	cmd := newCmdRepoMirrorView(repoFactory(repoCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Status: failed",
		"Enabled: true",
		"Last update: 2026-08-29T03:04:05Z",
		"Last error: authentication failed: password=<redacted>; token=<redacted>; push to https://example.com/team/demo.git",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	for _, secret := range []string{"secret", "hidden", "mirror-password-123", "mirror-token-456"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("text output leaked %q: %s", secret, out.String())
		}
	}
}

func TestRepoMirrorInfersRepository(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/inferred/repo/repo_remote_mirror" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{}`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRepoMirrorView(factory)
	cmd.SetOut(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoMirrorListRejectsInvalidLimitBeforeAuthentication(t *testing.T) {
	cfg := &repoRecordingConfig{}
	cmd := newCmdRepoMirrorList(&cmdutil.Factory{Config: cfg})
	cmd.SetOut(io.Discard)
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"team/demo"})
	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("error = %v", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken calls = %d", cfg.getTokenCalls)
	}
}

func TestRepoMirrorResolverErrorPrecedesAuthentication(t *testing.T) {
	resolverErr := errors.New("repository context unavailable")
	cfg := &repoRecordingConfig{}
	cfg.repoCommandConfig.tokenErr = errors.New("authentication reached")
	factory := &cmdutil.Factory{Config: cfg, RepositoryResolver: func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, resolverErr
	}}
	cmd := newCmdRepoMirrorView(factory)
	cmd.SetOut(io.Discard)
	err := cmd.RunE(cmd, nil)
	if !errors.Is(err, resolverErr) {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoMirrorAPIErrorsAreContextualAndRedacted(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		statusCode int
		want       string
	}{
		{name: "list permission", command: "list", statusCode: http.StatusForbidden, want: "failed to list push remote mirrors for team/demo"},
		{name: "view missing", command: "view", statusCode: http.StatusNotFound, want: "repository remote mirror was not found for team/demo"},
		{name: "view server error", command: "view", statusCode: http.StatusInternalServerError, want: "failed to view repository remote mirror for team/demo"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := fmt.Sprintf(`{"message":"cannot reach https://user:secret@example.com/repo.git?token=hidden status %d"}`, test.statusCode)
				return forkResponse(test.statusCode, body), nil
			})
			factory := repoFactory(repoCommandConfig{token: "token"}, transport)
			var cmd interface {
				SetOut(io.Writer)
				SetArgs([]string)
				Execute() error
			}
			if test.command == "list" {
				cmd = newCmdRepoMirrorList(factory)
			} else {
				cmd = newCmdRepoMirrorView(factory)
			}
			cmd.SetOut(io.Discard)
			cmd.SetArgs([]string{"team/demo"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), fmt.Sprint(test.statusCode)) {
				t.Fatalf("error = %v", err)
			}
			for _, secret := range []string{"user", "secret", "token", "hidden"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error leaked %q: %v", secret, err)
				}
			}
		})
	}
}
