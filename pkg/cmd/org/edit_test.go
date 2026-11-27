package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeOrgEdit(t *testing.T, f *cmdutil.Factory, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewCmdOrg(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(append([]string{"edit"}, args...))
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestOrgEditExplicitFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want map[string]string
	}{
		{"one field", []string{"--name", "Team Display Name"}, map[string]string{"name": "Team Display Name"}},
		{"description shorthand", []string{"-d", "line1\nline2"}, map[string]string{"description": "line1\nline2"}},
		{"clear optional fields", []string{"--description=", "--email=", "--location=", "--website="}, map[string]string{"description": "", "email": "", "location": "", "html_url": ""}},
		{"all fields", []string{"--name", "Team", "--description", "Projects", "--email", "team@example.com", "--location", "Wuhan", "--website", "www.example.com"}, map[string]string{"name": "Team", "description": "Projects", "email": "team@example.com", "location": "Wuhan", "html_url": "www.example.com"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "PATCH" || r.URL.EscapedPath() != "/api/v5/orgs/team%3Fx" || r.URL.RawQuery != "" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				var got map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if len(got) != len(tc.want) {
					t.Fatalf("body=%v want=%v", got, tc.want)
				}
				for key, value := range tc.want {
					encoded, err := json.Marshal(value)
					if err != nil {
						t.Fatal(err)
					}
					if string(got[key]) != string(encoded) {
						t.Fatalf("body[%q]=%s want=%s", key, got[key], encoded)
					}
				}
				return orgResponse(200, `{"id":42,"path":"team?x"}`), nil
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("inferred a Git repository for org edit")
				return cmdutil.Repository{}, nil
			}
			args := append([]string{"team?x", "--yes", "--json"}, tc.args...)
			out, stderr, err := executeOrgEdit(t, f, "", args...)
			if err != nil || stderr != "" || calls != 1 {
				t.Fatalf("error=%v stderr=%q calls=%d", err, stderr, calls)
			}
			var result struct {
				Organization struct {
					ID   int64
					Path string
				}
				Submitted    map[string]string `json:"submitted_fields"`
				Verification string
			}
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err)
			}
			if result.Organization.ID != 42 || result.Organization.Path != "team?x" || result.Verification != "identity_only" || !reflect.DeepEqual(result.Submitted, tc.want) {
				t.Fatalf("output=%s", out)
			}
		})
	}
}

func TestOrgEditConfirmation(t *testing.T) {
	for _, input := range []string{"yes\n", "n\n", ""} {
		t.Run(input, func(t *testing.T) {
			calls := 0
			tokenCalls := 0
			f := orgFactory(orgTestConfig{tokenCalls: &tokenCalls}, func(*http.Request) (*http.Response, error) {
				calls++
				return orgResponse(200, `{"id":42,"path":"team","html_url":"https://external.test"}`), nil
			})
			out, stderr, err := executeOrgEdit(t, f, input, "team", "--description=", "--location", "Wuhan")
			if err != nil {
				t.Fatal(err)
			}
			for _, part := range []string{"Organization: team", `"description": ""`, `"location": "Wuhan"`, "[y/N]"} {
				if !strings.Contains(stderr, part) {
					t.Fatalf("missing %q in confirmation %q", part, stderr)
				}
			}
			if input == "yes\n" {
				if calls != 1 || !strings.Contains(out, "Updated organization team (ID: 42)") || !strings.Contains(out, "not independently verified") || strings.Contains(out, "external.test") {
					t.Fatalf("calls=%d output=%q", calls, out)
				}
			} else if calls != 0 || tokenCalls != 0 || out != "" || !strings.Contains(stderr, "cancelled") {
				t.Fatalf("cancelled calls=%d credential reads=%d output=%q", calls, tokenCalls, out)
			}
		})
	}
}

func TestOrgEditValidationBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{nil, {"team", "extra", "--name", "New"}, {"", "--name", "New"}, {"a/b", "--name", "New"}, {"team"}, {"team", "--yes"}, {"team", "--name="}, {"team", "--name", " \t"}, {"team", "--path", "new"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			tokenCalls := 0
			f := orgFactory(orgTestConfig{tokenCalls: &tokenCalls}, func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid parameters sent a request")
				return nil, nil
			})
			out, stderr, err := executeOrgEdit(t, f, "yes\n", args...)
			if err == nil || tokenCalls != 0 || out != "" || stderr != "" {
				t.Fatalf("error=%v tokenCalls=%d stdout=%q stderr=%q", err, tokenCalls, out, stderr)
			}
		})
	}
}

func TestOrgEditAuthenticationFailure(t *testing.T) {
	cause := config.ErrNotAuthenticated
	f := orgFactory(orgTestConfig{tokenErr: cause}, func(*http.Request) (*http.Response, error) { t.Fatal("unauthenticated request sent"); return nil, nil })
	out, _, err := executeOrgEdit(t, f, "", "team", "--location", "Wuhan", "--yes")
	if !errors.Is(err, cause) || out != "" {
		t.Fatalf("error=%v output=%q", err, out)
	}
}

func TestOrgEditFailuresDoNotReplay(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		cause     error
		uncertain bool
	}{
		{name: "forbidden", status: http.StatusForbidden},
		{name: "not found", status: http.StatusNotFound},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "server error", status: http.StatusBadGateway, uncertain: true},
		{name: "timeout", cause: context.DeadlineExceeded, uncertain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodPatch || r.URL.Path != "/api/v5/orgs/team" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				if tc.cause != nil {
					return nil, tc.cause
				}
				response := orgResponse(tc.status, `{}`)
				response.Header.Set("Retry-After", "0")
				return response, nil
			})
			out, _, err := executeOrgEdit(t, f, "", "team", "--description", "New", "--yes")
			if err == nil || calls != 1 || out != "" {
				t.Fatalf("error=%v calls=%d stdout=%q", err, calls, out)
			}
			var uncertain *api.OrganizationEditUncertainError
			if errors.As(err, &uncertain) != tc.uncertain {
				t.Fatalf("unexpected uncertainty classification: %v", err)
			}
			if tc.cause != nil {
				if !errors.Is(err, tc.cause) {
					t.Fatalf("lost error cause: %v", err)
				}
			} else if !api.IsHTTPStatus(err, tc.status) {
				t.Fatalf("lost HTTP status: %v", err)
			}
		})
	}
}

func TestOrgEditClientFailureAndContext(t *testing.T) {
	cause := errors.New("client unavailable")
	f := orgFactory(orgTestConfig{}, nil)
	f.HTTPClient = func() (*http.Client, error) { return nil, cause }
	out, _, err := executeOrgEdit(t, f, "", "team", "--description", "New", "--yes")
	if !errors.Is(err, cause) || out != "" {
		t.Fatalf("error=%v output=%q", err, out)
	}
	f = orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() != context.Canceled {
			t.Fatal("command context was not propagated")
		}
		return nil, context.Canceled
	})
	cmd := newCmdOrgEdit(f)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("description", "New")
	_ = cmd.Flags().Set("yes", "true")
	if err := cmd.RunE(cmd, []string{"team"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
