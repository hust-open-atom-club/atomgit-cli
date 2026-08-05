package issue

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type issueEditRecordingConfig struct {
	getTokenCalls int
}

func (c *issueEditRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}

func (*issueEditRecordingConfig) GetUser() (string, error) { return "alice", nil }
func (*issueEditRecordingConfig) GetHost() string          { return "atomgit.com" }

func TestIssueEditUpdatesOnlyRequestedFields(t *testing.T) {
	tests := []struct {
		name         string
		flags        map[string]string
		wantFields   map[string]string
		wantRequests int
	}{
		{
			name:         "title only",
			flags:        map[string]string{"title": "Updated title"},
			wantRequests: 1,
			wantFields: map[string]string{
				"repo": "demo", "title": "Updated title",
			},
		},
		{
			name:         "body only",
			flags:        map[string]string{"body": "Updated body"},
			wantRequests: 2,
			wantFields: map[string]string{
				"repo": "demo", "title": "Existing title", "body": "Updated body",
			},
		},
		{
			name:         "clear body",
			flags:        map[string]string{"body": ""},
			wantRequests: 2,
			wantFields: map[string]string{
				"repo": "demo", "title": "Existing title", "body": "",
			},
		},
		{
			name:         "title and body",
			flags:        map[string]string{"title": "Updated title", "body": "Updated body"},
			wantRequests: 1,
			wantFields: map[string]string{
				"repo": "demo", "title": "Updated title", "body": "Updated body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, requests := issueEditTestFactory(t, tt.wantFields, http.StatusOK, tt.wantRequests == 2)
			cmd := newCmdIssueEdit(factory)
			for name, value := range tt.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}

			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
				t.Fatal(err)
			}
			if *requests != tt.wantRequests {
				t.Fatalf("requests = %d, want %d", *requests, tt.wantRequests)
			}
			if got := output.String(); got != "Updated issue #7: https://atomgit.com/alice/demo/issues/7\n" {
				t.Fatalf("output = %q", got)
			}
		})
	}
}

func TestIssueEditReadsBodyFile(t *testing.T) {
	bodyPath := filepath.Join(t.TempDir(), "issue.md")
	if err := os.WriteFile(bodyPath, []byte("Body from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	wantFields := map[string]string{
		"repo": "demo", "title": "Existing title", "body": "Body from file\n",
	}
	factory, _ := issueEditTestFactory(t, wantFields, http.StatusOK, true)
	cmd := newCmdIssueEdit(factory)
	cmd.SetOut(io.Discard)
	if err := cmd.Flags().Set("body-file", bodyPath); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
}

func TestIssueEditRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		wantError string
	}{
		{
			name:      "no updates",
			wantError: "at least one of --title, --body, --body-file, --assignee, or --remove-assignee must be provided",
		},
		{
			name:      "conflicting body sources",
			flags:     map[string]string{"body": "text", "body-file": "issue.md"},
			wantError: "--body and --body-file cannot be used together",
		},
		{
			name:      "empty title",
			flags:     map[string]string{"title": "   "},
			wantError: "title cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdIssueEdit(&cmdutil.Factory{})
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			for name, value := range tt.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			cmd.SetArgs([]string{"alice/demo", "7"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestIssueEditReportsBodyFileReadError(t *testing.T) {
	cmd := newCmdIssueEdit(&cmdutil.Factory{})
	if err := cmd.Flags().Set("body-file", filepath.Join(t.TempDir(), "missing.md")); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "failed to read body file") {
		t.Fatalf("error = %v", err)
	}
}

type issueEditAuthErrorConfig struct {
	issueTestConfig
}

func (issueEditAuthErrorConfig) GetToken() (string, error) {
	return "", errors.New("token not found")
}

func TestIssueEditReportsAuthenticationError(t *testing.T) {
	cmd := newCmdIssueEdit(&cmdutil.Factory{Config: issueEditAuthErrorConfig{}})
	if err := cmd.Flags().Set("title", "Updated title"); err != nil {
		t.Fatal(err)
	}

	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
}

func TestIssueEditReportsAPIErrors(t *testing.T) {
	tests := []struct {
		name      string
		patchCode int
		getCode   int
		wantError string
	}{
		{
			name:      "get current issue",
			getCode:   http.StatusNotFound,
			wantError: "failed to get issue before editing",
		},
		{
			name:      "update issue",
			getCode:   http.StatusOK,
			patchCode: http.StatusForbidden,
			wantError: "failed to edit issue",
		},
		{
			name:      "validate update",
			getCode:   http.StatusOK,
			patchCode: http.StatusUnprocessableEntity,
			wantError: "failed to edit issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method == http.MethodGet {
							return issueResponse(tt.getCode, `{"number":"7","title":"Existing title"}`), nil
						}
						return issueResponse(tt.patchCode, `{"message":"denied"}`), nil
					})}, nil
				},
			}
			cmd := newCmdIssueEdit(factory)
			flagName := "title"
			flagValue := "Updated title"
			if tt.name == "get current issue" {
				flagName = "body"
				flagValue = "Updated body"
			}
			if err := cmd.Flags().Set(flagName, flagValue); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			if output.Len() != 0 {
				t.Fatalf("unexpected success output: %q", output.String())
			}
		})
	}
}

func issueEditTestFactory(t *testing.T, wantFields map[string]string, patchStatus int, wantGet bool) (*cmdutil.Factory, *int) {
	t.Helper()
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				if req.Method == http.MethodGet {
					if !wantGet || requests != 1 {
						t.Fatalf("unexpected request #%d: %s %s", requests, req.Method, req.URL.Path)
					}
					if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/issues/7" {
						t.Fatalf("request = %s %s", req.Method, req.URL.Path)
					}
					return issueResponse(http.StatusOK, `{"number":"7","title":"Existing title","body":"Existing body","html_url":"https://atomgit.com/alice/demo/issues/7"}`), nil
				}
				wantPatchRequest := 1
				if wantGet {
					wantPatchRequest = 2
				}
				if requests != wantPatchRequest || req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/alice/issues/7" {
					t.Fatalf("unexpected request #%d: %s %s", requests, req.Method, req.URL.Path)
				}
				if err := req.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := len(req.MultipartForm.Value); got != len(wantFields) {
					t.Fatalf("form field count = %d, want %d: %#v", got, len(wantFields), req.MultipartForm.Value)
				}
				for key, want := range wantFields {
					values, ok := req.MultipartForm.Value[key]
					if !ok {
						t.Errorf("form field %s is missing", key)
						continue
					}
					if len(values) != 1 || values[0] != want {
						t.Errorf("form field %s = %#v, want %q", key, values, want)
					}
				}
				return &http.Response{
					StatusCode: patchStatus,
					Status:     http.StatusText(patchStatus),
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}
	return factory, &requests
}

func TestIssueEditAssigneeMutualExclusion(t *testing.T) {
	cmd := newCmdIssueEdit(&cmdutil.Factory{})
	if err := cmd.Flags().Set("assignee", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("remove-assignee", "true"); err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"alice/demo", "7"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--assignee and --remove-assignee cannot be used together") {
		t.Fatalf("error = %v, want mutual exclusion error", err)
	}
}

func TestIssueEditAssigneeRejectsInvalidNumberBeforeAuth(t *testing.T) {
	for _, number := range []string{"0", "-1", "abc"} {
		t.Run(number, func(t *testing.T) {
			config := &issueEditRecordingConfig{}
			cmd := newCmdIssueEdit(&cmdutil.Factory{Config: config})
			if err := cmd.Flags().Set("assignee", "alice"); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Flags().Set("yes", "true"); err != nil {
				t.Fatal(err)
			}

			err := cmd.RunE(cmd, []string{"alice/demo", number})
			if err == nil || !strings.Contains(err.Error(), "positive integer") {
				t.Fatalf("error = %v, want positive integer validation", err)
			}
			if config.getTokenCalls != 0 {
				t.Fatalf("GetToken called %d times", config.getTokenCalls)
			}
		})
	}
}

func TestIssueEditAssigneeConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		flags      map[string]string
		input      string
		wantPrompt bool
		wantDecl   bool
	}{
		{name: "set assignee with yes", flags: map[string]string{"assignee": "bob", "yes": "true"}, input: "", wantPrompt: false},
		{name: "set assignee confirm yes", flags: map[string]string{"assignee": "bob"}, input: "yes\n", wantPrompt: true},
		{name: "set assignee confirm y", flags: map[string]string{"assignee": "bob"}, input: "y\n", wantPrompt: true},
		{name: "set assignee decline no", flags: map[string]string{"assignee": "bob"}, input: "no\n", wantPrompt: true, wantDecl: true},
		{name: "set assignee decline empty", flags: map[string]string{"assignee": "bob"}, input: "\n", wantPrompt: true, wantDecl: true},
		{name: "set assignee decline eof", flags: map[string]string{"assignee": "bob"}, input: "", wantPrompt: true, wantDecl: true},
		{name: "remove assignee with yes", flags: map[string]string{"remove-assignee": "true", "yes": "true"}, input: "", wantPrompt: false},
		{name: "remove assignee decline", flags: map[string]string{"remove-assignee": "true"}, input: "no\n", wantPrompt: true, wantDecl: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{Config: issueEditAuthErrorConfig{}}
			cmd := newCmdIssueEdit(factory)
			cmd.SetIn(strings.NewReader(tt.input))
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			for name, value := range tt.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}

			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if tt.wantDecl {
				if err != nil {
					t.Fatalf("expected nil (decline returns nil), got %v", err)
				}
				if strings.TrimSpace(stderr.String()) == "" {
					t.Fatal("stderr prompt was not written on decline")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "not authenticated") {
				t.Fatalf("error = %v, want auth error (after confirmation)", err)
			}
			if tt.wantPrompt && !strings.Contains(stderr.String(), "Change assignee") {
				t.Fatalf("stderr = %q, want confirmation prompt", stderr.String())
			}
			if !tt.wantPrompt && strings.Contains(stderr.String(), "Change assignee") {
				t.Fatalf("stderr = %q, expected no prompt with --yes", stderr.String())
			}
		})
	}
}

func TestIssueEditAssigneeConfirmationToStderr(t *testing.T) {
	factory := &cmdutil.Factory{Config: issueEditAuthErrorConfig{}}
	cmd := newCmdIssueEdit(factory)
	cmd.SetIn(strings.NewReader("yes\n"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Flags().Set("assignee", "bob"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Change assignee") {
		t.Fatalf("stderr = %q, want confirmation prompt", stderr.String())
	}
}

func TestIssueEditAssigneeOnlyFetchesTitle(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method == http.MethodGet {
					if req.URL.Path != "/api/v5/repos/alice/demo/issues/7" {
						t.Fatalf("GET path = %s", req.URL.Path)
					}
					return issueResponse(http.StatusOK, `{"number":"7","title":"Existing title","html_url":"https://atomgit.com/alice/demo/issues/7"}`), nil
				}
				if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/alice/issues/7" {
					t.Fatalf("PATCH request = %s %s", req.Method, req.URL.Path)
				}
				if err := req.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := req.FormValue("repo"); got != "demo" {
					t.Fatalf("repo = %q", got)
				}
				if got := req.FormValue("title"); got != "Existing title" {
					t.Fatalf("title = %q", got)
				}
				if got := req.FormValue("assignee"); got != "bob" {
					t.Fatalf("assignee = %q", got)
				}
				return issueResponse(http.StatusOK, ""), nil
			})}, nil
		},
	}

	cmd := newCmdIssueEdit(factory)
	cmd.SetIn(strings.NewReader("yes\n"))
	if err := cmd.Flags().Set("assignee", "bob"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (1 GET + 1 PATCH)", requests)
	}
}

func TestIssueEditClearAssignee(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method == http.MethodGet {
					return issueResponse(http.StatusOK, `{"number":"7","title":"Existing title"}`), nil
				}
				if req.Method != http.MethodPatch {
					t.Fatalf("method = %s", req.Method)
				}
				if err := req.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := req.FormValue("assignee"); got != "" {
					t.Fatalf("assignee = %q, want empty string", got)
				}
				if got := req.FormValue("repo"); got != "demo" {
					t.Fatalf("repo = %q", got)
				}
				if got := req.FormValue("title"); got != "Existing title" {
					t.Fatalf("title = %q", got)
				}
				return issueResponse(http.StatusOK, ""), nil
			})}, nil
		},
	}

	cmd := newCmdIssueEdit(factory)
	cmd.SetIn(strings.NewReader("yes\n"))
	if err := cmd.Flags().Set("remove-assignee", "true"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (1 GET + 1 PATCH)", requests)
	}
}

func TestIssueEditCombinedAssigneesTitle(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method == http.MethodGet {
					t.Fatal("unexpected GET when title is provided")
				}
				if req.URL.Path != "/api/v5/repos/alice/issues/7" {
					t.Fatalf("PATCH path = %s", req.URL.Path)
				}
				if err := req.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := req.FormValue("assignee"); got != "bob" {
					t.Fatalf("assignee = %q", got)
				}
				if got := req.FormValue("title"); got != "New title" {
					t.Fatalf("title = %q", got)
				}
				if got := req.FormValue("repo"); got != "demo" {
					t.Fatalf("repo = %q", got)
				}
				return issueResponse(http.StatusOK, ""), nil
			})}, nil
		},
	}

	cmd := newCmdIssueEdit(factory)
	cmd.SetIn(strings.NewReader("yes\n"))
	if err := cmd.Flags().Set("assignee", "bob"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("title", "New title"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestIssueEditRejectsInvalidOptionsWithAssignee(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		wantError string
	}{
		{
			name:      "empty assignee",
			flags:     map[string]string{"assignee": "   "},
			wantError: "assignee cannot be empty",
		},
		{
			name:      "no updates when only yes",
			flags:     map[string]string{"yes": "true"},
			wantError: "at least one of --title, --body, --body-file, --assignee, or --remove-assignee must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdIssueEdit(&cmdutil.Factory{})
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			for name, value := range tt.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			cmd.SetArgs([]string{"alice/demo", "7"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}
