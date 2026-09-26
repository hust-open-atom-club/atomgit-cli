package release

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestReleaseViewJSON(t *testing.T) {
	cases := []struct {
		name     string
		response string
		want     string
	}{
		{
			name: "complete release with attachments and source archive",
			response: `{"tag_name":"v1/rc","name":"Release candidate","draft":true,"prerelease":false,
"release_status":"pre","target_commitish":"main","created_at":"2026-09-25T00:00:00Z",
"author":{"login":"alice","name":"Alice","id":"123","avatar_url":"https://example.com/avatar"},
"body":"First line\n\"quoted\"\ttext\\path\r\n\u001b[31m",
"assets":[{"id":42,"name":"app.zip","type":"attach","browser_download_url":"https://example.com/app.zip"},
{"name":"Source code","type":"source","browser_download_url":"https://example.com/source.zip"}],"future_api_field":"ignored"}`,
			want: `{"tag":"v1/rc","name":"Release candidate","draft":true,"prerelease":false,
"status":"prerelease","targetCommitish":"main","createdAt":"2026-09-25T00:00:00Z","author":"alice",
"body":"First line\n\"quoted\"\ttext\\path\r\n\u001b[31m",
"assets":[{"id":42,"name":"app.zip","type":"attach","browserDownloadUrl":"https://example.com/app.zip"},
{"id":0,"name":"Source code","type":"source","browserDownloadUrl":"https://example.com/source.zip"}]}`,
		},
		{
			name:     "missing optional fields",
			response: `{"tag_name":"v1/rc"}`,
			want:     `{"tag":"v1/rc","name":"","status":"release","draft":false,"prerelease":false,"targetCommitish":"","createdAt":"","author":"","body":"","assets":[]}`,
		},
		{
			name:     "empty assets and author without login",
			response: `{"tag_name":"v1/rc","release_status":"latest","author":{"name":"Alice"},"assets":[]}`,
			want:     `{"tag":"v1/rc","name":"","status":"latest","draft":false,"prerelease":false,"targetCommitish":"","createdAt":"","author":"","body":"","assets":[]}`,
		},
		{
			name:     "null optional fields",
			response: `{"tag_name":"v1/rc","name":null,"body":null,"author":null,"assets":null}`,
			want:     `{"tag":"v1/rc","name":"","status":"release","draft":false,"prerelease":false,"targetCommitish":"","createdAt":"","author":"","body":"","assets":[]}`,
		},
		{
			name:     "asset missing optional fields",
			response: `{"tag_name":"v1/rc","prerelease":true,"assets":[{}]}`,
			want:     `{"tag":"v1/rc","name":"","status":"prerelease","draft":false,"prerelease":true,"targetCommitish":"","createdAt":"","author":"","body":"","assets":[{"id":0,"name":"","type":"","browserDownloadUrl":""}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, explicit := range []bool{false, true} {
				t.Run(map[bool]string{false: "inferred repository", true: "explicit repository"}[explicit], func(t *testing.T) {
					requests := 0
					factory := releaseTestFactory(releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						owner := "alice"
						if explicit {
							owner = "bob"
						}
						if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/repos/"+owner+"/demo/releases/tags/v1%2Frc" {
							t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
						}
						return releaseResponse(http.StatusOK, tc.response), nil
					}))
					cmd := newCmdReleaseView(factory)
					args := []string{"v1/rc", "--json"}
					if explicit {
						args = append([]string{"bob/demo"}, args...)
					}
					cmd.SetArgs(args)
					var stdout, stderr bytes.Buffer
					cmd.SetOut(cmdutil.NewSanitizingWriter(&stdout))
					cmd.SetErr(&stderr)
					if err := cmd.Execute(); err != nil {
						t.Fatal(err)
					}
					if requests != 1 || stderr.Len() != 0 {
						t.Fatalf("requests = %d, stderr = %q", requests, stderr.String())
					}
					if !strings.HasSuffix(stdout.String(), "\n") {
						t.Fatal("JSON output must end with a newline")
					}
					var got, want map[string]any
					if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
						t.Fatalf("output is not one JSON object: %v; %q", err, stdout.String())
					}
					if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("output = %#v, want %#v", got, want)
					}

					// Check the public list output, including raw prerelease and login,
					// instead of assuming the detail DTO's shared mapper is correct.
					list := newCmdReleaseList(releaseTestFactory(releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
						return releaseResponse(http.StatusOK, "["+tc.response+"]"), nil
					})))
					var listOutput bytes.Buffer
					list.SetOut(&listOutput)
					list.SetArgs([]string{"--json"})
					if err := list.Execute(); err != nil {
						t.Fatal(err)
					}
					var rows []map[string]any
					if err := json.Unmarshal(listOutput.Bytes(), &rows); err != nil || len(rows) != 1 {
						t.Fatalf("invalid list output: %q, error: %v", listOutput.String(), err)
					}
					delete(got, "body")
					delete(got, "assets")
					if !reflect.DeepEqual(got, rows[0]) {
						t.Fatalf("shared fields differ: view = %#v, list = %#v", got, rows[0])
					}
				})
			}
		})
	}
}

func TestReleaseViewJSONControlCharacters(t *testing.T) {
	var body strings.Builder
	body.WriteString("中文 release notes\n\"quoted\"\\path")
	for r := rune(0); r <= 0x9f; r++ {
		if r < 0x20 || r >= 0x7f {
			body.WriteString(string(r))
		}
	}
	body.WriteString("\u2028\u2029\u202e\u2066")
	response, err := json.Marshal(api.Release{TagName: "v1", Body: body.String()})
	if err != nil {
		t.Fatal(err)
	}
	cmd := newCmdReleaseView(releaseTestFactory(releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return releaseResponse(http.StatusOK, string(response)), nil
	})))
	var stdout bytes.Buffer
	cmd.SetOut(cmdutil.NewSanitizingWriter(&stdout))
	cmd.SetArgs([]string{"v1", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got struct{ Body string }
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v; %q", err, stdout.String())
	}
	if got.Body != body.String() {
		t.Fatalf("body = %q, want %q", got.Body, body.String())
	}
}

func TestReleaseViewJSONFailureHasNoOutput(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"message":"not found"}`},
		{name: "invalid response", status: http.StatusOK, body: `{`},
		{name: "transport error", err: errors.New("connection failed")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCmdReleaseView(releaseTestFactory(releaseRoundTripFunc(func(*http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return releaseResponse(tc.status, tc.body), nil
			})))
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(io.Discard)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs([]string{"v1", "--json"})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `failed to view release "v1"`) {
				t.Fatalf("error = %v, want contextual query failure", err)
			}
			if stdout.Len() != 0 {
				t.Fatalf("query failure produced stdout: %q", stdout.String())
			}
		})
	}
}
