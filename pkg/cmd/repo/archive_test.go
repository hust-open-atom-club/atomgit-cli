package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepoArchiveAndUnarchive(t *testing.T) {
	for _, archived := range []bool{true, false} {
		for _, explicit := range []bool{true, false} {
			t.Run(fmt.Sprintf("archived=%t/explicit=%t", archived, explicit), func(t *testing.T) {
				puts, reads := 0, 0
				f := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodGet {
						reads++
						if req.URL.Path != "/api/v5/repos/team/demo" {
							t.Fatalf("GET %s", req.URL)
						}
						return forkResponse(200, `{"id":7,"path":"demo","namespace":{"path":"team"}}`), nil
					}
					puts++
					if req.Method != http.MethodPut || req.URL.Path != "/api/v5/org/team/repo/demo/status" {
						t.Fatalf("write %s %s", req.Method, req.URL)
					}
					var body map[string]int
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					want := 0
					if archived {
						want = 2
					}
					if got, ok := body["status"]; !ok || got != want || len(body) != 1 {
						t.Fatalf("body=%v", body)
					}
					return forkResponse(200, `{"code":0,"msg":"success"}`), nil
				}))
				f.RepositoryResolver = func() (cmdutil.Repository, error) {
					if explicit {
						t.Fatal("explicit argument did not win")
					}
					return cmdutil.Repository{Owner: "team", Name: "demo"}, nil
				}
				cmd := newCmdRepoArchive(f, archived)
				args := []string{}
				if explicit {
					args = append(args, "team/demo", "--yes")
				} else {
					cmd.SetIn(strings.NewReader("yes\n"))
				}
				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(io.Discard)
				cmd.SetArgs(args)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				if puts != 1 || reads != 1 || !strings.Contains(out.String(), "API acknowledged") || !strings.Contains(out.String(), "team/demo") {
					t.Fatalf("puts=%d reads=%d out=%s", puts, reads, &out)
				}
			})
		}
	}
}

func TestRepoArchiveCancellationAndValidationSendNoRequests(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		input string
		fail  bool
	}{
		{"cancel", []string{"team/demo"}, "n\n", false}, {"EOF", []string{"team/demo"}, "", false},
		{"bad repo", []string{"a/b/c"}, "yes\n", true}, {"too many args", []string{"team/demo", "extra"}, "yes\n", true},
		{"stdin needs yes", []string{"team/demo", "--password-stdin"}, "secret", true},
		{"conflicting passwords", []string{"team/demo", "--yes", "--password-stdin", "--with-password"}, "secret", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("unexpected request")
				return nil, nil
			}))
			cmd := newCmdRepoArchive(factory, true)
			cmd.SilenceErrors = true
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); (err != nil) != tc.fail {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRepoArchiveRejectsPersonalUnknownAndWrongRepository(t *testing.T) {
	for _, body := range []string{
		`{"id":7,"path":"demo","owner":{"login":"team","type":"User"}}`,
		`{"id":7,"path":"demo"}`, `{"path":"demo","namespace":{"path":"team"}}`,
		`{"id":7,"path":"other","namespace":{"path":"team"}}`,
		`{"id":7,"path":"demo","namespace":{"path":"elsewhere"}}`,
		`{"id":7,"path":"demo","owner":{"login":"elsewhere","type":"Organization"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			puts := 0
			f := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					puts++
				}
				return forkResponse(200, body), nil
			}))
			cmd := newCmdRepoArchive(f, true)
			cmd.SilenceErrors = true
			cmd.SetOut(io.Discard)
			cmd.SetArgs([]string{"team/demo", "--yes"})
			if err := cmd.Execute(); err == nil || puts != 0 {
				t.Fatalf("puts=%d err=%v", puts, err)
			}
		})
	}
}

func TestRepoArchivePasswordInput(t *testing.T) {
	const password = " password with spaces "
	for _, tc := range []struct {
		name, input string
		flags       []string
		fail        bool
	}{
		{"stdin", password + "\r\n", []string{"--password-stdin"}, false},
		{"empty", "", []string{"--password-stdin"}, true},
		{"oversized", strings.Repeat("x", maxRepositoryPasswordBytes+1), []string{"--password-stdin"}, true},
		{"hidden input requires terminal", "secret", []string{"--with-password"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			puts := 0
			f := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return forkResponse(200, `{"id":7,"name":"demo","owner":{"login":"team","type":"Organization"}}`), nil
				}
				puts++
				var body struct {
					Password string `json:"password"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Password != password {
					t.Fatal("password was changed")
				}
				return forkResponse(200, `{"code":0,"msg":"success"}`), nil
			}))
			cmd := newCmdRepoArchive(f, true)
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SilenceErrors = true
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetArgs(append([]string{"team/demo", "--yes"}, tc.flags...))
			err := cmd.Execute()
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected error=%v", err)
			}
			want := 1
			if tc.fail {
				want = 0
			}
			if puts != want {
				t.Fatalf("puts=%d", puts)
			}
			if strings.Contains(out.String()+stderr.String(), password) {
				t.Fatal("password was displayed")
			}
		})
	}
}

func TestRepoArchiveFailureDoesNotReportSuccessOrPassword(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"permission", 403, `{"msg":"sensitive-password"}`}, {"missing", 404, `{}`},
		{"business failure", 200, `{"code":9,"msg":"sensitive-password"}`},
		{"missing code", 200, `{"msg":"sensitive-password"}`}, {"malformed", 200, `{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return forkResponse(200, `{"id":7,"name":"demo","namespace":{"path":"team"}}`), nil
				}
				return forkResponse(tc.status, tc.body), nil
			}))
			cmd := newCmdRepoArchive(f, true)
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SilenceErrors = true
			cmd.SetIn(strings.NewReader("sensitive-password\n"))
			cmd.SetArgs([]string{"team/demo", "--yes", "--password-stdin"})
			err := cmd.Execute()
			if err == nil || strings.Contains(out.String(), "API acknowledged") {
				t.Fatal("reported success")
			}
			if strings.Contains(out.String()+stderr.String()+err.Error(), "sensitive-password") {
				t.Fatal("password exposed")
			}
		})
	}
	cmd := newCmdRepoArchive(repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, nil), true)
	cmd.SilenceErrors = true
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "--yes"})
	if err := cmd.Execute(); !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("lost authentication cause: %v", err)
	}
}
