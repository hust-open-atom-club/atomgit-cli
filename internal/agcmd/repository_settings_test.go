package agcmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepositorySettingsFailureDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		body, detail string
		cause        error
		uncertain    bool
	}{
		{name: "permission", status: 403, body: `{"secret":"private-token"}`, detail: "The server returned HTTP 403."},
		{name: "unexpected success", status: 201, detail: "The server returned HTTP 201.", uncertain: true},
		{name: "server error", status: 500, body: `{"secret":"private-token"}`, detail: "The server returned HTTP 500.", uncertain: true},
		{name: "malformed", status: 200, body: `{`, detail: "The server returned an invalid response.", uncertain: true},
		{name: "transport", cause: errors.New("private-token"), detail: "The HTTP request could not be completed.", uncertain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := api.NewClientWithHTTPClient("private-token", &http.Client{Transport: statusTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.cause != nil {
					return nil, tc.cause
				}
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})})
			value := false
			_, err := api.UpdateRepositorySettings(client, "team", "demo", api.UpdateRepositorySettingsRequest{ActionPipelineEnabled: &value})
			if err == nil || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
			for _, raw := range []bool{false, true} {
				var out bytes.Buffer
				writer := cmdutil.NewSanitizingWriter(&out)
				if err := writer.SetRaw(raw); err != nil {
					t.Fatal(err)
				}
				renderCommandError(writer, err)
				if err := cmdutil.FlushWriter(writer); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), tc.detail) || strings.Contains(out.String(), "private-token") || strings.Contains(out.String(), "may have changed") != tc.uncertain {
					t.Fatalf("raw=%t diagnostic=%q", raw, out.String())
				}
			}
		})
	}
}
