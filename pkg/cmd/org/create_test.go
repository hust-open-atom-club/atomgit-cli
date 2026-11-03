package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestOrgCreateRequest(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want map[string]string
	}{
		{"defaults omitted", []string{"team"}, map[string]string{"path": "team"}},
		{"custom", []string{"team", "--name", "My Team", "-d", "Team projects", "--visibility", "private"}, map[string]string{"path": "team", "name": "My Team", "description": "Team projects", "visibility": "private"}},
		{"explicit defaults and empty description", []string{"team", "--name", "team", "--description=", "--visibility", "public"}, map[string]string{"path": "team", "name": "team", "description": "", "visibility": "public"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/orgs" || req.URL.RawQuery != "" {
					t.Fatalf("request = %s %s", req.Method, req.URL)
				}
				if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Authorization") != "Bearer token" {
					t.Fatalf("headers = %v", req.Header)
				}
				var got map[string]string
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("body = %#v, want %#v", got, tt.want)
				}
				return orgResponse(200, `{"id":42,"path":"team","web_url":"https://example.test/groups/team","html_url":"https://wrong.test"}`), nil
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("organization creation inferred a repository")
				return cmdutil.Repository{}, nil
			}
			cmd := NewCmdOrg(f)
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs(append([]string{"create"}, tt.args...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || output.String() != "Created organization team: https://example.test/groups/team\n" {
				t.Fatalf("calls=%d output=%q", calls, output.String())
			}
		})
	}
}

func TestOrgCreateClientFailure(t *testing.T) {
	cause := errors.New("HTTP client unavailable")
	f := orgFactory(orgTestConfig{}, nil)
	f.HttpClient = func() (*http.Client, error) { return nil, cause }
	cmd := newCmdOrgCreate(f)
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	cmd.SetArgs([]string{"team"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); !errors.Is(err, cause) || out.Len() != 0 {
		t.Fatalf("error=%v output=%q", err, out.String())
	}
}

func TestOrgCreateValidationBeforeAuthentication(t *testing.T) {
	for _, args := range [][]string{
		nil, {"team", "extra"}, {""}, {"a/b"}, {"a b"}, {"."}, {".."}, {"a\nb"},
		{"team", "--name="}, {"team", "--name", "  "},
		{"team", "--visibility="}, {"team", "--visibility", "internal"}, {"team", "--visibility", "Public"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			calls := 0
			cmd := newCmdOrgCreate(orgFactory(orgTestConfig{tokenCalls: &calls}, nil))
			cmd.SilenceErrors, cmd.SilenceUsage = true, true
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("invalid arguments accepted")
			}
			if calls != 0 {
				t.Fatal("validation accessed credentials")
			}
		})
	}
}

func TestOrgCreateAuthenticationAndCompletion(t *testing.T) {
	calls := 0
	cmd := newCmdOrgCreate(orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated, tokenCalls: &calls}, nil))
	complete, ok := cmd.GetFlagCompletionFunc("visibility")
	if !ok {
		t.Fatal("visibility completion missing")
	}
	for prefix, want := range map[string][]string{"": {"public", "private"}, "pri": {"private"}} {
		got, directive := complete(cmd, nil, prefix)
		if !reflect.DeepEqual(got, want) || directive != cobra.ShellCompDirectiveNoFileComp {
			t.Fatalf("completion(%q) = %v, %v", prefix, got, directive)
		}
	}
	if calls != 0 {
		t.Fatal("completion accessed credentials")
	}
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	cmd.SetArgs([]string{"team"})
	if err := cmd.Execute(); err == nil || !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("authentication error = %v", err)
	}
}

func TestOrgCreateResponse(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"no URL", 200, `{"id":1,"path":"team"}`, "Created organization team\n"},
		{"conflict", 409, `{"message":"path already exists"}`, ""},
		{"forbidden", 403, `{"message":"permission denied"}`, ""},
		{"unauthorized", 401, `{}`, ""},
		{"unexpected success", 201, `{"id":1,"path":"team"}`, ""},
		{"server error", 500, `{}`, ""},
		{"empty", 200, ``, ""},
		{"malformed", 200, `{"id":`, ""},
		{"missing identity", 200, `{}`, ""},
		{"mismatched identity", 200, `{"id":1,"path":"other"}`, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
				calls++
				return orgResponse(tt.status, tt.body), nil
			})
			cmd := newCmdOrgCreate(f)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SilenceErrors, cmd.SilenceUsage = true, true
			cmd.SetArgs([]string{"team"})
			err := cmd.Execute()
			if (err == nil) != (tt.want != "") || out.String() != tt.want || calls != 1 {
				t.Fatalf("error=%v output=%q calls=%d", err, out.String(), calls)
			}
		})
	}
}
