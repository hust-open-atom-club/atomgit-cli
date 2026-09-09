package release

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type createTransport struct {
	gotMethod  string
	gotPath    string
	gotBody    map[string]any
	statusCode int
	respBody   string
	called     int
}

func (t *createTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.called++
	t.gotMethod = req.Method
	t.gotPath = req.URL.Path
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		t.gotBody = make(map[string]any)
		if err := json.Unmarshal(b, &t.gotBody); err != nil {
			return nil, err
		}
	}
	return releaseResponse(t.statusCode, t.respBody), nil
}

func newCreateFactory(transport http.RoundTripper) *cmdutil.Factory {
	return releaseTestFactory(transport)
}

func TestReleaseCreateRegistersCommandAndFlags(t *testing.T) {
	cmd := NewCmdRelease(&cmdutil.Factory{})

	create, _, err := cmd.Find([]string{"create"})
	if err != nil || create == nil {
		t.Fatal("release create not found")
	}
	for _, flag := range []string{"name", "body", "body-file", "target", "prerelease"} {
		if create.Flags().Lookup(flag) == nil {
			t.Fatalf("--%s flag was not registered", flag)
		}
	}
	if sh := create.Flags().Lookup("name").Shorthand; sh != "n" {
		t.Fatalf("--name shorthand = %q, want n", sh)
	}
	if sh := create.Flags().Lookup("body").Shorthand; sh != "b" {
		t.Fatalf("--body shorthand = %q, want b", sh)
	}
	if sh := create.Flags().Lookup("body-file").Shorthand; sh != "F" {
		t.Fatalf("--body-file shorthand = %q, want F", sh)
	}
	for _, forbidden := range []string{"draft", "latest", "yes", "no-prerelease"} {
		if create.Flags().Lookup(forbidden) != nil {
			t.Fatalf("--%s flag should not be registered", forbidden)
		}
	}
	if !strings.Contains(create.Long, cmdutil.RepositoryContextHelp) {
		t.Errorf("create Long missing repository context help: %q", create.Long)
	}
	if err := create.Args(create, []string{"v1.0.0"}); err != nil {
		t.Fatalf("Args(1) = %v, want nil", err)
	}
	if err := create.Args(create, []string{"alice/demo", "v1.0.0"}); err != nil {
		t.Fatalf("Args(2) = %v, want nil", err)
	}
	if err := create.Args(create, nil); err == nil {
		t.Fatal("Args(0) = nil, want error")
	}
	if err := create.Args(create, []string{"a", "b", "c"}); err == nil {
		t.Fatal("Args(3) = nil, want error")
	}
}

func TestReleaseCreateMinimalRequestOmitsTargetAndStatus(t *testing.T) {
	transport := &createTransport{statusCode: http.StatusCreated, respBody: `{"tag_name":"v1.0.0"}`}
	cmd := newCmdReleaseCreate(newCreateFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Flags().Set("body", "release notes"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if transport.gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", transport.gotMethod)
	}
	if transport.gotPath != "/api/v5/repos/alice/demo/releases" {
		t.Fatalf("path = %q", transport.gotPath)
	}
	if transport.gotBody["tag_name"] != "v1.0.0" {
		t.Fatalf("tag_name = %v", transport.gotBody["tag_name"])
	}
	if transport.gotBody["name"] != "v1.0.0" {
		t.Fatalf("name = %v (should default to tag)", transport.gotBody["name"])
	}
	if transport.gotBody["body"] != "release notes" {
		t.Fatalf("body = %v, want release notes", transport.gotBody["body"])
	}
	if _, present := transport.gotBody["target_commitish"]; present {
		t.Fatalf("target_commitish should be omitted; got %v", transport.gotBody["target_commitish"])
	}
	if _, present := transport.gotBody["release_status"]; present {
		t.Fatalf("release_status should be omitted; got %v", transport.gotBody["release_status"])
	}
	if got, want := out.String(), "Created release v1.0.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseCreateCompleteRequestWithAllFlags(t *testing.T) {
	transport := &createTransport{statusCode: http.StatusCreated, respBody: `{"tag_name":"v2.0.0"}`}
	cmd := newCmdReleaseCreate(newCreateFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)

	args := []string{"alice/demo", "v2.0.0",
		"--name", "Second",
		"--body", "release notes v2",
		"--target", "  develop  ",
		"--prerelease",
	}
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.gotBody["tag_name"] != "v2.0.0" {
		t.Fatalf("tag_name = %v", transport.gotBody["tag_name"])
	}
	if transport.gotBody["name"] != "Second" {
		t.Fatalf("name = %v", transport.gotBody["name"])
	}
	if transport.gotBody["body"] != "release notes v2" {
		t.Fatalf("body = %v", transport.gotBody["body"])
	}
	if transport.gotBody["target_commitish"] != "develop" {
		t.Fatalf("target_commitish = %v", transport.gotBody["target_commitish"])
	}
	if transport.gotBody["release_status"] != api.ReleaseStatusPre {
		t.Fatalf("release_status = %v, want %q", transport.gotBody["release_status"], api.ReleaseStatusPre)
	}
	if got, want := out.String(), "Created release v2.0.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseCreateBodyFileReadsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	notes := "body from file\nsecond line"
	if err := os.WriteFile(path, []byte(notes), 0600); err != nil {
		t.Fatal(err)
	}
	transport := &createTransport{statusCode: http.StatusCreated, respBody: `{"tag_name":"v3.0.0"}`}
	cmd := newCmdReleaseCreate(newCreateFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v3.0.0", "--body-file", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.gotBody["body"] != notes {
		t.Fatalf("body = %q, want file contents %q", transport.gotBody["body"], notes)
	}
}

func TestReleaseCreateRejectsMutexAndEmptyInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(emptyPath, []byte("\n\t"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "body option required",
			args:      []string{"alice/demo", "v1.0.0"},
			wantError: "release body is required",
		},
		{
			name:      "explicit empty body rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body", "   "},
			wantError: "body must not be empty",
		},
		{
			name:      "empty body file rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body-file", emptyPath},
			wantError: "body must not be empty",
		},
		{
			name:      "body and body-file mutually exclusive",
			args:      []string{"alice/demo", "v1.0.0", "--body", "inline", "--body-file", path},
			wantError: "mutually exclusive",
		},
		{
			name:      "empty tag rejected",
			args:      []string{"alice/demo", "   "},
			wantError: "release tag is required",
		},
		{
			name:      "explicit empty name rejected",
			args:      []string{"alice/demo", "v1.0.0", "--name", ""},
			wantError: "must not be empty",
		},
		{
			name:      "explicit empty target rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body", "release notes", "--target", "   "},
			wantError: "target",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &createTransport{statusCode: http.StatusCreated, respBody: `{}`}
			cmd := newCmdReleaseCreate(newCreateFactory(transport))
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tc.wantError)
			}
			if transport.called != 0 {
				t.Fatalf("transport called %d times, want 0 (validation before request)", transport.called)
			}
		})
	}
}

func TestReleaseCreateReturnsErrorOn409WithStatus(t *testing.T) {
	transport := &createTransport{statusCode: http.StatusConflict, respBody: "tag already exists"}
	cmd := newCmdReleaseCreate(newCreateFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1.0.0", "--body", "release notes"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 409, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create release") {
		t.Fatalf("error %q does not contain operation context", err.Error())
	}
	if !strings.Contains(err.Error(), "409") {
		t.Fatalf("error %q does not contain status 409", err.Error())
	}
}

func TestReleaseCreateInfersRepoFromResolver(t *testing.T) {
	transport := &createTransport{statusCode: http.StatusCreated, respBody: `{"tag_name":"v1.1.0"}`}
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "team", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
	cmd := newCmdReleaseCreate(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1.1.0", "--body", "release notes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", transport.gotMethod)
	}
	if transport.gotPath != "/api/v5/repos/team/demo/releases" {
		t.Fatalf("path = %q, want /api/v5/repos/team/demo/releases", transport.gotPath)
	}
	if transport.gotBody["tag_name"] != "v1.1.0" {
		t.Fatalf("tag_name = %v", transport.gotBody["tag_name"])
	}
	if got, want := out.String(), "Created release v1.1.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
