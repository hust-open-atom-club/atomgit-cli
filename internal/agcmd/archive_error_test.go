package agcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestArchiveRequestDiagnosticsRetainSafeFailureFacts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		cause  error
		detail string
	}{
		{name: "permission", status: 403, detail: "The server returned HTTP 403."},
		{name: "not found", status: 404, detail: "The server returned HTTP 404."},
		{name: "server", status: 500, detail: "The server returned HTTP 500."},
		{name: "canceled", cause: context.Canceled, detail: "The HTTP request was canceled."},
		{name: "transport", cause: errors.New("private-password"), detail: "The HTTP request could not be completed."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := api.NewClientWithHTTPClient("fixture", &http.Client{Transport: statusTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.cause != nil {
					return nil, tc.cause
				}
				return &http.Response{StatusCode: tc.status, Status: "private-password", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"msg":"private-password"}`)), Request: r}, nil
			})})
			err := api.SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 2, nil)
			if err == nil || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
			for _, raw := range []bool{false, true} {
				var out bytes.Buffer
				writer := cmdutil.NewSanitizingWriter(&out)
				if e := writer.SetRaw(raw); e != nil {
					t.Fatal(e)
				}
				renderCommandError(writer, fmt.Errorf("archive failed: %w", err))
				if e := cmdutil.FlushWriter(writer); e != nil {
					t.Fatal(e)
				}
				if !strings.Contains(out.String(), tc.detail) || strings.Contains(out.String(), "private-password") {
					t.Fatalf("raw=%t diagnostic=%q", raw, out.String())
				}
			}
		})
	}
}
