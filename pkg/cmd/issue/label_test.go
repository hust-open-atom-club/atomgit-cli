package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
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
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/labels" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var labels []string
				if err := json.NewDecoder(req.Body).Decode(&labels); err != nil {
					t.Fatal(err)
				}
				assertStringSlice(t, labels, []string{"bug", "help wanted", "priority/high"})
				return issueResponse(http.StatusOK, `{}`), nil
			})}, nil
		},
	}

	cmd := newCmdIssueLabel(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7", " bug, help wanted ,priority/high,bug "}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
	if got := output.String(); got != "Added labels to issue #7: bug, help wanted, priority/high\n" {
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
				if req.Method != http.MethodDelete {
					t.Fatalf("method = %s, want DELETE", req.Method)
				}
				wantPaths := []string{
					"/api/v5/repos/alice/demo/issues/7/labels/bug",
					"/api/v5/repos/alice/demo/issues/7/labels/priority%2Fhigh",
				}
				if requests > len(wantPaths) || req.URL.EscapedPath() != wantPaths[requests-1] {
					t.Fatalf("request %d path = %s", requests, req.URL.EscapedPath())
				}
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
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if got := output.String(); got != "Removed labels from issue #7: bug, priority/high\n" {
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

func TestIssueLabelReportsAPIErrors(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		factory := &cmdutil.Factory{
			Config: issueTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPost {
						t.Fatalf("method = %s, want POST", req.Method)
					}
					return issueLabelErrorResponse(http.StatusUnprocessableEntity, `{"message":"label not found"}`), nil
				})}, nil
			},
		}
		cmd := newCmdIssueLabel(factory)
		_ = cmd.Flags().Set("add", "missing")
		err := cmd.RunE(cmd, []string{"alice/demo", "7"})
		if err == nil || !strings.Contains(err.Error(), "failed to add labels to issue") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("remove", func(t *testing.T) {
		requests := 0
		factory := &cmdutil.Factory{
			Config: issueTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					requests++
					switch requests {
					case 1:
						if req.Method != http.MethodDelete {
							t.Fatalf("method = %s, want DELETE", req.Method)
						}
						return issueResponse(http.StatusOK, `{}`), nil
					case 2:
						if req.Method != http.MethodDelete {
							t.Fatalf("method = %s, want DELETE", req.Method)
						}
						return issueLabelErrorResponse(http.StatusNotFound, `{"message":"label not found"}`), nil
					case 3:
						if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/labels" {
							t.Fatalf("rollback request = %s %s", req.Method, req.URL.Path)
						}
						var labels []string
						if err := json.NewDecoder(req.Body).Decode(&labels); err != nil {
							t.Fatal(err)
						}
						assertStringSlice(t, labels, []string{"bug"})
						return issueResponse(http.StatusOK, `{}`), nil
					default:
						t.Fatalf("unexpected request %d", requests)
						return nil, nil
					}
				})}, nil
			},
		}
		cmd := newCmdIssueLabel(factory)
		_ = cmd.Flags().Set("remove", "bug,missing")
		err := cmd.RunE(cmd, []string{"alice/demo", "7"})
		if err == nil || !strings.Contains(err.Error(), `failed to remove label "missing"`) || !strings.Contains(err.Error(), "restored previously removed labels: bug") {
			t.Fatalf("error = %v", err)
		}
		if requests != 3 {
			t.Fatalf("requests = %d, want 3", requests)
		}
	})

	t.Run("remove rollback failure", func(t *testing.T) {
		requests := 0
		factory := &cmdutil.Factory{
			Config: issueTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					requests++
					switch requests {
					case 1:
						return issueResponse(http.StatusOK, `{}`), nil
					case 2:
						return issueLabelErrorResponse(http.StatusNotFound, `{"message":"label not found"}`), nil
					case 3:
						if req.Method != http.MethodPost {
							t.Fatalf("method = %s, want POST", req.Method)
						}
						return issueLabelErrorResponse(http.StatusInternalServerError, `{"message":"rollback failed"}`), nil
					default:
						t.Fatalf("unexpected request %d", requests)
						return nil, nil
					}
				})}, nil
			},
		}
		cmd := newCmdIssueLabel(factory)
		_ = cmd.Flags().Set("remove", "bug,missing")
		err := cmd.RunE(cmd, []string{"alice/demo", "7"})
		if err == nil || !strings.Contains(err.Error(), "failed to restore previously removed labels") || !strings.Contains(err.Error(), "rollback failed") {
			t.Fatalf("error = %v", err)
		}
		if requests != 3 {
			t.Fatalf("requests = %d, want 3", requests)
		}
	})
}

func issueLabelErrorResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
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
