package release

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type downloadTransport struct {
	totalCalls int
	requests   []downloadRequest
	apiHandler func(method, escapedPath, rawQuery, accept string) (*http.Response, error)
}

type downloadRequest struct {
	Method      string
	EscapedPath string
	Accept      string
}

func (t *downloadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.totalCalls++
	t.requests = append(t.requests, downloadRequest{
		Method:      req.Method,
		EscapedPath: req.URL.EscapedPath(),
		Accept:      req.Header.Get("Accept"),
	})
	handler := t.apiHandler
	if handler == nil {
		handler = downloadNotFoundHandler
	}
	return handler(req.Method, req.URL.EscapedPath(), req.URL.RawQuery, req.Header.Get("Accept"))
}

func downloadNotFoundHandler(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       io.NopCloser(strings.NewReader("not found")),
		Header:     make(http.Header),
	}, nil
}

func downloadJSONResponse(statusCode int, body string) *http.Response {
	return releaseResponse(statusCode, body)
}

func newDownloadFactory(transport *downloadTransport) *cmdutil.Factory {
	return releaseTestFactory(transport)
}

func setupDownloadReleaseHandler(tag, assetName string, assetID int64, downloadStatusCode int, downloadBody string) func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
	releaseBody := releaseWithAssetsJSON(tag, []uploadAsset{{ID: assetID, Name: assetName, Type: "attach"}})
	return func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
		switch {
		case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
			return downloadJSONResponse(http.StatusOK, releaseBody), nil
		case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
			return &http.Response{
				StatusCode: downloadStatusCode,
				Status:     fmt.Sprintf("%d %s", downloadStatusCode, http.StatusText(downloadStatusCode)),
				Body:       io.NopCloser(strings.NewReader(downloadBody)),
				Header:     make(http.Header),
			}, nil
		default:
			return downloadNotFoundHandler(method, escapedPath, rawQuery, accept)
		}
	}
}

func TestReleaseDownloadRegistersCommandFlagsAndArgs(t *testing.T) {
	root := NewCmdRelease(&cmdutil.Factory{})
	download, _, err := root.Find([]string{"download"})
	if err != nil || download == nil {
		t.Fatal("release download not found")
	}
	if got, want := download.Use, "download [<owner>/<repo>] <tag> <asset>"; got != want {
		t.Fatalf("Use = %q, want %q", got, want)
	}

	if err := download.Args(download, []string{"v1.0.0", "app.tar.gz"}); err != nil {
		t.Fatalf("Args(2) = %v, want nil", err)
	}
	if err := download.Args(download, []string{"alice/demo", "v1.0.0", "app.tar.gz"}); err != nil {
		t.Fatalf("Args(3) = %v, want nil", err)
	}
	if err := download.Args(download, []string{"v1.0.0"}); err == nil {
		t.Fatal("Args(1) = nil, want error")
	}
	if err := download.Args(download, []string{"a", "b", "c", "d"}); err == nil {
		t.Fatal("Args(4) = nil, want error")
	}

	outputFlag := download.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("--output flag was not registered")
	}
	if outputFlag.Shorthand != "o" {
		t.Fatalf("--output shorthand = %q, want %q", outputFlag.Shorthand, "o")
	}
	root.SilenceUsage = true
	var rootOut bytes.Buffer
	root.SetOut(&rootOut)
	root.SetArgs([]string{"download", "v1.0.0", "app.tar.gz"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "required flag(s) \"output\"") {
		t.Fatalf("root Execute without --output: err = %v, want error mentioning required output", err)
	}

	if download.Flags().Lookup("overwrite") == nil {
		t.Fatal("--overwrite flag was not registered")
	}
	timeoutFlag := download.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("--timeout flag was not registered")
	}
	if got := timeoutFlag.DefValue; got != defaultReleaseTransferTimeout.String() {
		t.Fatalf("--timeout default = %q, want %q", got, defaultReleaseTransferTimeout.String())
	}
	for _, name := range []string{"yes", "skip-existing", "name"} {
		if download.Flags().Lookup(name) != nil {
			t.Fatalf("--%s flag should not be registered on download", name)
		}
	}
}

func TestReleaseDownloadExistingOutputWithoutOverwriteIssuesZeroRequests(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "existing.tar.gz")
	if err := os.WriteFile(output, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	transport := &downloadTransport{}
	cmd := newCmdReleaseDownload(newDownloadFactory(transport))
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1.0.0", "app.tar.gz", "-o", output})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want error containing \"already exists\"", err)
	}
	if transport.totalCalls != 0 {
		t.Fatalf("total calls = %d, want 0; requests=%+v", transport.totalCalls, transport.requests)
	}
	if got, err := os.ReadFile(output); err != nil || string(got) != "old" {
		t.Fatalf("output = %q, %v; want %q unchanged", got, err, "old")
	}
}

func TestReleaseDownloadSuccessInfersRepoAndEscapesPath(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "app one.zip")
	const payload = "release-binary-payload"

	transport := &downloadTransport{
		apiHandler: setupDownloadReleaseHandler("v1/rc", "dir/app one.zip", 42, http.StatusOK, payload),
	}
	cmd := newCmdReleaseDownload(newDownloadFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1/rc", "dir/app one.zip", "-o", output})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if transport.totalCalls != 2 {
		t.Fatalf("total calls = %d, want 2; requests=%+v", transport.totalCalls, transport.requests)
	}

	getRelease := transport.requests[0]
	if getRelease.Method != http.MethodGet {
		t.Fatalf("release method = %q, want GET", getRelease.Method)
	}
	const wantReleasePath = "/api/v5/repos/alice/demo/releases/tags/v1%2Frc"
	if getRelease.EscapedPath != wantReleasePath {
		t.Fatalf("release path = %q, want %q", getRelease.EscapedPath, wantReleasePath)
	}

	getDownload := transport.requests[1]
	if getDownload.Method != http.MethodGet {
		t.Fatalf("download method = %q, want GET", getDownload.Method)
	}
	const wantDownloadPath = "/api/v5/repos/alice/demo/releases/v1%2Frc/attach_files/dir%2Fapp%20one.zip/download"
	if got := getDownload.EscapedPath; got != wantDownloadPath {
		t.Fatalf("download path = %q, want %q", got, wantDownloadPath)
	}
	if got := getDownload.Accept; got != "*/*" {
		t.Fatalf("download Accept = %q, want %q", got, "*/*")
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("output payload = %q, want %q", got, payload)
	}

	wantOut := "Downloaded attachment dir/app one.zip from release v1/rc to " + output + "\n"
	if got := out.String(); got != wantOut {
		t.Fatalf("output = %q, want %q", got, wantOut)
	}
}

func TestReleaseDownloadOverwriteReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "app.tar.gz")
	if err := os.WriteFile(output, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	const payload = "new-payload"

	transport := &downloadTransport{
		apiHandler: setupDownloadReleaseHandler("v1.0.0", "app.tar.gz", 7, http.StatusOK, payload),
	}
	cmd := newCmdReleaseDownload(newDownloadFactory(transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1.0.0", "app.tar.gz", "-o", output, "--overwrite"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("output payload = %q, want %q", got, payload)
	}
}

// failingDownloadBody simulates an interrupted response body.
type failingDownloadBody struct {
	read   int
	closed bool
}

func (b *failingDownloadBody) Read(p []byte) (int, error) {
	b.read++
	if b.read == 1 {
		copy(p, "partial")
		return len("partial"), nil
	}
	return 0, errors.New("read interrupted")
}

func (b *failingDownloadBody) Close() error { b.closed = true; return nil }

func TestReleaseDownloadEmptyTagOrAssetFailsBeforeNetwork(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"empty tag", []string{"", "app.tar.gz"}, "release tag is required"},
		{"whitespace tag", []string{"   ", "app.tar.gz"}, "release tag is required"},
		{"empty asset", []string{"v1.0.0", ""}, "attachment name is required"},
		{"whitespace asset", []string{"v1.0.0", "   "}, "attachment name is required"},
		{"negative timeout", []string{"v1.0.0", "app.tar.gz", "--timeout", "-1s"}, "must not be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "out.tar.gz")

			transport := &downloadTransport{}
			cmd := newCmdReleaseDownload(newDownloadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(append(append([]string{}, tt.args...), "-o", output))

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want error containing %q", err, tt.want)
			}
			if transport.totalCalls != 0 {
				t.Fatalf("total calls = %d, want 0; requests=%+v", transport.totalCalls, transport.requests)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Fatalf("output exists after local failure: %v", statErr)
			}
		})
	}
}

func TestReleaseDownloadAssetScreeningRejections(t *testing.T) {
	tests := []struct {
		name   string
		assets []uploadAsset
		want   string
	}{
		{
			name:   "not found",
			assets: []uploadAsset{{ID: 9, Name: "other.tar.gz", Type: "attach"}},
			want:   "not found",
		},
		{
			name:   "duplicate same name",
			assets: []uploadAsset{{ID: 7, Name: "app.tar.gz", Type: "attach"}, {ID: 8, Name: "app.tar.gz", Type: "attach"}},
			want:   "need exactly one",
		},
		{
			name:   "source asset",
			assets: []uploadAsset{{ID: 1, Name: "app.tar.gz", Type: "source"}},
			want:   "not \"attach\"",
		},
		{
			name:   "attach id zero",
			assets: []uploadAsset{{ID: 0, Name: "app.tar.gz", Type: "attach"}},
			want:   "invalid id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "out.tar.gz")

			const tag = "v1.0.0"
			releaseBody := releaseWithAssetsJSON(tag, tt.assets)
			transport := &downloadTransport{
				apiHandler: func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
					return downloadJSONResponse(http.StatusOK, releaseBody), nil
				},
			}
			cmd := newCmdReleaseDownload(newDownloadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{tag, "app.tar.gz", "-o", output})

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want error containing %q", err, tt.want)
			}
			if transport.totalCalls != 1 {
				t.Fatalf("total calls = %d, want 1 (release only); requests=%+v", transport.totalCalls, transport.requests)
			}
			if got := transport.requests[0].EscapedPath; !strings.HasPrefix(got, "/api/v5/repos/alice/demo/releases/tags/") {
				t.Fatalf("first request path = %q, want release-by-tag path", got)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Fatalf("output exists after screening rejection: %v", statErr)
			}
		})
	}
}

func TestReleaseDownloadReleaseNotFound(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.tar.gz")

	transport := &downloadTransport{
		apiHandler: func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
			return downloadJSONResponse(http.StatusNotFound, "not found"), nil
		},
	}
	cmd := newCmdReleaseDownload(newDownloadFactory(transport))
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1.0.0", "app.tar.gz", "-o", output})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("error = nil, want non-nil for release 404")
	}
	if !strings.Contains(err.Error(), "failed to get release") {
		t.Fatalf("error = %v, want error containing \"failed to get release\"", err)
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want error containing the 404 status", err)
	}
	if transport.totalCalls != 1 {
		t.Fatalf("total calls = %d, want 1", transport.totalCalls)
	}
	if got := transport.requests[0].Method; got != http.MethodGet {
		t.Fatalf("request method = %q, want GET", got)
	}
	if got := transport.requests[0].EscapedPath; !strings.HasPrefix(got, "/api/v5/repos/alice/demo/releases/tags/") {
		t.Fatalf("request path = %q, want release-by-tag path", got)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output exists after release 404: %v", statErr)
	}
}

func TestReleaseDownloadAttachmentHTTPFailures(t *testing.T) {
	for _, statusCode := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "out.tar.gz")

			transport := &downloadTransport{
				apiHandler: func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
					switch {
					case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
						return downloadJSONResponse(http.StatusOK, releaseWithAssetsJSON("v1.0.0", []uploadAsset{{ID: 5, Name: "app.tar.gz", Type: "attach"}})), nil
					case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
						return &http.Response{
							StatusCode: statusCode,
							Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
							Body:       io.NopCloser(strings.NewReader("error body")),
							Header:     make(http.Header),
						}, nil
					default:
						return downloadNotFoundHandler(method, escapedPath, rawQuery, accept)
					}
				},
			}
			cmd := newCmdReleaseDownload(newDownloadFactory(transport))
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"v1.0.0", "app.tar.gz", "-o", output})

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("error = nil, want non-nil for status %d", statusCode)
			}
			want := fmt.Sprintf("%d", statusCode)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want error containing status %q", err, want)
			}
			if transport.totalCalls != 2 {
				t.Fatalf("total calls = %d, want 2", transport.totalCalls)
			}
			if got := transport.requests[0].EscapedPath; !strings.HasPrefix(got, "/api/v5/repos/alice/demo/releases/tags/") {
				t.Fatalf("first request path = %q, want release-by-tag path", got)
			}
			if got := transport.requests[1].EscapedPath; !strings.HasSuffix(got, "/attach_files/app.tar.gz/download") {
				t.Fatalf("second request path = %q, want attach download path", got)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Fatalf("output exists after download failure: %v", statErr)
			}
		})
	}
}

func TestReleaseDownloadTransferInterruptPreservesOld(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "app.tar.gz")
	const old = "old-payload"
	if err := os.WriteFile(output, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	body := &failingDownloadBody{}
	releaseBody := releaseWithAssetsJSON("v1.0.0", []uploadAsset{{ID: 3, Name: "app.tar.gz", Type: "attach"}})
	transport := &downloadTransport{
		apiHandler: func(method, escapedPath, rawQuery, accept string) (*http.Response, error) {
			switch {
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/tags/"):
				return downloadJSONResponse(http.StatusOK, releaseBody), nil
			case method == http.MethodGet && strings.HasPrefix(escapedPath, "/api/v5/repos/alice/demo/releases/"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       body,
					Header:     make(http.Header),
				}, nil
			default:
				return downloadNotFoundHandler(method, escapedPath, rawQuery, accept)
			}
		},
	}
	cmd := newCmdReleaseDownload(newDownloadFactory(transport))
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"v1.0.0", "app.tar.gz", "-o", output, "--overwrite"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "read interrupted") {
		t.Fatalf("error = %v, want error containing \"read interrupted\"", err)
	}
	if got, readErr := os.ReadFile(output); readErr != nil || string(got) != old {
		t.Fatalf("output = %q, %v; want %q preserved", got, readErr, old)
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".app.tar.gz.tmp-*"))
	if globErr != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, %v; want none", matches, globErr)
	}
	if !body.closed {
		t.Fatal("response body was not closed after the interrupted transfer")
	}
}
