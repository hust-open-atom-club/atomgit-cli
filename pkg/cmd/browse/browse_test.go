package browse

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type browseTestConfig struct{}

func (browseTestConfig) GetToken() (string, error) { return "token", nil }
func (browseTestConfig) GetUser() (string, error)  { return "alice", nil }
func (browseTestConfig) GetHost() string           { return "atomgit.com" }

type browseRoundTripFunc func(*http.Request) (*http.Response, error)

func (f browseRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func mockRepoHTTPClient(defaultBranch string) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		return &http.Client{Transport: browseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v5/repos/alice/demo" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"default_branch": "` + defaultBranch + `"}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		})}, nil
	}
}

func TestBrowseExplicitRepoHome(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseIssueNumber(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: browseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/api/v5/repos/alice/demo/issues/42" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}, nil
		},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "42"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/issues/42" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowsePRNumber(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: browseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Path {
				case "/api/v5/repos/alice/demo/issues/45":
					return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
				case "/api/v5/repos/alice/demo/pulls/45":
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
				}
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}, nil
		},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "45"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/pull/45" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseNonexistentNumber(t *testing.T) {
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: browseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "99"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no issue or pull request with number 99 found") {
		t.Fatalf("error = %v", err)
	}
}

func TestBrowseNumberAPICallsReturnError(t *testing.T) {
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: browseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/api/v5/repos/alice/demo/issues/42" {
					return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
				}
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "42"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unexpected status checking issue #42") {
		t.Fatalf("error = %v, want unexpected status error", err)
	}
}

func TestBrowseFilePath(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config:       browseTestConfig{},
		HttpClient:   mockRepoHTTPClient("main"),
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "main.go"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/blob/main/main.go" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseFileWithLine(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config:       browseTestConfig{},
		HttpClient:   mockRepoHTTPClient("main"),
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "main.go:312"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedURL, "#L312") {
		t.Fatalf("URL = %q, want #L312", capturedURL)
	}
}

func TestBrowseFileWithLineRange(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config:       browseTestConfig{},
		HttpClient:   mockRepoHTTPClient("main"),
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "main.go:312-320"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedURL, "#L312-L320") {
		t.Fatalf("URL = %q, want #L312-L320", capturedURL)
	}
}

func TestBrowseNonMainDefaultBranch(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config:       browseTestConfig{},
		HttpClient:   mockRepoHTTPClient("develop"),
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "README.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/blob/develop/README.md" {
		t.Fatalf("URL = %q, want https://atomgit.com/alice/demo/blob/develop/README.md", capturedURL)
	}
}

func TestBrowseCommit(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "abc123def456"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/commit/abc123def456" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseCommitFlag(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "-c", "abc1234"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/commit/abc1234" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseCommitFlagWithPath(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "-c", "abc1234", "main.go"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/blob/abc1234/main.go" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseReleases(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "-r"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/releases" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestBrowseNoBrowser(t *testing.T) {
	openerCalled := false
	f := &cmdutil.Factory{
		Config: browseTestConfig{},
		BrowserOpener: func(rawURL string) error {
			openerCalled = true
			return nil
		},
	}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "alice/demo", "-n"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if openerCalled {
		t.Fatal("browser was opened with --no-browser")
	}
}

func TestBrowseNoRepoError(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	f := &cmdutil.Factory{Config: browseTestConfig{}}
	cmd := NewCmdBrowse(f)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no repository specified") {
		t.Fatalf("error = %v", err)
	}
}

func TestBrowseInvalidRepoFlag(t *testing.T) {
	f := &cmdutil.Factory{Config: browseTestConfig{}}
	cmd := NewCmdBrowse(f)
	cmd.SetArgs([]string{"-R", "invalid"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid --repo format") {
		t.Fatalf("error = %v, want invalid format", err)
	}
}

func TestClassifyArg(t *testing.T) {
	tests := []struct {
		arg  string
		want argType
	}{
		{"42", argTypeNumber},
		{"abc123def456", argTypeCommit},
		{"abc123", argTypeCommit},
		{"abcdefabcdefabcdefabcdefabcdefabcdefabcd", argTypeCommit},
		{"main.go", argTypePath},
		{"pkg/cmd/main.go", argTypePath},
		{"abc", argTypePath},
		{"abc12", argTypePath},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := classifyArg(tt.arg); got != tt.want {
				t.Errorf("classifyArg(%q) = %d, want %d", tt.arg, got, tt.want)
			}
		})
	}
}

func TestParseFilePathArg(t *testing.T) {
	tests := []struct {
		arg                string
		wantPath           string
		wantStart, wantEnd int
	}{
		{"main.go", "main.go", 0, 0},
		{"main.go:312", "main.go", 312, 0},
		{"main.go:312-320", "main.go", 312, 320},
		{"main.go:312..320", "main.go", 312, 320},
		{"pkg/cmd/main.go:42", "pkg/cmd/main.go", 42, 0},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			path, start, end := parseFilePathArg(tt.arg)
			if path != tt.wantPath || start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("parseFilePathArg(%q) = (%q, %d, %d), want (%q, %d, %d)",
					tt.arg, path, start, end, tt.wantPath, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
