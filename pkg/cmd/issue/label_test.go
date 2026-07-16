package issue

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestIssueLabelAddsLabel(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/labels" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				if got := req.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q", got)
				}
				var labels []string
				if err := json.NewDecoder(req.Body).Decode(&labels); err != nil {
					t.Fatal(err)
				}
				want := []string{"bug", "help wanted", "priority/high"}
				if len(labels) != len(want) {
					t.Fatalf("labels = %#v, want %#v", labels, want)
				}
				for i := range want {
					if labels[i] != want[i] {
						t.Fatalf("labels = %#v, want %#v", labels, want)
					}
				}
				return issueResponse(http.StatusOK, `[{"name":"bug"},{"name":"help wanted"},{"name":"priority/high"}]`), nil
			})}, nil
		},
	}

	cmd := newCmdIssueLabel(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7", " bug, help wanted ,priority/high "}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Added labels to issue #7: bug, help wanted, priority/high\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestIssueLabelRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "invalid repository", args: []string{"demo", "7", "bug"}, wantError: "invalid repository format"},
		{name: "empty label", args: []string{"alice/demo", "7", "   "}, wantError: "label cannot be empty"},
		{name: "empty item", args: []string{"alice/demo", "7", "bug, ,feat"}, wantError: "label cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdIssueLabel(&cmdutil.Factory{})
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestIssueLabelReportsAPIError(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
					Status:     "422 Unprocessable Entity",
					Body:       io.NopCloser(strings.NewReader(`{"message":"label not found"}`)),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdIssueLabel(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	err := cmd.RunE(cmd, []string{"alice/demo", "7", "missing"})
	if err == nil || !strings.Contains(err.Error(), "failed to add labels to issue") {
		t.Fatalf("error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected success output: %q", output.String())
	}
}
