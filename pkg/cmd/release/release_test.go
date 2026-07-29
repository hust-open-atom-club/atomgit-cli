package release

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type releaseTestConfig struct{}

func (releaseTestConfig) GetToken() (string, error) { return "token", nil }
func (releaseTestConfig) GetUser() (string, error)  { return "alice", nil }
func (releaseTestConfig) GetHost() string           { return "atomgit.com" }

type recordingReleaseConfig struct{ getTokenCalls int }

func (c *recordingReleaseConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}
func (*recordingReleaseConfig) GetUser() (string, error) { return "alice", nil }
func (*recordingReleaseConfig) GetHost() string          { return "atomgit.com" }

type releaseRoundTripFunc func(*http.Request) (*http.Response, error)

func (f releaseRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func releaseResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func releaseTestFactoryForRepository(transport http.RoundTripper, owner, repo string) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: releaseTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: owner, Name: repo}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
}

func releaseTestFactory(transport http.RoundTripper) *cmdutil.Factory {
	return releaseTestFactoryForRepository(transport, "alice", "demo")
}

func staticRepoFactory(owner, repo string) *cmdutil.Factory {
	transport := releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return releaseResponse(http.StatusOK, `[]`), nil
	})
	return releaseTestFactoryForRepository(transport, owner, repo)
}

func TestNewCmdReleaseRegistersCommands(t *testing.T) {
	cmd := NewCmdRelease(&cmdutil.Factory{})
	if !strings.Contains(cmd.Long, "#18") || !strings.Contains(cmd.Long, "does not") {
		t.Fatalf("release Long does not explain the #18 boundary: %q", cmd.Long)
	}
	wantExample := `  ag release list owner/repo
  ag release view owner/repo v1.0.0
  ag release create owner/repo v1.0.0 --name "Version 1.0.0" --body "Release notes"
  ag release upload owner/repo v1.0.0 ./dist/app.tar.gz`
	if cmd.Example != wantExample {
		t.Fatalf("release Example = %q, want %q", cmd.Example, wantExample)
	}
	want := map[string]bool{"list": false, "view": false, "create": false, "edit": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}
	list, _, err := cmd.Find([]string{"list"})
	if err != nil || list == nil {
		t.Fatal("release list not found")
	}
	if flag := list.Flags().Lookup("limit"); flag == nil {
		t.Fatal("release list --limit flag was not registered")
	} else if flag.Shorthand != "L" {
		t.Fatalf("release list --limit shorthand = %q, want L", flag.Shorthand)
	}
	view, _, err := cmd.Find([]string{"view"})
	if err != nil || view == nil {
		t.Fatal("release view not found")
	}
	for _, child := range []*cobra.Command{list, view} {
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s Long missing repository context help: %q", child.Name(), child.Long)
		}
	}
}

func TestReleaseListRejectsInvalidLimit(t *testing.T) {
	cases := []struct {
		name  string
		limit string
	}{
		{name: "zero", limit: "0"},
		{name: "negative", limit: "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCmdReleaseList(staticRepoFactory("alice", "demo"))
			if err := cmd.Flags().Set("limit", tc.limit); err != nil {
				t.Fatal(err)
			}
			err := cmd.RunE(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v, want containing %q", err, "must be positive")
			}
		})
	}
}

func TestReleaseListResolvesRepositoryBeforeAuthentication(t *testing.T) {
	cfg := &recordingReleaseConfig{}
	cmd := newCmdReleaseList(&cmdutil.Factory{Config: cfg})
	err := cmd.RunE(cmd, []string{"demo"})
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want invalid repository format", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; repository resolution must finish before authentication", cfg.getTokenCalls)
	}
}

func TestReleaseListUsesPaginationQueryAndOutputsRows(t *testing.T) {
	var gotPath, gotPage, gotPerPage, gotDirection string
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotPath = req.URL.Path
				gotPage = req.URL.Query().Get("page")
				gotPerPage = req.URL.Query().Get("per_page")
				gotDirection = req.URL.Query().Get("direction")
				return releaseResponse(http.StatusOK, `[
					{"tag_name":"v1.0.0","name":"First","prerelease":false,"release_status":"latest","created_at":"2026-01-01T00:00:00Z"},
					{"tag_name":"v0.9.0-rc","name":"RC","prerelease":true,"release_status":"pre","created_at":"2026-02-01T00:00:00Z"}
				]`), nil
			})}, nil
		},
	}

	cmd := newCmdReleaseList(factory)
	if err := cmd.Flags().Set("limit", "5"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v5/repos/alice/demo/releases" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPage != "1" || gotPerPage != "100" {
		t.Fatalf("page/per_page = %q/%q", gotPage, gotPerPage)
	}
	if gotDirection != "desc" {
		t.Fatalf("direction = %q, want desc", gotDirection)
	}
	want := "v1.0.0\tFirst\tlatest\t2026-01-01T00:00:00Z\n" +
		"v0.9.0-rc\tRC\tprerelease\t2026-02-01T00:00:00Z\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseListEmptyResults(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return releaseResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdReleaseList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "No releases found\n" {
		t.Fatalf("output = %q, want %q", got, "No releases found\n")
	}
}

func TestReleaseStatusDerivation(t *testing.T) {
	cases := []struct {
		name string
		r    api.Release
		want string
	}{
		{name: "prerelease flag true", r: api.Release{Prerelease: true}, want: "prerelease"},
		{name: "release_status pre", r: api.Release{ReleaseStatus: api.ReleaseStatusPre}, want: "prerelease"},
		{name: "prerelease flag true with latest status", r: api.Release{Prerelease: true, ReleaseStatus: api.ReleaseStatusLatest}, want: "prerelease"},
		{name: "release_status latest", r: api.Release{ReleaseStatus: api.ReleaseStatusLatest}, want: "latest"},
		{name: "empty release_status", r: api.Release{}, want: "release"},
		{name: "none release_status", r: api.Release{ReleaseStatus: "none"}, want: "release"},
		{name: "unknown release_status", r: api.Release{ReleaseStatus: "draft"}, want: "release"},
		{name: "prerelease flag false with pre status", r: api.Release{Prerelease: false, ReleaseStatus: api.ReleaseStatusPre}, want: "prerelease"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := releaseStatus(tc.r); got != tc.want {
				t.Fatalf("releaseStatus = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReleaseViewInferRepoAndEscapeTag(t *testing.T) {
	var gotPath string
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "atomclub", Name: "ag"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotPath = req.URL.EscapedPath()
				return releaseResponse(http.StatusOK, `{
					"tag_name":"v1/rc","target_commitish":"main","prerelease":false,
					"name":"First","body":"release notes","release_status":"latest",
					"created_at":"2026-01-01T00:00:00Z",
					"author":{"login":"alice","name":"Alice"},
					"assets":[{"name":"a.tar.gz","type":"attach","browser_download_url":"https://raw/a.tar.gz"}]
				}`), nil
			})}, nil
		},
	}
	cmd := newCmdReleaseView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"v1/rc"}); err != nil {
		t.Fatal(err)
	}
	const wantPath = "/api/v5/repos/atomclub/ag/releases/tags/v1%2Frc"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
}

func TestReleaseViewOutputsAllFields(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return releaseResponse(http.StatusOK, `{
					"tag_name":"v2.0.0","target_commitish":"main","prerelease":true,
					"name":"Second","body":"notes v2","release_status":"pre",
					"created_at":"2026-03-01T00:00:00Z",
					"author":{"login":"bob","name":"Bob"},
					"assets":[
						{"name":"a.zip","type":"attach","browser_download_url":"https://raw/a.zip"},
						{"name":"src.tar.gz","type":"source","browser_download_url":"https://raw/src.tar.gz"}
					]
				}`), nil
			})}, nil
		},
	}
	cmd := newCmdReleaseView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"atomclub/ag", "v2.0.0"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Name: Second",
		"Tag: v2.0.0",
		"Target: main",
		"Status: prerelease",
		"Created: 2026-03-01T00:00:00Z",
		"Author: Bob (bob)",
		"Body: notes v2",
		"Assets:",
		"  a.zip\tattach\thttps://raw/a.zip",
		"  src.tar.gz\tsource\thttps://raw/src.tar.gz",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\noutput=%s", want, got)
		}
	}
}

func TestReleaseViewShowsNoneForEmptyAssets(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return releaseResponse(http.StatusOK, `{
					"tag_name":"v0.1.0","name":"Seed","target_commitish":"main",
					"prerelease":false,"release_status":"latest",
					"created_at":"2026-01-01T00:00:00Z",
					"author":{"login":"alice","name":"Alice"},
					"body":"seed","assets":[]
				}`), nil
			})}, nil
		},
	}
	cmd := newCmdReleaseView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"atomclub/ag", "v0.1.0"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Assets:") {
		t.Fatalf("output missing Assets header: %s", got)
	}
	if !strings.Contains(got, "  None") {
		t.Fatalf("output missing None marker: %s", got)
	}
}

func TestReleaseViewReturnsErrorOn404(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return releaseResponse(http.StatusNotFound, "not found"), nil
			})}, nil
		},
	}
	cmd := newCmdReleaseView(factory)
	err := cmd.RunE(cmd, []string{"atomclub/ag", "missing-tag"})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "failed to view release") {
		t.Fatalf("error %q does not contain operation context", err.Error())
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error %q does not contain status 404", err.Error())
	}
}
