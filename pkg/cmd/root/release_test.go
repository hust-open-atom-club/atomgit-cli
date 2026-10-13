package root

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestReleaseViewOutputThroughRoot(t *testing.T) {
	const body = "notes\n\"quoted\"\\path\x00\x1b[31m\x7f\u0085\u202e"
	response, err := json.Marshal(map[string]any{
		"tag_name": "v1", "name": "First", "body": body,
		"author": map[string]string{"login": "alice", "name": "Alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	const textOutput = "Name: First\nTag: v1\nTarget: \nStatus: release\nCreated: \nAuthor: Alice (alice)\nBody: " + body + "\nAssets:\n  None\n"
	cases := []struct {
		name  string
		flags []string
		json  bool
		raw   bool
	}{
		{name: "default text"},
		{name: "explicit text", flags: []string{"--json=false"}},
		{name: "raw text", flags: []string{"--raw-output"}, raw: true},
		{name: "JSON", flags: []string{"--json"}, json: true},
		{name: "raw JSON", flags: []string{"--json", "--raw-output"}, json: true, raw: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: rootTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: rootRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/releases/tags/v1" {
							t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
						}
						return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(response)), Header: make(http.Header)}, nil
					})}, nil
				},
			}
			var stdout, stderr bytes.Buffer
			cmd, err := newCmdRootWithWriters(factory, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(append([]string{"release", "view", "alice/demo", "v1"}, tc.flags...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if err := cmdutil.FlushWriter(cmd.OutOrStdout()); err != nil {
				t.Fatal(err)
			}
			if requests != 1 || stderr.Len() != 0 {
				t.Fatalf("requests = %d, stderr = %q", requests, stderr.String())
			}
			if tc.json {
				var got struct{ Body string }
				if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
					t.Fatalf("invalid JSON: %v; %q", err, stdout.String())
				}
				if got.Body != body || !strings.HasSuffix(stdout.String(), "\n") {
					t.Fatalf("JSON body or trailing newline changed: %q", stdout.String())
				}
			} else {
				want := textOutput
				if !tc.raw {
					want = cmdutil.SanitizeTerminal(want)
				}
				if stdout.String() != want {
					t.Fatalf("text = %q, want %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestReleaseViewJSONFailureThroughRoot(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: rootTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: rootRoundTripFunc(func(*http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"message":"request failed"}`)), Header: make(http.Header)}, nil
					})}, nil
				},
			}
			var stdout, stderr bytes.Buffer
			cmd, err := newCmdRootWithWriters(factory, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs([]string{"release", "view", "alice/demo", "v1", "--json"})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `failed to view release "v1"`) {
				t.Fatalf("error = %v, want contextual query failure", err)
			}
			// Main renders the returned error to stderr; Cobra must not add
			// usage text or a partial success object to either stream.
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
			}
		})
	}
}
