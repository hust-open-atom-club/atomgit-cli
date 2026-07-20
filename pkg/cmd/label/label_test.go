package label

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type labelTestConfig struct{}

func (labelTestConfig) GetToken() (string, error) { return "token", nil }
func (labelTestConfig) GetUser() (string, error)  { return "alice", nil }
func (labelTestConfig) GetHost() string           { return "atomgit.com" }

type labelRoundTripFunc func(*http.Request) (*http.Response, error)

func (f labelRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewCmdLabelRegistersCommands(t *testing.T) {
	cmd := NewCmdLabel(&cmdutil.Factory{})
	want := map[string][]string{
		"list":   {"limit"},
		"create": {"name", "color"},
		"edit":   {"name", "color"},
		"delete": {"yes"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Fatalf("%s --%s flag was not registered", name, flag)
			}
		}
		if !strings.Contains(child.Example, "ag label "+name) {
			t.Fatalf("%s example = %q", name, child.Example)
		}
		if child.Flags().Lookup("description") != nil {
			t.Fatalf("%s registered unsupported --description flag", name)
		}
	}
}

func TestLabelListHonorsLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: labelTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: labelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/labels" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				return labelResponse(http.StatusOK, `[
					{"name":"bug","color":"#ff0000","description":"Defect"},
					{"name":"help wanted","color":"#2865E0","description":""}
				]`), nil
			})}, nil
		},
	}

	cmd := newCmdLabelList(factory)
	if err := cmd.Flags().Set("limit", "1"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if got := output.String(); got != "bug [#ff0000] Defect\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestLabelListRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		limit     string
		wantError string
	}{
		{name: "missing repository", wantError: "unable to determine repository"},
		{name: "invalid repository", args: []string{"demo"}, wantError: "invalid repository format"},
		{name: "zero limit", args: []string{"alice/demo"}, limit: "0", wantError: "must be positive"},
		{name: "negative limit", args: []string{"alice/demo"}, limit: "-1", wantError: "must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdLabelList(&cmdutil.Factory{Config: labelTestConfig{}})
			if tt.limit != "" {
				if err := cmd.Flags().Set("limit", tt.limit); err != nil {
					t.Fatal(err)
				}
			}
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestLabelCreate(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: labelTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: labelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/labels" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
					t.Fatalf("Content-Type = %q", got)
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				values, err := url.ParseQuery(string(body))
				if err != nil {
					t.Fatal(err)
				}
				if values.Get("name") != "bug" || values.Get("color") != "#ff0000" {
					t.Fatalf("form = %#v", values)
				}
				return labelResponse(http.StatusOK, `{"id":1,"name":"bug","color":"#ff0000"}`), nil
			})}, nil
		},
	}

	cmd := newCmdLabelCreate(factory)
	_ = cmd.Flags().Set("name", " bug ")
	_ = cmd.Flags().Set("color", "#ff0000")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created label \"bug\" [#ff0000]\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestLabelEdit(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: labelTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: labelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/api/v5/repos/alice/demo/labels/help%20wanted" {
					t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
				}
				if req.URL.Query().Get("name") != "support" || req.URL.Query().Get("color") != "#abc" {
					t.Fatalf("query = %#v", req.URL.Query())
				}
				return labelResponse(http.StatusOK, `{}`), nil
			})}, nil
		},
	}

	cmd := newCmdLabelEdit(factory)
	_ = cmd.Flags().Set("name", "support")
	_ = cmd.Flags().Set("color", "#abc")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "help wanted"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Updated label \"help wanted\"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestLabelMutationsReportAPIErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		command   func(*cmdutil.Factory) *cobra.Command
		args      []string
		wantError string
	}{
		{
			name:   "create duplicate",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"label already exists"}`,
			command: func(factory *cmdutil.Factory) *cobra.Command {
				cmd := newCmdLabelCreate(factory)
				_ = cmd.Flags().Set("name", "bug")
				_ = cmd.Flags().Set("color", "#ff0000")
				return cmd
			},
			args: []string{"alice/demo"}, wantError: "failed to create label",
		},
		{
			name:   "edit missing label",
			status: http.StatusNotFound,
			body:   `{"message":"label not found"}`,
			command: func(factory *cmdutil.Factory) *cobra.Command {
				cmd := newCmdLabelEdit(factory)
				_ = cmd.Flags().Set("color", "#abc")
				return cmd
			},
			args: []string{"alice/demo", "missing"}, wantError: "failed to edit label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: labelTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: labelRoundTripFunc(func(*http.Request) (*http.Response, error) {
						return labelResponse(tt.status, tt.body), nil
					})}, nil
				},
			}
			cmd := tt.command(factory)
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) || !strings.Contains(err.Error(), http.StatusText(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLabelDelete(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: labelTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: labelRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/api/v5/repos/alice/demo/labels/help%20wanted" {
					t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
				}
				return labelResponse(http.StatusOK, `{}`), nil
			})}, nil
		},
	}

	cmd := newCmdLabelDelete(factory)
	_ = cmd.Flags().Set("yes", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "help wanted"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
	if got := output.String(); got != "Deleted label \"help wanted\"\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestLabelDeleteConfirmation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantRequests int
		wantOutput   string
	}{
		{name: "accept", input: "yes\n", wantRequests: 1, wantOutput: "Deleted label \"obsolete\"\n"},
		{name: "cancel", input: "n\n", wantOutput: "Deletion cancelled.\n"},
		{name: "empty input", wantOutput: "Deletion cancelled.\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: labelTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: labelRoundTripFunc(func(*http.Request) (*http.Response, error) {
						requests++
						return labelResponse(http.StatusOK, `{}`), nil
					})}, nil
				},
			}
			cmd := newCmdLabelDelete(factory)
			cmd.SetIn(strings.NewReader(tt.input))
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "obsolete"}); err != nil {
				t.Fatal(err)
			}
			if requests != tt.wantRequests {
				t.Fatalf("requests = %d, want %d", requests, tt.wantRequests)
			}
			if got := output.String(); !strings.HasSuffix(got, tt.wantOutput) {
				t.Fatalf("output = %q, want suffix %q", got, tt.wantOutput)
			}
		})
	}
}

func TestLabelDeleteReportsAPIError(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: labelTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: labelRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return labelResponse(http.StatusForbidden, `{"message":"denied"}`), nil
			})}, nil
		},
	}
	cmd := newCmdLabelDelete(factory)
	_ = cmd.Flags().Set("yes", "true")
	err := cmd.RunE(cmd, []string{"alice/demo", "protected"})
	if err == nil || !strings.Contains(err.Error(), "failed to delete label") || !strings.Contains(err.Error(), "Forbidden") {
		t.Fatalf("error = %v", err)
	}
}

func TestLabelMutationsRejectInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		command   func() *cobra.Command
		args      []string
		wantError string
	}{
		{
			name: "create missing name",
			command: func() *cobra.Command {
				cmd := newCmdLabelCreate(&cmdutil.Factory{})
				_ = cmd.Flags().Set("color", "#fff")
				return cmd
			},
			args: []string{"alice/demo"}, wantError: "label name is required",
		},
		{
			name: "create invalid color",
			command: func() *cobra.Command {
				cmd := newCmdLabelCreate(&cmdutil.Factory{})
				_ = cmd.Flags().Set("name", "bug")
				_ = cmd.Flags().Set("color", "ff0000")
				return cmd
			},
			args: []string{"alice/demo"}, wantError: "invalid label color",
		},
		{
			name: "edit no changes",
			command: func() *cobra.Command {
				return newCmdLabelEdit(&cmdutil.Factory{})
			},
			args: []string{"alice/demo", "bug"}, wantError: "at least one",
		},
		{
			name: "edit empty new name",
			command: func() *cobra.Command {
				cmd := newCmdLabelEdit(&cmdutil.Factory{})
				_ = cmd.Flags().Set("name", "  ")
				return cmd
			},
			args: []string{"alice/demo", "bug"}, wantError: "label name is required",
		},
		{
			name: "edit invalid color",
			command: func() *cobra.Command {
				cmd := newCmdLabelEdit(&cmdutil.Factory{})
				_ = cmd.Flags().Set("color", "#abcd")
				return cmd
			},
			args: []string{"alice/demo", "bug"}, wantError: "invalid label color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.command()
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func labelResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
