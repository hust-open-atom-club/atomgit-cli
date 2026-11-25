package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeModuleEdit(t *testing.T, f *cmdutil.Factory, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoModule(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(input))
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(append([]string{"edit"}, args...))
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestModuleEditExplicitBooleans(t *testing.T) {
	for _, flag := range []struct{ flag, key string }{{"wiki", "has_wiki"}, {"issue", "has_issue"}, {"security", "has_security"}, {"merge-request", "has_merge_request"}, {"fork", "has_fork"}, {"analysis", "has_analysis"}, {"discussion", "has_discussion"}} {
		for _, enabled := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s=%t", flag.flag, enabled), func(t *testing.T) {
				calls := 0
				f := policyFactory(t, func(r *http.Request) (int, string) {
					calls++
					if r.Method != "PUT" || r.URL.Path != "/api/v5/repos/team/demo/module/setting" {
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
					}
					var body map[string]bool
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(body, map[string]bool{flag.key: enabled}) {
						t.Fatalf("body=%v", body)
					}
					return 200, `{"code":1,"msg":"success"}`
				})
				f.RepositoryResolver = func() (cmdutil.Repository, error) {
					t.Fatal("explicit repository not honored")
					return cmdutil.Repository{}, nil
				}
				out, stderr, err := executeModuleEdit(t, f, "", "team/demo", "--"+flag.flag+"="+fmt.Sprint(enabled), "--yes", "--json")
				if err != nil || stderr != "" || calls != 1 {
					t.Fatalf("err=%v stderr=%q calls=%d", err, stderr, calls)
				}
				var result struct {
					Repository   string
					Fields       map[string]bool `json:"submitted_fields"`
					Verification string
				}
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatal(err)
				}
				if result.Repository != "team/demo" || result.Verification != "api_acknowledged" || !reflect.DeepEqual(result.Fields, map[string]bool{flag.key: enabled}) {
					t.Fatalf("output=%s", out)
				}
			})
		}
	}
}

func TestModuleEditConfirmationAndInference(t *testing.T) {
	for _, input := range []string{"YES\n", "n\n", ""} {
		t.Run(input, func(t *testing.T) {
			cfg := &repoRecordingConfig{}
			calls := 0
			f := repoFactory(repoCommandConfig{}, forkRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var body map[string]bool
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, map[string]bool{"has_wiki": true, "has_issue": false}) {
					t.Fatalf("body=%v", body)
				}
				return forkResponse(200, `{"code":1,"msg":"success"}`), nil
			}))
			f.Config = cfg
			f.RepositoryResolver = func() (cmdutil.Repository, error) { return cmdutil.Repository{Owner: "team", Name: "demo"}, nil }
			out, stderr, err := executeModuleEdit(t, f, input, "--wiki", "--issue=false")
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Repository: team/demo", `"has_wiki": true`, `"has_issue": false`, "[y/N]"} {
				if !strings.Contains(stderr, want) {
					t.Fatalf("missing %q: %q", want, stderr)
				}
			}
			if input == "YES\n" {
				if calls != 1 || !strings.Contains(out, "API acknowledged") || !strings.Contains(out, "not independently verified") {
					t.Fatalf("calls=%d out=%q", calls, out)
				}
			} else if cfg.getTokenCalls != 0 || calls != 0 || out != "" {
				t.Fatalf("credentials=%d requests=%d out=%q", cfg.getTokenCalls, calls, out)
			}
		})
	}
}

func TestModuleEditValidationAndAuth(t *testing.T) {
	for _, args := range [][]string{nil, {"team/demo"}, {"--yes"}, {"team/demo", "--json"}, {"team/demo", "--wiki=bad"}, {"team/demo", "extra", "--wiki"}, {"bad", "--wiki"}} {
		cfg := &repoRecordingConfig{}
		f := &cmdutil.Factory{Config: cfg, HttpClient: func() (*http.Client, error) { t.Fatal("invalid arguments created HTTP client"); return nil, nil }}
		out, _, err := executeModuleEdit(t, f, "yes\n", args...)
		if err == nil || cfg.getTokenCalls != 0 || out != "" {
			t.Fatalf("args=%v err=%v tokenCalls=%d out=%q", args, err, cfg.getTokenCalls, out)
		}
	}
	f := repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unauthenticated write"); return nil, nil }))
	out, _, err := executeModuleEdit(t, f, "", "team/demo", "--wiki=false", "--yes")
	if !errors.Is(err, config.ErrNotAuthenticated) || out != "" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestModuleEditErrorsAndContext(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{403, `{}`}, {404, `{}`}, {502, `{}`}, {200, `{"code":0,"msg":"private-message"}`}, {200, `{"code":1}`}} {
		calls := 0
		f := policyFactory(t, func(*http.Request) (int, string) { calls++; return tc.status, tc.body })
		out, _, err := executeModuleEdit(t, f, "", "team/demo", "--wiki=false", "--yes")
		if err == nil || calls != 1 || out != "" || strings.Contains(err.Error(), "private-message") {
			t.Fatalf("calls=%d out=%q err=%v", calls, out, err)
		}
	}
	f := repoFactory(repoCommandConfig{token: "fixture"}, forkRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() != context.Canceled {
			t.Fatal("lost cancellation")
		}
		return nil, context.Canceled
	}))
	cmd := newCmdRepoModuleEdit(f)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cmd.SetContext(ctx)
	if err := cmd.Flags().Set("wiki", "false"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"team/demo"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
