package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func statusCredentials(t *testing.T, data string) string {
	t.Helper()
	isolateAuthConfig(t)
	path, err := config.PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if data != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func executeStatus(ctx context.Context, f *cmdutil.Factory, args ...string) (string, string, error) {
	cmd := NewCmdAuth(f)
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"status"}, args...))
	err := cmd.ExecuteContext(ctx)
	return out.String(), stderr.String(), err
}

func TestStatusLocalAndReadOnly(t *testing.T) {
	for _, tc := range []struct {
		name, data, status string
		present            *bool
	}{
		{"missing", "", "missing", new(false)},
		{"short token", `{"access_token":"xyz789","user":"alice","refresh_token":"private-refresh"}`, "configured", new(true)},
		{"long token", `{"access_token":"1234567890abcdef","user":"alice"}`, "configured", new(true)},
		{"secret in account", `{"access_token":"xyz789","user":"private-refresh","refresh_token":"private-refresh"}`, "configured", new(true)},
		{"missing user", `{"access_token":"private-access"}`, "invalid", nil},
		{"empty token", `{"access_token":"","user":"alice"}`, "invalid", nil},
		{"whitespace token", `{"access_token":"   ","user":"alice"}`, "invalid", nil},
		{"broken", `{"private-access":`, "invalid", nil},
		{"invalid active", `{"version":2,"active":"private-access","accounts":[{"user":"alice","access_token":"private-access"}]}`, "invalid", nil},
		{"multiple accounts", `{"version":2,"active":"bob","accounts":[{"user":"alice","access_token":"private-access"},{"user":"bob","access_token":"bob-secret","refresh_token":"private-refresh"}]}`, "configured", new(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := statusCredentials(t, tc.data)
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) { t.Fatal("offline check created client"); return nil, nil }}
			for _, asJSON := range []bool{false, true} {
				args := []string{}
				if asJSON {
					args = append(args, "--json")
				}
				out, stderr, err := executeStatus(context.Background(), f, args...)
				if (err == nil) != (tc.status == "configured") || stderr != "" {
					t.Fatalf("out=%s stderr=%s err=%v", out, stderr, err)
				}
				all := out + stderr
				if err != nil {
					all += err.Error()
				}
				for _, secret := range []string{"xyz789", "private-access", "private-refresh", "bob-secret", "1234", "cdef"} {
					if strings.Contains(all, secret) {
						t.Fatalf("leaked credential: %s", all)
					}
				}
				if asJSON {
					var r authStatusReport
					if err := json.Unmarshal([]byte(out), &r); err != nil {
						t.Fatal(err)
					}
					if r.LocalStatus != tc.status || !reflect.DeepEqual(r.CredentialsPresent, tc.present) || r.Verification.Status != "not_requested" || r.Verification.Performed || r.Verification.Requested {
						t.Fatalf("report=%+v", r)
					}
					if tc.name == "multiple accounts" && (r.LocalAccount == nil || *r.LocalAccount != "bob") {
						t.Fatal("wrong active account")
					}
				} else if tc.status == "configured" && !strings.Contains(out, "online validity has not been verified") {
					t.Fatal(out)
				}
			}
			if tc.data != "" {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != tc.data {
					t.Fatal("credentials modified")
				}
			} else if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("created credentials")
			}
			// Verification cannot run without valid local configuration.
			if tc.status != "configured" {
				out, _, err := executeStatus(context.Background(), f, "--verify", "--json")
				var r authStatusReport
				if json.Unmarshal([]byte(out), &r) != nil || err == nil || r.Verification.Status != "skipped" || r.Verification.Performed {
					t.Fatalf("%s %v", out, err)
				}
			}
		})
	}
}

func TestStatusOnlineResults(t *testing.T) {
	for _, tc := range []struct {
		name         string
		code         int
		body         string
		transportErr error
		status       string
	}{
		{"valid", 200, `{"login":"alice"}`, nil, "verified"},
		{"canonical casing", 200, `{"login":"Alice"}`, nil, "verified"},
		{"mismatch", 200, `{"login":"bob"}`, nil, "identity_mismatch"},
		{"missing login", 200, `{}`, nil, "invalid_response"},
		{"null", 200, `null`, nil, "invalid_response"},
		{"invalid type", 200, `{"login":123}`, nil, "invalid_response"},
		{"whitespace login", 200, `{"login":"  "}`, nil, "invalid_response"},
		{"control login", 200, `{"login":"alice\n"}`, nil, "invalid_response"},
		{"malformed", 200, `{"login":"alice"`, nil, "invalid_response"},
		{"trailing payload", 200, `{"login":"alice"} private-access`, nil, "invalid_response"},
		{"empty", 200, ``, nil, "invalid_response"},
		{"401", 401, `{"message":"xyz789 private-refresh"}`, nil, "unauthorized"},
		{"403", 403, `{"message":"xyz789 private-refresh"}`, nil, "forbidden"},
		{"500", 500, `{"message":"xyz789 private-refresh"}`, nil, "server_error"},
		{"429", 429, `{"message":"xyz789 private-refresh"}`, nil, "http_error"},
		{"redirect", 302, `{"login":"alice"}`, nil, "http_error"},
		{"unexpected success", 201, `{"login":"alice"}`, nil, "http_error"},
		{"network", 0, "", errors.New("xyz789 private-refresh"), "network_error"},
		{"timeout", 0, "", context.DeadlineExceeded, "timeout"},
		{"canceled", 0, "", context.Canceled, "canceled"},
		{"echo access", 200, `{"login":"xyz789"}`, nil, "identity_mismatch"},
		{"echo refresh", 200, `{"login":"private-refresh"}`, nil, "identity_mismatch"},
		{"oversized", 200, strings.Repeat(" ", 1<<20) + `{"login":"alice"}`, nil, "invalid_response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := `{"access_token":"xyz789","user":"alice","refresh_token":"private-refresh"}`
			path := statusCredentials(t, data)
			calls := 0
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: authRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != "GET" || req.URL.String() != "https://api.atomgit.com/api/v5/user" || req.Header.Get("Authorization") != "Bearer xyz789" {
						t.Fatalf("unexpected request %s %s", req.Method, req.URL)
					}
					deadline, ok := req.Context().Deadline()
					if !ok || time.Until(deadline) > 30*time.Second {
						t.Fatal("missing bounded deadline")
					}
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					return &http.Response{StatusCode: tc.code, Status: http.StatusText(tc.code), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
				})}, nil
			}}
			for _, asJSON := range []bool{false, true} {
				args := []string{"--verify"}
				if asJSON {
					args = append(args, "--json")
				}
				out, stderr, err := executeStatus(context.Background(), f, args...)
				if (err == nil) != (tc.status == "verified") {
					t.Fatalf("%s %v", out, err)
				}
				all := out + stderr
				if err != nil {
					all += err.Error()
				}
				for _, secret := range []string{"xyz789", "private-refresh", "private-access"} {
					if strings.Contains(all, secret) {
						t.Fatal("credential leak: " + all)
					}
				}
				if asJSON {
					var r authStatusReport
					if json.Unmarshal([]byte(out), &r) != nil || r.Verification.Status != tc.status || !r.Verification.Performed || !r.Verification.Requested {
						t.Fatalf("%s", out)
					}
					if tc.code >= 400 && (r.Verification.HTTPStatus == nil || *r.Verification.HTTPStatus != tc.code) {
						t.Fatal("missing HTTP status")
					}
				}
			}
			if calls == 0 {
				t.Fatal("verification skipped")
			}
			got, _ := os.ReadFile(path)
			if string(got) != data {
				t.Fatal("verification rewrote credentials")
			}
		})
	}
}

func TestStatusContextAndClientFailure(t *testing.T) {
	statusCredentials(t, `{"access_token":"xyz789","user":"alice"}`)
	for _, tc := range []struct {
		name   string
		ctx    func() (context.Context, context.CancelFunc)
		status string
	}{
		{"cancel", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, "canceled"},
		{"deadline", func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}, "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.ctx()
			defer cancel()
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: authRoundTripFunc(func(*http.Request) (*http.Response, error) {
					t.Fatal("request after canceled context")
					return nil, nil
				})}, nil
			}}
			r := inspectAuthStatus(ctx, f, true)
			if r.OK || r.Verification.Status != tc.status || r.Verification.Performed {
				t.Fatalf("%+v", r)
			}
		})
	}
	f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) { return nil, errors.New("xyz789") }}
	r := inspectAuthStatus(context.Background(), f, true)
	if r.OK || r.Verification.Status != "client_error" || r.Verification.Performed || strings.Contains(r.Message, "xyz789") {
		t.Fatalf("%+v", r)
	}
}

func TestStatusCancellationDuringRequest(t *testing.T) {
	statusCredentials(t, `{"access_token":"xyz789","user":"alice"}`)
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "timeout"}[timeout], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if timeout {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 20*time.Millisecond)
				defer stop()
			}
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: authRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if !timeout {
						cancel()
					}
					<-req.Context().Done()
					return nil, req.Context().Err()
				})}, nil
			}}
			r := inspectAuthStatus(ctx, f, true)
			want := "canceled"
			if timeout {
				want = "timeout"
			}
			if r.OK || r.Verification.Status != want {
				t.Fatalf("%+v", r)
			}
		})
	}
}

type failedStatusBody struct {
	read   bool
	closed bool
}

func (b *failedStatusBody) Read([]byte) (int, error) {
	b.read = true
	return 0, context.DeadlineExceeded
}

func (b *failedStatusBody) Close() error { b.closed = true; return nil }

func TestStatusHTTPFailureDoesNotDependOnErrorBody(t *testing.T) {
	statusCredentials(t, `{"access_token":"xyz789","user":"alice"}`)
	for _, tc := range []struct {
		code int
		want string
	}{{401, "unauthorized"}, {403, "forbidden"}, {503, "server_error"}} {
		t.Run(tc.want, func(t *testing.T) {
			body := &failedStatusBody{}
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: authRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: tc.code, Header: make(http.Header), Body: body, Request: req}, nil
				})}, nil
			}}
			r := inspectAuthStatus(context.Background(), f, true)
			if r.OK || r.Verification.Status != tc.want || r.Verification.HTTPStatus == nil || *r.Verification.HTTPStatus != tc.code {
				t.Fatalf("report = %+v", r)
			}
			if body.read || !body.closed {
				t.Fatalf("error body read=%v closed=%v", body.read, body.closed)
			}
		})
	}
}
