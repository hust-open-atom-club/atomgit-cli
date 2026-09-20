package doctor

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type transport func(*http.Request) (*http.Response, error)

func (fn transport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func credentials(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "ag-cli", "token.json")
	if body != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

const validCredentials = `{"version":2,"active":"alice","accounts":[{"user":"alice","access_token":"very-secret-token"}]}`

func execute(t *testing.T, f *cmdutil.Factory, args ...string) (report, error, string) {
	t.Helper()
	cmd := NewCmdDoctor(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(append(args, "--json"))
	err := cmd.Execute()
	var r report
	if out.Len() > 0 {
		if e := json.Unmarshal(out.Bytes(), &r); e != nil {
			t.Fatalf("invalid JSON %q: %v", out.String(), e)
		}
	}
	return r, err, out.String()
}
func row(t *testing.T, r report, id string) check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("missing %s: %+v", id, r)
	return check{}
}

func TestOfflineReadOnly(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"missing", "", "warn"},
		{"legacy", `{"user":"alice","access_token":"very-secret-token"}`, "pass"},
		{"current", validCredentials, "pass"},
		{"corrupt", `{"access_token":"very-secret-token",broken`, "fail"},
		{"version", `{"version":99,"access_token":"very-secret-token"}`, "fail"},
		{"active", `{"version":2,"active":"very-secret-token","accounts":[{"user":"alice","access_token":"very-secret-token"}]}`, "fail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := credentials(t, tc.body)
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) { t.Fatal("offline requested HTTP client"); return nil, nil }}
			r, err, out := execute(t, f)
			if got := row(t, r, "credentials").Status; got != tc.want {
				t.Fatalf("status %s", got)
			}
			if (err != nil) != (tc.want == "fail") {
				t.Fatalf("error %v", err)
			}
			if strings.Contains(out, "very-secret-token") {
				t.Fatal("credential leaked")
			}
			if tc.body != "" {
				b, e := os.ReadFile(path)
				if e != nil || string(b) != tc.body {
					t.Fatal("credentials changed")
				}
			} else {
				if _, e := os.Stat(path); !os.IsNotExist(e) {
					t.Fatal("credentials created")
				}
			}
			if tc.want == "pass" && row(t, r, "credential_expiry").Status != "skip" {
				t.Fatal("invented expiry")
			}
		})
	}
}

func TestInsecurePermissionsAreNotRepaired(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	path := credentials(t, validCredentials)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err, _ := execute(t, &cmdutil.Factory{})
	if err == nil || row(t, r, "credential_permissions").Status != "fail" {
		t.Fatal("unsafe permissions accepted")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o644 {
		t.Fatal("doctor changed permissions")
	}
}

func TestLiveReadOnlyAndRedaction(t *testing.T) {
	for _, status := range []int{200, 401, 403, 404, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			credentials(t, validCredentials)
			var paths []string
			f := &cmdutil.Factory{RepositoryResolver: func() (cmdutil.Repository, error) { t.Fatal("explicit repo ignored"); return cmdutil.Repository{}, nil }}
			f.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
					if req.Method != "GET" {
						t.Fatal("non-read request")
					}
					if _, ok := req.Context().Deadline(); !ok {
						t.Fatal("missing deadline")
					}
					paths = append(paths, req.URL.Path)
					body := `{"login":"alice"}`
					switch {
					case strings.HasSuffix(req.URL.Path, "/actions/workflows"):
						body = `{"total_count":0,"workflows":[]}`
					case strings.HasSuffix(req.URL.Path, "/discuss"):
						body = `[]`
					case strings.HasSuffix(req.URL.Path, "/repos/team/demo"):
						body = `{"full_name":"team/demo"}`
					}
					code := 200
					if strings.HasSuffix(req.URL.Path, "/actions/workflows") {
						code = status
						if status != 200 {
							body = `{"message":"very-secret-token https://user:password@proxy.invalid/?access_token=secret"}`
						}
					}
					return &http.Response{StatusCode: code, Status: http.StatusText(code), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
				})}, nil
			}
			r, err, out := execute(t, f, "team/demo", "--live")
			if (err != nil) != (status != 200) {
				t.Fatalf("error %v; %s", err, out)
			}
			if len(paths) < 4 {
				t.Fatalf("requests: %v", paths)
			}
			if row(t, r, "connectivity").Status != "pass" || row(t, r, "authentication").Status != "pass" {
				t.Fatal(out)
			}
			if strings.Contains(out, "very-secret-token") || strings.Contains(out, "password") || strings.Contains(out, "proxy.invalid") {
				t.Fatal("error leaked secrets")
			}
			if status == 403 && !strings.Contains(row(t, r, "actions").Message, "not established") {
				t.Fatal("403 cause overstated")
			}
		})
	}
}

func TestRepositoryCanonicalPath(t *testing.T) {
	for _, inferred := range []bool{false, true} {
		for _, tc := range []struct {
			name, fullName string
			valid          bool
		}{
			{"canonical owner", "team/demo", true},
			{"same owner", "TEAM/demo", true},
			{"different owner", "other/demo", false},
			{"different repository", "team/other", false},
			{"repository case differs", "team/DEMO", true},
			{"empty response", "", false},
		} {
			t.Run(fmt.Sprintf("inferred=%t/%s", inferred, tc.name), func(t *testing.T) {
				credentials(t, validCredentials)
				f := &cmdutil.Factory{RepositoryResolver: func() (cmdutil.Repository, error) {
					return cmdutil.Repository{Owner: "TEAM", Name: "demo"}, nil
				}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
						body := `{"login":"alice"}`
						switch {
						case strings.HasSuffix(req.URL.Path, "/actions/workflows"):
							body = `{"total_count":0,"workflows":[]}`
						case strings.HasSuffix(req.URL.Path, "/discuss"):
							body = `[]`
						case strings.HasSuffix(req.URL.Path, "/repos/TEAM/demo"):
							b, _ := json.Marshal(map[string]string{"full_name": tc.fullName})
							body = string(b)
						}
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
					})}, nil
				}}
				args := []string{"--live"}
				if !inferred {
					args = append(args, "TEAM/demo")
				}
				r, err, out := execute(t, f, args...)
				if (err == nil) != tc.valid || r.OK != tc.valid || (row(t, r, "repository_access").Status == "pass") != tc.valid {
					t.Fatalf("valid=%t error=%v: %s", tc.valid, err, out)
				}
			})
		}
	}
}

func TestRejectedCredentialsSuggestForcedLogin(t *testing.T) {
	credentials(t, validCredentials)
	f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"message":"Unauthorized"}`)), Request: req}, nil
		})}, nil
	}}
	r, err, out := execute(t, f, "team/demo", "--live")
	if err == nil || r.OK {
		t.Fatalf("rejected credentials accepted: %s", out)
	}
	for _, id := range []string{"authentication", "repository_access", "actions", "discussion"} {
		c := row(t, r, id)
		if c.Status != "fail" || !strings.Contains(c.Hint, "ag auth login --force") {
			t.Fatalf("%s must suggest forced reauthentication: %+v", id, c)
		}
	}
}

func TestCancellationAndTransportErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		message string
	}{
		{"dns", &net.DNSError{Err: "very-secret-token", Name: "private.invalid"}, "DNS lookup failed"},
		{"canceled", context.Canceled, "Probe canceled"},
		{"timeout", context.DeadlineExceeded, "Probe timed out"},
		{"tls", &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}, "TLS certificate verification failed"},
		{"unknown", errors.New("very-secret-token"), "Network connection failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credentials(t, validCredentials)
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, tc.err })}, nil
			}}
			r, err, out := execute(t, f, "--live")
			if err == nil || row(t, r, "connectivity").Message != tc.message || strings.Contains(out, "very-secret-token") {
				t.Fatalf("%v %s", err, out)
			}
		})
	}
	t.Run("factory context", func(t *testing.T) {
		credentials(t, validCredentials)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &cmdutil.Factory{Context: func() context.Context { return ctx }, HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				if r.Context().Err() != context.Canceled {
					t.Fatal("context dropped")
				}
				return nil, r.Context().Err()
			})}, nil
		}}
		r, err, _ := execute(t, f, "--live")
		if err == nil || row(t, r, "connectivity").Message != "Probe canceled" {
			t.Fatal(r)
		}
	})
}

func TestLiveWithoutCredentialsStillChecksConnectivity(t *testing.T) {
	credentials(t, "")
	calls := 0
	f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("Authorization") != "" {
				t.Fatal("anonymous probe sent authorization")
			}
			return &http.Response{StatusCode: 401, Status: "Unauthorized", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"message":"not logged in"}`)), Request: req}, nil
		})}, nil
	}}
	r, err, out := execute(t, f, "team/demo", "--live")
	if err != nil || calls != 1 || row(t, r, "connectivity").Status != "pass" || row(t, r, "authentication").Status != "skip" {
		t.Fatalf("%v %s", err, out)
	}
}

func TestKnownExpiryAndInvalidUserResponse(t *testing.T) {
	credentials(t, `{"user":"alice","access_token":"synthetic","created_at":1,"expires_in":1}`)
	f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
		})}, nil
	}}
	r, err, _ := execute(t, f, "--live")
	if err == nil || row(t, r, "credential_expiry").Status != "warn" || row(t, r, "authentication").Status != "fail" {
		t.Fatal(r)
	}
}

func TestInvalidArgumentsBeforeAccess(t *testing.T) {
	_, err, out := execute(t, &cmdutil.Factory{}, "https://secret@example.com/repo")
	if err == nil || out != "" || strings.Contains(err.Error(), "secret") {
		t.Fatalf("%v %s", err, out)
	}
}

func TestConfigFailureDoesNotHideOtherChecks(t *testing.T) {
	path := credentials(t, validCredentials)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "config.json"), []byte(`{"aliases":"very-secret-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err, out := execute(t, &cmdutil.Factory{})
	if err == nil || row(t, r, "config").Status != "fail" || row(t, r, "credentials").Status != "pass" || strings.Contains(out, "very-secret-token") {
		t.Fatal(out)
	}
}

func TestAnonymousServiceFailures(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			credentials(t, "")
			f := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"message":"synthetic-private-response"}`)), Request: req}, nil
				})}, nil
			}}
			r, err, out := execute(t, f, "--live")
			failed := status != 401 && status != 403
			if (err != nil) != failed || r.OK == failed {
				t.Fatalf("status %d: err=%v report=%s", status, err, out)
			}
			if row(t, r, "connectivity").Status != "pass" {
				t.Fatal("HTTP response proves connectivity")
			}
			if row(t, r, "authentication").Status != "skip" {
				t.Fatal("anonymous authentication should be skipped")
			}
			want := "skip"
			if failed {
				want = "fail"
			}
			if row(t, r, "service").Status != want {
				t.Fatal(out)
			}
			if strings.Contains(out, "synthetic-private-response") {
				t.Fatal("response leaked")
			}
		})
	}
}
