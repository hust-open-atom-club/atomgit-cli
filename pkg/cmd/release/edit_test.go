package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type editTransport struct {
	getCalls   int
	patchCalls int

	gotMethod string
	getPath   string
	patchPath string
	gotBody   map[string]any

	getStatusCode int
	getRespBody   string
	getErr        error

	patchStatusCode int
	patchRespBody   string
	patchErr        error
}

func (t *editTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.gotMethod = req.Method
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
	switch req.Method {
	case http.MethodGet:
		t.getCalls++
		t.getPath = req.URL.EscapedPath()
		if t.getErr != nil {
			return nil, t.getErr
		}
		return &http.Response{
			StatusCode: t.getStatusCode,
			Status:     fmt.Sprintf("%d %s", t.getStatusCode, http.StatusText(t.getStatusCode)),
			Body:       io.NopCloser(strings.NewReader(t.getRespBody)),
			Header:     make(http.Header),
		}, nil
	case http.MethodPatch:
		t.patchCalls++
		t.patchPath = req.URL.EscapedPath()
		if t.patchErr != nil {
			return nil, t.patchErr
		}
		return &http.Response{
			StatusCode: t.patchStatusCode,
			Status:     fmt.Sprintf("%d %s", t.patchStatusCode, http.StatusText(t.patchStatusCode)),
			Body:       io.NopCloser(strings.NewReader(t.patchRespBody)),
			Header:     make(http.Header),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected method %q", req.Method)
	}
}

func newEditFactory(transport *editTransport) *cmdutil.Factory {
	return releaseTestFactory(transport)
}

func TestReleaseEditRegistersCommandFlagsAndArgs(t *testing.T) {
	cmd := NewCmdRelease(&cmdutil.Factory{})
	edit, _, err := cmd.Find([]string{"edit"})
	if err != nil || edit == nil {
		t.Fatal("release edit not found")
	}
	for _, c := range []struct{ name, sh string }{
		{"name", "n"}, {"body", "b"}, {"body-file", "F"},
	} {
		f := edit.Flags().Lookup(c.name)
		if f == nil {
			t.Fatalf("--%s flag was not registered", c.name)
		}
		if f.Shorthand != c.sh {
			t.Fatalf("--%s shorthand = %q, want %q", c.name, f.Shorthand, c.sh)
		}
	}
	for _, name := range []string{"prerelease", "latest"} {
		if edit.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag was not registered", name)
		}
	}
	for _, forbidden := range []string{"target", "draft", "yes", "no-prerelease"} {
		if edit.Flags().Lookup(forbidden) != nil {
			t.Fatalf("--%s flag should not be registered", forbidden)
		}
	}
	if !strings.Contains(edit.Long, cmdutil.RepositoryContextHelp) {
		t.Errorf("edit Long missing repository context help: %q", edit.Long)
	}
	if err := edit.Args(edit, []string{"v1.0.0"}); err != nil {
		t.Fatalf("Args(1) = %v, want nil", err)
	}
	if err := edit.Args(edit, []string{"alice/demo", "v1.0.0"}); err != nil {
		t.Fatalf("Args(2) = %v, want nil", err)
	}
	if err := edit.Args(edit, nil); err == nil {
		t.Fatal("Args(0) = nil, want error")
	}
	if err := edit.Args(edit, []string{"a", "b", "c"}); err == nil {
		t.Fatal("Args(3) = nil, want error")
	}
}

func TestReleaseEditValidatesInputsBeforeAnyRequest(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(bodyFile, []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	emptyBodyFile := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(emptyBodyFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	whitespaceBodyFile := filepath.Join(dir, "whitespace.md")
	if err := os.WriteFile(whitespaceBodyFile, []byte("  \n\t"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "no change flag",
			args:      []string{"alice/demo", "v1.0.0"},
			wantError: "at least one of",
		},
		{
			name:      "body and body-file mutually exclusive",
			args:      []string{"alice/demo", "v1.0.0", "--body", "inline", "--body-file", bodyFile},
			wantError: "mutually exclusive",
		},
		{
			name:      "prerelease and latest mutually exclusive",
			args:      []string{"alice/demo", "v1.0.0", "--prerelease", "--latest"},
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
			name:      "explicit empty body rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body", ""},
			wantError: "release body must not be empty",
		},
		{
			name:      "explicit whitespace body rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body", "   "},
			wantError: "release body must not be empty",
		},
		{
			name:      "empty body file rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body-file", emptyBodyFile},
			wantError: "release body must not be empty",
		},
		{
			name:      "whitespace body file rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body-file", whitespaceBodyFile},
			wantError: "release body must not be empty",
		},
		{
			name:      "prerelease=false rejected",
			args:      []string{"alice/demo", "v1.0.0", "--prerelease=false"},
			wantError: "--prerelease=false is not supported",
		},
		{
			name:      "latest=false rejected",
			args:      []string{"alice/demo", "v1.0.0", "--latest=false"},
			wantError: "--latest=false is not supported",
		},
		{
			name:      "non-existent body file rejected",
			args:      []string{"alice/demo", "v1.0.0", "--body-file", filepath.Join(dir, "missing.md")},
			wantError: "failed to read body file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &editTransport{}
			cmd := newCmdReleaseEdit(newEditFactory(transport))
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
			if transport.getCalls != 0 || transport.patchCalls != 0 {
				t.Fatalf("transport calls = GET %d / PATCH %d, want 0/0", transport.getCalls, transport.patchCalls)
			}
		})
	}
}

func TestReleaseEditNameOnlyMergesCurrentBody(t *testing.T) {
	transport := &editTransport{
		getStatusCode:   http.StatusOK,
		getRespBody:     `{"tag_name":"v1.0.0","name":"Old","body":"old body","release_status":"pre"}`,
		patchStatusCode: http.StatusOK,
		patchRespBody:   `{"tag_name":"v1.0.0","name":"New","body":"old body","release_status":"pre"}`,
	}
	cmd := newCmdReleaseEdit(newEditFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1.0.0", "--name", "New"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.getCalls != 1 || transport.patchCalls != 1 {
		t.Fatalf("transport calls = GET %d / PATCH %d, want 1/1", transport.getCalls, transport.patchCalls)
	}
	if transport.gotMethod != http.MethodPatch {
		t.Fatalf("recorded method = %q, want PATCH", transport.gotMethod)
	}
	const wantPatchPath = "/api/v5/repos/alice/demo/releases/v1.0.0"
	if transport.patchPath != wantPatchPath {
		t.Fatalf("PATCH path = %q, want %q", transport.patchPath, wantPatchPath)
	}
	if transport.gotBody["name"] != "New" {
		t.Fatalf("PATCH name = %v, want %q", transport.gotBody["name"], "New")
	}
	if transport.gotBody["body"] != "old body" {
		t.Fatalf("PATCH body = %v, want %q", transport.gotBody["body"], "old body")
	}
	if _, present := transport.gotBody["release_status"]; present {
		t.Fatalf("PATCH release_status should be omitted; got %v", transport.gotBody["release_status"])
	}
	if got, want := out.String(), "Updated release v1.0.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseEditPrereleaseSetsPreStatus(t *testing.T) {
	transport := &editTransport{
		getStatusCode:   http.StatusOK,
		getRespBody:     `{"tag_name":"v1.0.0","name":"Old","body":"old body","release_status":"latest"}`,
		patchStatusCode: http.StatusOK,
		patchRespBody:   `{"tag_name":"v1.0.0","name":"Old","body":"old body","release_status":"pre"}`,
	}
	cmd := newCmdReleaseEdit(newEditFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1.0.0", "--prerelease"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.getCalls != 1 || transport.patchCalls != 1 {
		t.Fatalf("transport calls = GET %d / PATCH %d, want 1/1", transport.getCalls, transport.patchCalls)
	}
	if transport.gotMethod != http.MethodPatch {
		t.Fatalf("recorded method = %q, want PATCH", transport.gotMethod)
	}
	const wantPatchPath = "/api/v5/repos/alice/demo/releases/v1.0.0"
	if transport.patchPath != wantPatchPath {
		t.Fatalf("PATCH path = %q, want %q", transport.patchPath, wantPatchPath)
	}
	if transport.gotBody["name"] != "Old" {
		t.Fatalf("PATCH name = %v, want %q", transport.gotBody["name"], "Old")
	}
	if transport.gotBody["body"] != "old body" {
		t.Fatalf("PATCH body = %v, want %q", transport.gotBody["body"], "old body")
	}
	if transport.gotBody["release_status"] != api.ReleaseStatusPre {
		t.Fatalf("PATCH release_status = %v, want %q", transport.gotBody["release_status"], api.ReleaseStatusPre)
	}
	if got, want := out.String(), "Updated release v1.0.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseEditBodyFileAndLatest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	notes := "new body from file"
	if err := os.WriteFile(path, []byte(notes), 0600); err != nil {
		t.Fatal(err)
	}
	transport := &editTransport{
		getStatusCode:   http.StatusOK,
		getRespBody:     `{"tag_name":"v1.0.0","name":"Old","body":"old body","release_status":"pre"}`,
		patchStatusCode: http.StatusOK,
		patchRespBody:   `{"tag_name":"v1.0.0","name":"Old","body":"new body from file","release_status":"latest"}`,
	}
	cmd := newCmdReleaseEdit(newEditFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1.0.0", "--body-file", path, "--latest"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.getCalls != 1 || transport.patchCalls != 1 {
		t.Fatalf("transport calls = GET %d / PATCH %d, want 1/1", transport.getCalls, transport.patchCalls)
	}
	if transport.gotMethod != http.MethodPatch {
		t.Fatalf("recorded method = %q, want PATCH", transport.gotMethod)
	}
	const wantPatchPath = "/api/v5/repos/alice/demo/releases/v1.0.0"
	if transport.patchPath != wantPatchPath {
		t.Fatalf("PATCH path = %q, want %q", transport.patchPath, wantPatchPath)
	}
	if transport.gotBody["name"] != "Old" {
		t.Fatalf("PATCH name = %v, want %q", transport.gotBody["name"], "Old")
	}
	if transport.gotBody["body"] != notes {
		t.Fatalf("PATCH body = %v, want %q", transport.gotBody["body"], notes)
	}
	if transport.gotBody["release_status"] != api.ReleaseStatusLatest {
		t.Fatalf("PATCH release_status = %v, want %q", transport.gotBody["release_status"], api.ReleaseStatusLatest)
	}
	if got, want := out.String(), "Updated release v1.0.0\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseEditInfersRepoAndEscapesSlashTag(t *testing.T) {
	transport := &editTransport{
		getStatusCode:   http.StatusOK,
		getRespBody:     `{"tag_name":"v1/rc","name":"Old","body":"old body","release_status":"pre"}`,
		patchStatusCode: http.StatusOK,
		patchRespBody:   `{"tag_name":"v1/rc","name":"New","body":"old body","release_status":"pre"}`,
	}
	factory := &cmdutil.Factory{
		Config: releaseTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "team", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
	cmd := newCmdReleaseEdit(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1/rc", "--name", "New"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.getCalls != 1 || transport.patchCalls != 1 {
		t.Fatalf("transport calls = GET %d / PATCH %d, want 1/1", transport.getCalls, transport.patchCalls)
	}
	const wantGetPath = "/api/v5/repos/team/demo/releases/tags/v1%2Frc"
	if transport.getPath != wantGetPath {
		t.Fatalf("GET path = %q, want %q", transport.getPath, wantGetPath)
	}
	const wantPatchPath = "/api/v5/repos/team/demo/releases/v1%2Frc"
	if transport.patchPath != wantPatchPath {
		t.Fatalf("PATCH path = %q, want %q", transport.patchPath, wantPatchPath)
	}
	if transport.gotMethod != http.MethodPatch {
		t.Fatalf("recorded method = %q, want PATCH", transport.gotMethod)
	}
	if transport.gotBody["name"] != "New" {
		t.Fatalf("PATCH name = %v, want %q", transport.gotBody["name"], "New")
	}
	if got, want := out.String(), "Updated release v1/rc\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestReleaseEditGET404ReturnsErrorWithStatus(t *testing.T) {
	transport := &editTransport{
		getStatusCode: http.StatusNotFound,
		getRespBody:   "not found",
	}
	cmd := newCmdReleaseEdit(newEditFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "missing-tag", "--name", "New"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for GET 404, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get release before editing") {
		t.Fatalf("error %q does not contain GET context", err.Error())
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error %q does not contain status 404", err.Error())
	}
	if transport.patchCalls != 0 {
		t.Fatalf("PATCH calls = %d, want 0 after GET failure", transport.patchCalls)
	}
}

func TestReleaseEditPATCHErrorsWrapWithContextAndStatus(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
	}{
		{name: "403 forbidden", statusCode: http.StatusForbidden},
		{name: "429 too many requests", statusCode: http.StatusTooManyRequests},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &editTransport{
				getStatusCode:   http.StatusOK,
				getRespBody:     `{"tag_name":"v1.0.0","name":"Old","body":"old body","release_status":"pre"}`,
				patchStatusCode: tc.statusCode,
				patchRespBody:   "patch failed",
			}
			cmd := newCmdReleaseEdit(newEditFactory(transport))
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"alice/demo", "v1.0.0", "--name", "New"})
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error for PATCH failure, got nil")
			}
			if !strings.Contains(err.Error(), "failed to edit release") {
				t.Fatalf("error %q does not contain PATCH context", err.Error())
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", tc.statusCode)) {
				t.Fatalf("error %q does not contain status %d", err.Error(), tc.statusCode)
			}
		})
	}
}
