package license

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type checkTransport func(*http.Request) (*http.Response, error)

func (f checkTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type checkBody struct {
	data   string
	err    error
	closed bool
}

func (b *checkBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	if b.data == "" {
		if b.err != nil {
			return n, b.err
		}
		return n, io.EOF
	}
	return n, nil
}
func (b *checkBody) Close() error { b.closed = true; return nil }

func TestCheckPreservesStatusReadExecutionAndContext(t *testing.T) {
	cases := []struct {
		name                string
		status              int
		body                string
		readErr, executeErr error
		want                string
	}{
		{"text", 200, "compliant", nil, nil, "compliant\n"},
		{"empty", 200, "", nil, nil, "未知\n"},
		{"status and read", 403, "denied", io.ErrUnexpectedEOF, nil, ""},
		{"success read", 200, "partial", io.ErrUnexpectedEOF, nil, ""},
		{"execution", 200, "", nil, io.ErrClosedPipe, ""},
		{"cancellation", 200, "", nil, context.Canceled, ""},
		{"deadline", 200, "", nil, context.DeadlineExceeded, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &checkBody{data: tc.body, err: tc.readErr}
			calls := 0
			type checkContextKey struct{}
			ctx := context.WithValue(context.Background(), checkContextKey{}, "command-context")
			injected := &http.Client{Transport: checkTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Context().Value(checkContextKey{}) != "command-context" || r.URL.Query().Get("license") != "MIT + BSD" || r.Header.Get("Authorization") != "" {
					t.Fatalf("request context, query or credentials: %v", r)
				}
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > 31*time.Second {
					t.Fatal("default timeout missing")
				}
				if tc.executeErr != nil {
					return nil, tc.executeErr
				}
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Header: http.Header{"Retry-After": []string{"8"}}, Body: body, Request: r}, nil
			})}
			cmd := newCmdCheck(&cmdutil.Factory{HttpClient: func() (*http.Client, error) { return injected, nil }})
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			cmd.SetContext(ctx)
			cmd.SetArgs([]string{"MIT + BSD"})
			err := cmd.Execute()
			if calls != 1 || out.String() != tc.want || injected.Timeout != 0 {
				t.Fatalf("calls=%d out=%q timeout=%v err=%v", calls, out.String(), injected.Timeout, err)
			}
			switch {
			case tc.executeErr != nil:
				if _, ok := errors.AsType[*api.RequestExecutionError](err); !ok || !errors.Is(err, tc.executeErr) {
					t.Fatalf("execution cause lost: %v", err)
				}
			case tc.status >= 400:
				failure, ok := errors.AsType[*api.HTTPError](err)
				if !ok || failure.StatusCode != tc.status || failure.RetryAfter != "8" || !errors.Is(err, tc.readErr) {
					t.Fatalf("status/read cause lost: %v", err)
				}
			case tc.readErr != nil:
				if _, ok := errors.AsType[*api.ResponseReadError](err); !ok || !errors.Is(err, tc.readErr) {
					t.Fatalf("read cause lost: %v", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
			if tc.executeErr == nil && !body.closed {
				t.Fatal("response body not closed")
			}
		})
	}
}

func TestCheckPreparationAndInjectedTimeout(t *testing.T) {
	cause := errors.New("client failed")
	for _, mode := range []string{"client failure", "nil client", "custom timeout"} {
		t.Run(mode, func(t *testing.T) {
			client := &http.Client{Timeout: 2 * time.Minute, Transport: checkTransport(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) < time.Minute {
					t.Fatal("injected timeout was overwritten")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("compliant"))}, nil
			})}
			cmd := newCmdCheck(&cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				switch mode {
				case "client failure":
					return nil, cause
				case "nil client":
					return nil, nil
				default:
					return client, nil
				}
			}})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"MIT"})
			err := cmd.Execute()
			if mode == "custom timeout" {
				if err != nil || client.Timeout != 2*time.Minute {
					t.Fatalf("err=%v timeout=%v", err, client.Timeout)
				}
			} else if !errors.Is(err, api.ErrRequestPreparation) || mode == "client failure" && !errors.Is(err, cause) {
				t.Fatalf("preparation cause lost: %v", err)
			}
		})
	}
}
