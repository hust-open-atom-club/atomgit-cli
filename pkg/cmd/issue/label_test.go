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

func TestIssueLabelFlags(t *testing.T) {
	cmd := newCmdIssueLabel(&cmdutil.Factory{})
	for _, name := range []string{"add", "remove"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag was not registered", name)
		}
	}
	if !strings.Contains(cmd.Example, "--add") || !strings.Contains(cmd.Example, "--remove") {
		t.Fatalf("examples = %q", cmd.Example)
	}
}

func TestIssueLabelAddsLabels(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				switch requests {
				case 1:
					if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/issues/7" {
						t.Fatalf("request = %s %s", req.Method, req.URL.Path)
					}
					return issueResponse(http.StatusOK, `{"labels":[{"name":"existing"},{"name":"bug"}]}`), nil
				case 2:
					if req.Method != http.MethodPut || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/labels" {
						t.Fatalf("request = %s %s", req.Method, req.URL.Path)
					}
					var labels []string
					if err := json.NewDecoder(req.Body).Decode(&labels); err != nil {
						t.Fatal(err)
					}
					want := []string{"existing", "bug", "help wanted", "priority/high"}
					assertStringSlice(t, labels, want)
					return issueResponse(http.StatusOK, `{}`), nil
				default:
					t.Fatalf("unexpected request %d", requests)
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdIssueLabel(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7", " bug, help wanted ,priority/high,bug "}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
	if got := output.String(); got != "Added labels to issue #7: help wanted, priority/high\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestIssueLabelRemovesLabels(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					return issueResponse(http.StatusOK, `{"labels":[{"name":"bug"},{"name":"help wanted"},{"name":"priority/high"}]}`), nil
				}
				if req.Method != http.MethodPut || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/labels" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var labels []string
				if err := json.NewDecoder(req.Body).Decode(&labels); err != nil {
					t.Fatal(err)
				}
				assertStringSlice(t, labels, []string{"help wanted"})
				return issueResponse(http.StatusOK, `{}`), nil
			})}, nil
		},
	}

	cmd := newCmdIssueLabel(factory)
	_ = cmd.Flags().Set("remove", "bug, priority/high")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Removed labels from issue #7: bug, priority/high\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestIssueLabelNoOpAdd(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return issueResponse(http.StatusOK, `{"labels":[{"name":"bug"}]}`), nil
			})}, nil
		},
	}
	cmd := newCmdIssueLabel(factory)
	_ = cmd.Flags().Set("add", "bug")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only the issue lookup", requests)
	}
	if got := output.String(); got != "Issue #7 already has labels: bug\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestIssueLabelRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		add       *string
		remove    *string
		wantError string
	}{
		{name: "invalid repository", args: []string{"demo", "7", "bug"}, wantError: "invalid repository format"},
		{name: "invalid issue number", args: []string{"alice/demo", "zero", "bug"}, wantError: "invalid issue number"},
		{name: "missing labels", args: []string{"alice/demo", "7"}, wantError: "labels are required"},
		{name: "empty label", args: []string{"alice/demo", "7", "   "}, wantError: "label cannot be empty"},
		{name: "empty item", args: []string{"alice/demo", "7", "bug, ,feat"}, wantError: "label cannot be empty"},
		{name: "both operations", args: []string{"alice/demo", "7"}, add: stringPointer("bug"), remove: stringPointer("feat"), wantError: "cannot be used together"},
		{name: "positional with flag", args: []string{"alice/demo", "7", "bug"}, add: stringPointer("feat"), wantError: "positional labels cannot be used"},
		{name: "empty add flag", args: []string{"alice/demo", "7"}, add: stringPointer(""), wantError: "label cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdIssueLabel(&cmdutil.Factory{})
			if tt.add != nil {
				_ = cmd.Flags().Set("add", *tt.add)
			}
			if tt.remove != nil {
				_ = cmd.Flags().Set("remove", *tt.remove)
			}
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestIssueLabelRejectsMissingRemoval(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return issueResponse(http.StatusOK, `{"labels":[{"name":"bug"}]}`), nil
			})}, nil
		},
	}
	cmd := newCmdIssueLabel(factory)
	_ = cmd.Flags().Set("remove", "missing")
	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "labels not found on issue: missing") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only the issue lookup", requests)
	}
}

func TestIssueLabelReportsAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		failOnCall int
		wantError  string
	}{
		{name: "issue lookup", failOnCall: 1, wantError: "failed to get issue labels"},
		{name: "label update", failOnCall: 2, wantError: "failed to add labels on issue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
						requests++
						if requests == tt.failOnCall {
							return &http.Response{
								StatusCode: http.StatusUnprocessableEntity,
								Status:     "422 Unprocessable Entity",
								Body:       io.NopCloser(strings.NewReader(`{"message":"label not found"}`)),
								Header:     make(http.Header),
							}, nil
						}
						return issueResponse(http.StatusOK, `{"labels":[]}`), nil
					})}, nil
				},
			}

			cmd := newCmdIssueLabel(factory)
			_ = cmd.Flags().Set("add", "missing")
			var output bytes.Buffer
			cmd.SetOut(&output)
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v", err)
			}
			if output.Len() != 0 {
				t.Fatalf("unexpected success output: %q", output.String())
			}
		})
	}
}

func TestUpdateIssueLabels(t *testing.T) {
	updated, changed, err := updateIssueLabels([]string{"bug"}, []string{"bug", "feature"}, issueLabelAdd)
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, updated, []string{"bug", "feature"})
	assertStringSlice(t, changed, []string{"feature"})

	updated, changed, err = updateIssueLabels([]string{"bug", "feature"}, []string{"bug"}, issueLabelRemove)
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlice(t, updated, []string{"feature"})
	assertStringSlice(t, changed, []string{"bug"})
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice = %#v, want %#v", got, want)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
