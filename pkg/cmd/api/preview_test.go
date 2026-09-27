package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestDryRunMatchesPreparedRequestsWithoutDependencies(t *testing.T) {
	file := filepath.Join(t.TempDir(), "private-filename")
	const input = `{"password":"private-password","nested":{"token":"private-nested"},"items":["private-item"]}`
	if err := os.WriteFile(file, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		endpoint string
		opts     options
		args     []string
		stdin    string
		source   string
	}{
		{name: "GET query and duplicate fields", endpoint: "repos/alice/demo/issues?q=private-query", opts: options{method: "GET", fields: []string{"q=private-second", "state=private-state"}}, args: []string{"repos/alice/demo/issues?q=private-query", "-f", "q=private-second", "-f", "state=private-state"}, source: "none"},
		{name: "POST fields", endpoint: "/repos/alice/demo/issues", opts: options{method: "POST", fields: []string{"title=private-old", "title=private-last", "password=x!"}}, args: []string{"/repos/alice/demo/issues", "-X", "POST", "-f", "title=private-old", "-f", "title=private-last", "-f", "password=x!"}, source: "fields"},
		{name: "PATCH file", endpoint: "/repos/alice/demo/issues/42", opts: options{method: "PATCH", input: file}, args: []string{"/repos/alice/demo/issues/42", "-X", "PATCH", "--input", file}, source: "file"},
		{name: "PUT stdin", endpoint: "/user", opts: options{method: "PUT", input: "-"}, args: []string{"/user", "-X", "PUT", "--input", "-"}, stdin: input, source: "stdin"},
		{name: "DELETE", endpoint: "/repos/alice/demo/hooks/42", opts: options{method: "DELETE"}, args: []string{"/repos/alice/demo/hooks/42", "-X", "DELETE"}, source: "none"},
		{name: "GET body", endpoint: "/user", opts: options{method: "GET", input: "-"}, args: []string{"/user", "--input", "-"}, stdin: "private-opaque", source: "stdin"},
		{name: "default pagination", endpoint: "/repos/alice/demo/issues", opts: options{method: "GET", paginate: true}, args: []string{"/repos/alice/demo/issues", "--paginate"}, source: "none"},
		{name: "custom pagination", endpoint: "/repos/alice/demo/issues?page=3", opts: options{method: "GET", paginate: true, fields: []string{"per_page=7"}}, args: []string{"/repos/alice/demo/issues?page=3", "--paginate", "-f", "per_page=7"}, source: "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			opts.accept = "application/json"
			realRequest, err := prepare(tc.endpoint, opts, strings.NewReader(tc.stdin))
			if err != nil {
				t.Fatal(err)
			}
			opts.dryRun = true
			previewRequest, err := prepare(tc.endpoint, opts, strings.NewReader(tc.stdin))
			if err != nil || !reflect.DeepEqual(realRequest, previewRequest) {
				t.Fatalf("preparation differs: %v", err)
			}
			fixture := newAPITestFixture("private-token", nil)
			fixture.config.tokenErr = errors.New("credentials must not be read")
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetIn(strings.NewReader(tc.stdin))
			cmd.SetOut(cmdutil.NewSanitizingWriter(fixture.stdout))
			cmd.SetErr(fixture.stderr)
			cmd.SetArgs(append(tc.args, "--dry-run"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 || fixture.stderr.Len() != 0 {
				t.Fatal("dry run accessed dependencies or emitted diagnostics")
			}
			var preview requestPreview
			if err := json.Unmarshal(fixture.stdout.Bytes(), &preview); err != nil {
				t.Fatal(err)
			}
			if preview.SchemaVersion != 1 || !preview.DryRun || preview.Executed || preview.Method != realRequest.method || preview.Host != "api.atomgit.com" || preview.APIVersion != "v5" || preview.BasePath != "/api/v5" {
				t.Fatalf("unexpected preview: %s", fixture.stdout)
			}
			if preview.Body.Source != tc.source || preview.Body.ByteLength != len(realRequest.body) || preview.Body.ContentType != realRequest.contentType {
				t.Fatalf("body metadata differs: %+v", preview.Body)
			}
			if preview.Pagination.Enabled != tc.opts.paginate {
				t.Fatalf("pagination = %+v", preview.Pagination)
			}
			if realRequest.pagination != nil && (*preview.Pagination.FirstPage != realRequest.pagination.page || *preview.Pagination.PerPage != realRequest.pagination.perPage || preview.Pagination.Strategy != "total_page-or-short-array") {
				t.Fatalf("pagination differs: %+v", preview.Pagination)
			}
			if !strings.HasSuffix(fixture.stdout.String(), "\n") {
				t.Fatal("preview has no trailing newline")
			}
			for _, secret := range []string{"private-", "alice", "demo", "x!"} {
				assertNoTokenLeak(t, secret, fixture.stdout.String())
			}
			// Capture the real-mode request as well, without touching a service.
			actual := newAPITestFixture("test-token", func(req *http.Request) (*http.Response, error) {
				if req.Method != realRequest.method || req.URL.String() != realRequest.baseURL+realRequest.path {
					t.Fatalf("actual request differs: %s %s", req.Method, req.URL)
				}
				var body []byte
				if req.Body != nil {
					body, err = io.ReadAll(req.Body)
					if err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(body, realRequest.body) || req.Header.Get("Content-Type") != realRequest.contentType {
					t.Fatal("actual body differs")
				}
				return apiTestResponse(req, http.StatusOK, "[]"), nil
			})
			actualCmd := NewCmdAPI(actual.factory)
			actualCmd.SetOut(io.Discard)
			if err := execute(actualCmd, actual.factory, realRequest); err != nil || actual.requests != 1 {
				t.Fatalf("real execution: %v, requests: %d", err, actual.requests)
			}
		})
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != input {
		t.Fatal("input file changed")
	}
}

func TestDryRunStableShape(t *testing.T) {
	cmd := NewCmdAPI(nil)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"repos/alice/demo/issues?state=open&state=closed&private-key=value", "-X", "POST", "-f", "title=private-title", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	const expected = `{"schemaVersion":1,"dryRun":true,"executed":false,"method":"POST","apiVersion":"v5","host":"api.atomgit.com","basePath":"/api/v5","path":"/repos/[redacted]/[redacted]/issues","query":[{"name":"[redacted]","type":"string","count":1},{"name":"state","type":"string","count":2}],"accept":"application/json","body":{"source":"fields","contentType":"application/json","byteLength":25,"type":"object","count":1,"fields":[{"name":"title","type":"string"}],"truncated":false},"pagination":{"enabled":false,"firstPage":null,"perPage":null,"strategy":"none"}}`
	var got, want any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preview = %s", out.String())
	}
}

func TestDryRunOmitsUntrustedBodyAndHeaderData(t *testing.T) {
	for _, body := range []string{
		`{"title":"private-title","password":"x!","nested":{"secret":"private-secret"},"private-key":"private-value","array":["private-array"],"count":123456789,"ok":true,"none":null}`,
		`["private-element", {"private-key":"private-value"}]`,
		`"private-scalar"`, `123456789`, `true`, `null`,
		"private-body\x00\x1b[31m\x7f\u202e", "", `{"broken":"private-body"`,
	} {
		t.Run(body, func(t *testing.T) {
			cmd := NewCmdAPI(nil)
			var out bytes.Buffer
			// No sanitizing writer: --raw-output must not disable redaction.
			cmd.SetOut(&out)
			cmd.SetIn(strings.NewReader(body))
			cmd.SetArgs([]string{"/repos/private-owner/private-repo/%1b?token=x!&code=private-code&password=private-password&private-key=private-value", "-X", "PUT", "--input", "-", "--accept", "private-accept", "--dry-run"})
			if err := cmd.Execute(); err != nil || !json.Valid(out.Bytes()) {
				t.Fatalf("%v: %q", err, out.String())
			}
			for _, secret := range []string{"private-", "x!", "123456789", "\x1b", "\x7f", "\u202e"} {
				assertNoTokenLeak(t, secret, out.String())
			}
		})
	}
}

func TestDryRunPreservesValidationAndHidesErrorInputs(t *testing.T) {
	cases := []struct {
		endpoint string
		opts     options
		stdin    io.Reader
	}{
		{endpoint: "https://name:private-password@example.com/path", opts: options{method: "GET"}},
		{endpoint: "/private-secret%zz", opts: options{method: "GET"}},
		{endpoint: "/user?code=private-secret%zz", opts: options{method: "GET"}},
		{endpoint: "/../private-secret", opts: options{method: "GET"}},
		{endpoint: "/user#private-secret", opts: options{method: "GET"}},
		{endpoint: "/user", opts: options{method: "private-method"}},
		{endpoint: "/user", opts: options{method: "GET", fields: []string{"private-field"}}},
		{endpoint: "/user", opts: options{method: "GET", input: filepath.Join(t.TempDir(), "private-missing")}},
		{endpoint: "/user", opts: options{method: "GET", input: "-"}, stdin: failingReader{err: errors.New("private-reader-error")}},
		{endpoint: "/user", opts: options{method: "GET", fields: []string{"title=private-title"}, input: "-"}},
		{endpoint: "/user", opts: options{method: "DELETE", paginate: true}},
		{endpoint: "/user", opts: options{method: "GET", paginate: true, input: "-"}},
		{endpoint: "/user?page=private-page", opts: options{method: "GET", paginate: true}},
		{endpoint: "/user?page=1&page=2", opts: options{method: "GET", paginate: true}},
	}
	for _, tc := range cases {
		opts := tc.opts
		opts.accept = "application/json"
		_, realErr := prepare(tc.endpoint, opts, tc.stdin)
		opts.dryRun = true
		_, previewErr := prepare(tc.endpoint, opts, tc.stdin)
		if realErr == nil || previewErr == nil {
			t.Fatalf("validation differs for %+v: %v / %v", tc, realErr, previewErr)
		}
		assertNoTokenLeak(t, "private-", previewErr.Error())
	}
}

func TestPreviewBodyShapeAndLimits(t *testing.T) {
	cases := []struct {
		body  string
		kind  string
		count int
	}{
		{body: "", kind: "opaque"},
		{body: "not JSON", kind: "opaque"},
		{body: `"private-scalar"`, kind: "string"},
		{body: `1e1000000`, kind: "number"},
		{body: `true`, kind: "boolean"},
		{body: `null`, kind: "null"},
		{body: `[]`, kind: "array"},
		{body: `[null,"private-scalar",{}]`, kind: "array", count: 3},
		{body: `{}`, kind: "object"},
		{body: `{"title":"private-value","private-key":[],"body":{}}`, kind: "object", count: 3},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			result := describeBody(preparedRequest{bodySource: "stdin", body: []byte(tc.body)})
			if result.Type != tc.kind || result.Count != tc.count || result.ByteLength != len(tc.body) || result.ContentType != "" || result.Fields == nil || result.Truncated {
				t.Fatalf("summary = %+v", result)
			}
		})
	}
	fields := make(map[string]any)
	for i := range 60 {
		fields["private-key-"+strings.Repeat("x", i)] = "private-value"
	}
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	result := describeBody(preparedRequest{bodySource: "file", body: body})
	if !result.Truncated || len(result.Fields) != 50 || result.Count != 60 {
		t.Fatalf("unbounded summary: %+v", result)
	}
}

func TestDryRunEncodedSecretsAndControlCharacters(t *testing.T) {
	const secret = "private-secret\x00\x1b[31m\x7f\u202e"
	body, err := json.Marshal(map[string]any{secret: secret, "title": secret, "nested": map[string]string{"code": secret}})
	if err != nil {
		t.Fatal(err)
	}
	cmd := NewCmdAPI(nil)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(bytes.NewReader(body))
	cmd.SetArgs([]string{"/repos/private%2Fsecret/%7f?private%1b=private-secret&access_token=private-secret&webhook_secret=x!", "--input", "-", "-X", "POST", "--dry-run"})
	if err := cmd.Execute(); err != nil || !json.Valid(out.Bytes()) {
		t.Fatalf("preview failed: %v: %q", err, out.String())
	}
	for _, forbidden := range []string{"private", "x!", "\x00", "\x1b", "\x7f", "\u202e"} {
		assertNoTokenLeak(t, forbidden, out.String())
	}
}

func TestDryRunErrorsHaveNoOutputOrDependencies(t *testing.T) {
	for _, args := range [][]string{
		{"/user", "--input", filepath.Join(t.TempDir(), "private-missing")},
		{"/user", "--field", "private-invalid-field"},
		{"/user", "--accept", "private-header\r\n"},
		{"/user", "--method", "private-method\x1b"},
		{"/user", "--input", "-"},
	} {
		fixture := newAPITestFixture("x!", nil)
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetOut(fixture.stdout)
		cmd.SetErr(fixture.stderr)
		cause := errors.New("private-reader-error")
		cmd.SetIn(failingReader{err: cause})
		cmd.SetArgs(append(args, "--dry-run"))
		err := cmd.Execute()
		if err == nil || fixture.stdout.Len() != 0 || fixture.clientReads != 0 || fixture.config.tokenReads != 0 {
			t.Fatalf("failure had side effects: %v %q", err, fixture.stdout)
		}
		assertNoTokenLeak(t, "private-", err.Error(), fixture.stderr.String())
		if args[len(args)-1] == "-" && !errors.Is(err, cause) {
			t.Fatal("read error identity lost")
		}
	}
}

func TestAPIFlagErrorsDoNotEchoInputs(t *testing.T) {
	for _, badFlag := range []string{"--paginate=private-secret", "--dry-run=private-secret", "--private-secret=value", "-private-secret"} {
		for _, position := range []string{"before", "after", "absent"} {
			t.Run(badFlag+"/"+position, func(t *testing.T) {
				args := []string{"/user"}
				if position == "before" {
					args = append(args, "--dry-run")
				}
				args = append(args, badFlag)
				if position == "after" {
					args = append(args, "--dry-run")
				}
				fixture := newAPITestFixture("x!", nil)
				cmd := NewCmdAPI(fixture.factory)
				cmd.SetOut(fixture.stdout)
				cmd.SetErr(fixture.stderr)
				cmd.SetArgs(args)
				err := cmd.Execute()
				if err == nil || fixture.stdout.Len() != 0 || fixture.clientReads != 0 || fixture.config.tokenReads != 0 {
					t.Fatalf("flag error had side effects: %v %q", err, fixture.stdout)
				}
				assertNoTokenLeak(t, "private-", err.Error(), fixture.stderr.String())
				if errors.Unwrap(err) == nil {
					t.Fatal("flag error identity lost")
				}
			})
		}
	}
}
