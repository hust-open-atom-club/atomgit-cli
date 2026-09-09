package release

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const uploadHost = "store.example.com"

type uploadAsset struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type uploadTransport struct {
	calls      []string
	totalCalls int

	apiCalls []uploadAPIRequest

	putHeaders       http.Header
	putBody          []byte
	putCalls         int
	putStatusCode    int
	putContentLength int64

	apiHandler func(method, path, rawQuery string, body []byte) (*http.Response, error)
}

type uploadAPIRequest struct {
	Method   string
	Path     string
	RawQuery string
}

func (t *uploadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.totalCalls++
	host := req.URL.Host
	t.calls = append(t.calls, req.Method+" "+host+req.URL.RequestURI())

	switch host {
	case "api.atomgit.com":
		var body []byte
		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			_ = req.Body.Close()
			if err != nil {
				return nil, err
			}
			body = b
		}
		t.apiCalls = append(t.apiCalls, uploadAPIRequest{
			Method:   req.Method,
			Path:     req.URL.EscapedPath(),
			RawQuery: req.URL.RawQuery,
		})
		handler := t.apiHandler
		if handler == nil {
			handler = uploadNotFoundHandler
		}
		return handler(req.Method, req.URL.EscapedPath(), req.URL.RawQuery, body)
	case uploadHost:
		t.putCalls++
		var body []byte
		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			_ = req.Body.Close()
			if err != nil {
				return nil, err
			}
			body = b
		}
		t.putBody = body
		t.putHeaders = req.Header.Clone()
		t.putContentLength = req.ContentLength
		statusCode := t.putStatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		return &http.Response{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected host %q", host)
	}
}

func uploadNotFoundHandler(method, path, rawQuery string, body []byte) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       io.NopCloser(strings.NewReader("not found")),
		Header:     make(http.Header),
	}, nil
}

func uploadJSONResponse(statusCode int, body string) *http.Response {
	return releaseResponse(statusCode, body)
}

func newUploadFactory(transport *uploadTransport) *cmdutil.Factory {
	return releaseTestFactory(transport)
}

func releaseWithAssetsJSON(tag string, assets []uploadAsset) string {
	var b strings.Builder
	b.WriteString(`{"tag_name":"`)
	b.WriteString(tag)
	b.WriteString(`","name":"Release","body":"body","assets":[`)
	for i, a := range assets {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":%d,"name":%q,"type":%q}`, a.ID, a.Name, a.Type)
	}
	b.WriteString(`]}`)
	return b.String()
}

func TestReleaseUploadRegistersCommandFlagsAndArgs(t *testing.T) {
	root := NewCmdRelease(&cmdutil.Factory{})
	upload, _, err := root.Find([]string{"upload"})
	if err != nil || upload == nil {
		t.Fatal("release upload not found")
	}

	if got, want := upload.Use, "upload [<owner>/<repo>] <tag> <file>"; got != want {
		t.Fatalf("Use = %q, want %q", got, want)
	}
	if err := upload.Args(upload, []string{"v1.0.0", "file.tar.gz"}); err != nil {
		t.Fatalf("Args(2) = %v, want nil", err)
	}
	if err := upload.Args(upload, []string{"alice/demo", "v1.0.0", "file.tar.gz"}); err != nil {
		t.Fatalf("Args(3) = %v, want nil", err)
	}
	if err := upload.Args(upload, []string{"v1.0.0"}); err == nil {
		t.Fatal("Args(1) = nil, want error")
	}
	if err := upload.Args(upload, []string{"a", "b", "c", "d"}); err == nil {
		t.Fatal("Args(4) = nil, want error")
	}

	for _, c := range []struct{ name, sh string }{
		{"name", "n"},
	} {
		f := upload.Flags().Lookup(c.name)
		if f == nil {
			t.Fatalf("--%s flag was not registered", c.name)
		}
		if f.Shorthand != c.sh {
			t.Fatalf("--%s shorthand = %q, want %q", c.name, f.Shorthand, c.sh)
		}
	}
	for _, name := range []string{"skip-existing", "overwrite", "timeout"} {
		if upload.Flags().Lookup(name) == nil {
			t.Fatalf("--%s flag was not registered", name)
		}
	}
	if got := upload.Flags().Lookup("timeout").DefValue; got != defaultReleaseTransferTimeout.String() {
		t.Fatalf("--timeout default = %q, want %q", got, defaultReleaseTransferTimeout.String())
	}
	for _, forbidden := range []string{"yes", "draft", "target"} {
		if upload.Flags().Lookup(forbidden) != nil {
			t.Fatalf("--%s flag should not be registered", forbidden)
		}
	}
	if !strings.Contains(upload.Long, cmdutil.RepositoryContextHelp) {
		t.Errorf("upload Long missing repository context help: %q", upload.Long)
	}
}

func TestReleaseUploadValidatesInputsBeforeAnyRequest(t *testing.T) {
	dir := t.TempDir()
	dirPath := filepath.Join(dir, "subdir")
	if err := os.Mkdir(dirPath, 0o700); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(regularFile, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "empty tag",
			args:      []string{"alice/demo", "   ", regularFile},
			wantError: "release tag is required",
		},
		{
			name:      "explicit empty name",
			args:      []string{"alice/demo", "v1.0.0", regularFile, "--name", "   "},
			wantError: "must not be empty",
		},
		{
			name:      "default whitespace name",
			args:      []string{"alice/demo", "v1.0.0", dirPath, "--name", "   "},
			wantError: "must not be empty",
		},
		{
			name:      "negative timeout",
			args:      []string{"alice/demo", "v1.0.0", regularFile, "--timeout", "-1s"},
			wantError: "must not be negative",
		},
		{
			name:      "non-existent file",
			args:      []string{"alice/demo", "v1.0.0", filepath.Join(dir, "missing.tar.gz")},
			wantError: "failed to stat upload file",
		},
		{
			name:      "directory instead of file",
			args:      []string{"alice/demo", "v1.0.0", dirPath},
			wantError: "not a regular file",
		},
		{
			name:      "skip and overwrite mutually exclusive",
			args:      []string{"alice/demo", "v1.0.0", regularFile, "--skip-existing", "--overwrite"},
			wantError: "mutually exclusive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &uploadTransport{}
			cmd := newCmdReleaseUpload(newUploadFactory(transport))
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
			if transport.totalCalls != 0 {
				t.Fatalf("transport calls = %d, want 0; calls=%v", transport.totalCalls, transport.calls)
			}
		})
	}
}

func setupUploadSuccess(t *testing.T, tag string, putStatusCode int) *uploadTransport {
	t.Helper()
	transport := &uploadTransport{
		putStatusCode: putStatusCode,
		apiHandler: func(method, path, rawQuery string, body []byte) (*http.Response, error) {
			switch {
			case method == http.MethodGet && strings.HasPrefix(path, "/api/v5/repos/alice/demo/releases/tags/"):
				return uploadJSONResponse(http.StatusOK, releaseWithAssetsJSON(tag, nil)), nil
			case method == http.MethodGet && strings.HasPrefix(path, "/api/v5/repos/alice/demo/releases/"):
				return uploadJSONResponse(http.StatusOK, `{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream","X-MD5":"deadbeef"}}`), nil
			default:
				return uploadNotFoundHandler(method, path, rawQuery, body)
			}
		},
	}
	return transport
}

func TestReleaseUploadSuccessChain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	payload := []byte("release-binary-payload")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	transport := setupUploadSuccess(t, "v1/rc", http.StatusOK)
	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if transport.totalCalls != 3 {
		t.Fatalf("total calls = %d, want 3; calls=%v", transport.totalCalls, transport.calls)
	}
	if transport.putCalls != 1 {
		t.Fatalf("external PUT calls = %d, want 1", transport.putCalls)
	}

	if len(transport.apiCalls) != 2 {
		t.Fatalf("api calls = %d, want 2", len(transport.apiCalls))
	}
	getRelease := transport.apiCalls[0]
	if getRelease.Method != http.MethodGet {
		t.Fatalf("GET release method = %q, want GET", getRelease.Method)
	}
	const wantGetPath = "/api/v5/repos/alice/demo/releases/tags/v1%2Frc"
	if getRelease.Path != wantGetPath {
		t.Fatalf("GET release path = %q, want %q", getRelease.Path, wantGetPath)
	}

	getUploadURL := transport.apiCalls[1]
	if getUploadURL.Method != http.MethodGet {
		t.Fatalf("GET upload_url method = %q, want GET", getUploadURL.Method)
	}
	const wantUploadPath = "/api/v5/repos/alice/demo/releases/v1%2Frc/upload_url"
	if getUploadURL.Path != wantUploadPath {
		t.Fatalf("GET upload_url path = %q, want %q", getUploadURL.Path, wantUploadPath)
	}
	wantName := filepath.Base(path)
	parsedQuery, err := url.ParseQuery(getUploadURL.RawQuery)
	if err != nil {
		t.Fatalf("parse upload_url query: %v", err)
	}
	if got := parsedQuery.Get("file_name"); got != wantName {
		t.Fatalf("upload_url file_name = %q, want %q", got, wantName)
	}

	if !bytes.Equal(transport.putBody, payload) {
		t.Fatalf("external PUT body = %q, want %q", transport.putBody, payload)
	}
	if transport.putContentLength != int64(len(payload)) {
		t.Fatalf("external PUT Content-Length = %d, want %d", transport.putContentLength, len(payload))
	}
	if got := transport.putHeaders.Get("Authorization"); got != "" {
		t.Fatalf("external PUT leaked Authorization: %q", got)
	}

	wantOut := "Uploaded attachment " + wantName + " to release v1/rc\n"
	if got := out.String(); got != wantOut {
		t.Fatalf("output = %q, want %q", got, wantOut)
	}
}

func TestReleaseUploadCustomNameInQueryAndOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	transport := setupUploadSuccess(t, "v1/rc", http.StatusOK)
	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1/rc", path, "--name", "custom.bin"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(transport.apiCalls) != 2 {
		t.Fatalf("api calls = %d, want 2", len(transport.apiCalls))
	}
	const wantGetPath = "/api/v5/repos/alice/demo/releases/tags/v1%2Frc"
	if got := transport.apiCalls[0].Path; got != wantGetPath {
		t.Fatalf("GET release path = %q, want %q", got, wantGetPath)
	}
	getUploadURL := transport.apiCalls[1]
	parsedQuery, err := url.ParseQuery(getUploadURL.RawQuery)
	if err != nil {
		t.Fatalf("parse upload_url query: %v", err)
	}
	if got := parsedQuery.Get("file_name"); got != "custom.bin" {
		t.Fatalf("upload_url file_name = %q, want %q", got, "custom.bin")
	}
	if got, want := out.String(), "Uploaded attachment custom.bin to release v1/rc\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func setupExistingAttachTransport(name string) *uploadTransport {
	return &uploadTransport{
		apiHandler: func(method, path, rawQuery string, body []byte) (*http.Response, error) {
			if method == http.MethodGet && strings.HasPrefix(path, "/api/v5/repos/alice/demo/releases/tags/") {
				return uploadJSONResponse(
					http.StatusOK,
					releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: name, Type: "attach"}}),
				), nil
			}
			return uploadNotFoundHandler(method, path, rawQuery, body)
		},
	}
}

func TestReleaseUploadDefaultNameCollidesErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	transport := setupExistingAttachTransport(baseName)
	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"already exists", "--skip-existing", "--overwrite"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
	if transport.totalCalls != 1 {
		t.Fatalf("total calls = %d, want 1; calls=%v", transport.totalCalls, transport.calls)
	}
	if len(transport.apiCalls) != 1 || transport.apiCalls[0].Method != http.MethodGet {
		t.Fatalf("api calls = %+v, want exactly one GET release", transport.apiCalls)
	}
	if transport.putCalls != 0 {
		t.Fatalf("external PUT calls = %d, want 0", transport.putCalls)
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty", out.String())
	}
}

func TestReleaseUploadSkipExistingReportsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	transport := setupExistingAttachTransport(baseName)
	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path, "--skip-existing"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if transport.totalCalls != 1 {
		t.Fatalf("total calls = %d, want 1; calls=%v", transport.totalCalls, transport.calls)
	}
	if len(transport.apiCalls) != 1 || transport.apiCalls[0].Method != http.MethodGet {
		t.Fatalf("api calls = %+v, want exactly one GET release", transport.apiCalls)
	}
	if transport.putCalls != 0 {
		t.Fatalf("external PUT calls = %d, want 0", transport.putCalls)
	}
	wantOut := "Skipped existing attachment " + baseName + " on release v1/rc\n"
	if got := out.String(); got != wantOut {
		t.Fatalf("output = %q, want %q", got, wantOut)
	}
}

func TestReleaseUploadOverwriteUniqueAttach(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	payload := []byte("overwrite-payload")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	transport := &uploadTransport{
		putStatusCode: http.StatusOK,
		apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
			switch {
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
				return uploadJSONResponse(
					http.StatusOK,
					releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}),
				), nil
			case method == http.MethodDelete && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
				return uploadJSONResponse(http.StatusNoContent, ""), nil
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
				return uploadJSONResponse(
					http.StatusOK,
					`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream","X-MD5":"deadbeef"}}`,
				), nil
			default:
				return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
			}
		},
	}

	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path, "--overwrite"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if transport.totalCalls != 4 {
		t.Fatalf("total calls = %d, want 4; calls=%v", transport.totalCalls, transport.calls)
	}
	if transport.putCalls != 1 {
		t.Fatalf("external PUT calls = %d, want 1", transport.putCalls)
	}
	if !bytes.Equal(transport.putBody, payload) {
		t.Fatalf("external PUT body = %q, want %q", transport.putBody, payload)
	}

	if len(transport.apiCalls) != 3 {
		t.Fatalf("api calls = %d, want 3", len(transport.apiCalls))
	}
	wantAPI := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v5/repos/alice/demo/releases/tags/v1%2Frc"},
		{http.MethodGet, "/api/v5/repos/alice/demo/releases/v1%2Frc/upload_url"},
		{http.MethodDelete, "/api/v5/repos/alice/demo/releases/v1%2Frc/attach_files/42"},
	}
	for i, want := range wantAPI {
		got := transport.apiCalls[i]
		if got.Method != want.method {
			t.Fatalf("api call %d method = %q, want %q", i, got.Method, want.method)
		}
		if got.Path != want.path {
			t.Fatalf("api call %d path = %q, want %q", i, got.Path, want.path)
		}
	}

	wantOut := "Uploaded attachment " + baseName + " to release v1/rc\n"
	if got := out.String(); got != wantOut {
		t.Fatalf("output = %q, want %q", got, wantOut)
	}
}

func TestReleaseUploadOverwriteReconcilesLostDeleteResponse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	var releaseGets int
	transport := &uploadTransport{
		putStatusCode: http.StatusOK,
		apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
			switch {
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
				releaseGets++
				assets := []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}
				if releaseGets > 1 {
					assets = nil
				}
				return uploadJSONResponse(http.StatusOK, releaseWithAssetsJSON("v1/rc", assets)), nil
			case method == http.MethodGet && strings.HasSuffix(escapedPath, "/upload_url"):
				return uploadJSONResponse(
					http.StatusOK,
					`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream"}}`,
				), nil
			case method == http.MethodDelete:
				return nil, errors.New("delete response lost")
			default:
				return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
			}
		},
	}

	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	cmd.SilenceUsage = true
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path, "--overwrite"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if transport.putCalls != 1 {
		t.Fatalf("external PUT calls = %d, want 1", transport.putCalls)
	}
	deleteCalls := 0
	for _, call := range transport.apiCalls {
		if call.Method == http.MethodDelete {
			deleteCalls++
		}
	}
	if deleteCalls != 1 {
		t.Fatalf("DELETE calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(errOut.String(), "attachment "+baseName+" is absent") {
		t.Fatalf("stderr = %q, want reconciliation message", errOut.String())
	}
}

func TestReleaseUploadOverwriteStopsWhenDeleteStateIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	transport := &uploadTransport{
		apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
			switch {
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
				return uploadJSONResponse(
					http.StatusOK,
					releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}),
				), nil
			case method == http.MethodGet && strings.HasSuffix(escapedPath, "/upload_url"):
				return uploadJSONResponse(
					http.StatusOK,
					`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream"}}`,
				), nil
			case method == http.MethodDelete:
				return nil, errors.New("delete response lost")
			default:
				return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
			}
		},
	}

	cmd := newCmdReleaseUpload(newUploadFactory(transport))
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"alice/demo", "v1/rc", path, "--overwrite"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "delete response lost") {
		t.Fatalf("error = %v, want delete transport context", err)
	}
	if transport.putCalls != 0 {
		t.Fatalf("external PUT calls = %d, want 0", transport.putCalls)
	}
}

func TestReleaseUploadOverwriteRejectsUnsafeMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	cases := []struct {
		name      string
		assets    []uploadAsset
		wantError []string
	}{
		{
			name:      "source asset rejected",
			assets:    []uploadAsset{{ID: 0, Name: baseName, Type: "source"}},
			wantError: []string{"type", "cannot overwrite"},
		},
		{
			name:      "attach with id 0 rejected",
			assets:    []uploadAsset{{ID: 0, Name: baseName, Type: "attach"}},
			wantError: []string{"invalid id"},
		},
		{
			name: "two same-name attach assets rejected",
			assets: []uploadAsset{
				{ID: 1, Name: baseName, Type: "attach"},
				{ID: 2, Name: baseName, Type: "attach"},
			},
			wantError: []string{"found 2", "exactly one"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assets := tc.assets
			transport := &uploadTransport{
				apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
					if method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/") {
						return uploadJSONResponse(http.StatusOK, releaseWithAssetsJSON("v1/rc", assets)), nil
					}
					return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
				},
			}

			cmd := newCmdReleaseUpload(newUploadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"alice/demo", "v1/rc", path, "--overwrite"})

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tc.wantError {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not contain %q", err.Error(), want)
				}
			}
			if transport.totalCalls != 1 {
				t.Fatalf("total calls = %d, want 1; calls=%v", transport.totalCalls, transport.calls)
			}
			if len(transport.apiCalls) != 1 || transport.apiCalls[0].Method != http.MethodGet {
				t.Fatalf("api calls = %+v, want exactly one GET release", transport.apiCalls)
			}
			if transport.putCalls != 0 {
				t.Fatalf("external PUT calls = %d, want 0", transport.putCalls)
			}
		})
	}
}

func TestReleaseUploadHTTPErrorFirstBatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	cases := []struct {
		name       string
		args       []string
		apiHandler func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error)
		wantError  []string
		wantCalls  int
		wantPUTs   int
	}{
		{
			name: "GET release 404",
			args: []string{"alice/demo", "v1/rc", path},
			apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
				return uploadJSONResponse(http.StatusNotFound, `{"message":"not found"}`), nil
			},
			wantError: []string{"failed to get release before uploading", "404"},
			wantCalls: 1,
			wantPUTs:  0,
		},
		{
			name: "DELETE 403 during overwrite",
			args: []string{"alice/demo", "v1/rc", path, "--overwrite"},
			apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
				switch {
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
					return uploadJSONResponse(
						http.StatusOK,
						releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}),
					), nil
				case method == http.MethodGet && strings.HasSuffix(escapedPath, "/upload_url"):
					return uploadJSONResponse(
						http.StatusOK,
						`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream"}}`,
					), nil
				case method == http.MethodDelete && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
					return uploadJSONResponse(http.StatusForbidden, `{"message":"forbidden"}`), nil
				default:
					return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
				}
			},
			wantError: []string{"failed to delete existing attachment", "403"},
			wantCalls: 4,
			wantPUTs:  0,
		},
		{
			name: "GET upload_url 429 before overwrite delete",
			args: []string{"alice/demo", "v1/rc", path, "--overwrite"},
			apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
				switch {
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
					return uploadJSONResponse(
						http.StatusOK,
						releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}),
					), nil
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
					return uploadJSONResponse(http.StatusTooManyRequests, `{"message":"too many requests"}`), nil
				default:
					return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
				}
			},
			wantError: []string{"failed to get upload url", "429"},
			wantCalls: 2,
			wantPUTs:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &uploadTransport{apiHandler: tc.apiHandler}
			cmd := newCmdReleaseUpload(newUploadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tc.wantError {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not contain %q", err.Error(), want)
				}
			}
			if transport.totalCalls != tc.wantCalls {
				t.Fatalf("total calls = %d, want %d; calls=%v", transport.totalCalls, tc.wantCalls, transport.calls)
			}
			if transport.putCalls != tc.wantPUTs {
				t.Fatalf("external PUT calls = %d, want %d", transport.putCalls, tc.wantPUTs)
			}
			if out.String() != "" {
				t.Fatalf("output = %q, want empty", out.String())
			}
		})
	}
}

func TestReleaseUploadPUTFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseName := filepath.Base(path)

	cases := []struct {
		name       string
		args       []string
		apiHandler func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error)
		wantError  []string
		wantCalls  int
	}{
		{
			name: "plain upload PUT 500",
			args: []string{"alice/demo", "v1/rc", path},
			apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
				switch {
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
					return uploadJSONResponse(http.StatusOK, releaseWithAssetsJSON("v1/rc", nil)), nil
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
					return uploadJSONResponse(
						http.StatusOK,
						`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream","X-MD5":"deadbeef"}}`,
					), nil
				default:
					return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
				}
			},
			wantError: []string{"failed to upload attachment", "500"},
			wantCalls: 3,
		},
		{
			name: "overwrite then PUT 500",
			args: []string{"alice/demo", "v1/rc", path, "--overwrite"},
			apiHandler: func(method, escapedPath, rawQuery string, body []byte) (*http.Response, error) {
				switch {
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
					return uploadJSONResponse(
						http.StatusOK,
						releaseWithAssetsJSON("v1/rc", []uploadAsset{{ID: 42, Name: baseName, Type: "attach"}}),
					), nil
				case method == http.MethodDelete && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
					return uploadJSONResponse(http.StatusNoContent, ""), nil
				case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
					return uploadJSONResponse(
						http.StatusOK,
						`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream","X-MD5":"deadbeef"}}`,
					), nil
				default:
					return uploadNotFoundHandler(method, escapedPath, rawQuery, body)
				}
			},
			wantError: []string{"upload after overwrite failed", "old attachment", "already deleted", "500"},
			wantCalls: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &uploadTransport{
				putStatusCode: http.StatusInternalServerError,
				apiHandler:    tc.apiHandler,
			}
			cmd := newCmdReleaseUpload(newUploadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tc.wantError {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not contain %q", err.Error(), want)
				}
			}
			if transport.totalCalls != tc.wantCalls {
				t.Fatalf("total calls = %d, want %d; calls=%v", transport.totalCalls, tc.wantCalls, transport.calls)
			}
			if transport.putCalls != 1 {
				t.Fatalf("external PUT calls = %d, want 1", transport.putCalls)
			}
			if out.String() != "" {
				t.Fatalf("output = %q, want empty", out.String())
			}
		})
	}
}
