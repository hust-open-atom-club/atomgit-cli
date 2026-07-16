package label

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type labelTestConfig struct{}

func (labelTestConfig) GetToken() (string, error) { return "token", nil }
func (labelTestConfig) GetUser() (string, error)  { return "alice", nil }
func (labelTestConfig) GetHost() string           { return "atomgit.com" }

type labelRoundTripFunc func(*http.Request) (*http.Response, error)

func (f labelRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewCmdLabelRegistersList(t *testing.T) {
	cmd := NewCmdLabel(&cmdutil.Factory{})
	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Flags().Lookup("limit") == nil {
		t.Fatal("list limit flag was not registered")
	}
	if !strings.Contains(list.Example, "ag label list") {
		t.Fatalf("list example = %q", list.Example)
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
		{name: "missing repository", wantError: "repository required"},
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

func labelResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
