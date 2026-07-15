package issue

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestIssueEditUpdatesOnlyRequestedFields(t *testing.T) {
	tests := []struct {
		name         string
		flags        map[string]string
		wantFields   map[string]interface{}
		wantRequests int
	}{
		{
			name:         "title only",
			flags:        map[string]string{"title": "Updated title"},
			wantRequests: 1,
			wantFields: map[string]interface{}{
				"repo": "demo", "title": "Updated title",
			},
		},
		{
			name:         "body only",
			flags:        map[string]string{"body": "Updated body"},
			wantRequests: 2,
			wantFields: map[string]interface{}{
				"repo": "demo", "title": "Existing title", "body": "Updated body",
			},
		},
		{
			name:         "clear body",
			flags:        map[string]string{"body": ""},
			wantRequests: 2,
			wantFields: map[string]interface{}{
				"repo": "demo", "title": "Existing title", "body": "",
			},
		},
		{
			name:         "title and body",
			flags:        map[string]string{"title": "Updated title", "body": "Updated body"},
			wantRequests: 1,
			wantFields: map[string]interface{}{
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

	wantFields := map[string]interface{}{
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
			wantError: "at least one of --title, --body, or --body-file must be provided",
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
			for name, value := range tt.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
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

func issueEditTestFactory(t *testing.T, wantFields map[string]interface{}, patchStatus int, wantGet bool) (*cmdutil.Factory, *int) {
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
				if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
					t.Fatalf("Content-Type = %q", contentType)
				}
				var fields map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(fields, wantFields) {
					t.Fatalf("fields = %#v, want %#v", fields, wantFields)
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
