package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runFollowing(t *testing.T, f *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdUser(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(append([]string{"following"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestUserFollowingMixedOutputAndAuthentication(t *testing.T) {
	for _, anon := range []bool{false, true} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{true: "anonymous", false: "authenticated"}[anon], map[bool]string{true: "json", false: "text"}[jsonOutput]}, "/"), func(t *testing.T) {
				cfg := userTestConfig{}
				if anon {
					cfg.tokenErr = config.ErrNotAuthenticated
				}
				calls := 0
				f := userFactory(cfg, func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "GET" || r.URL.EscapedPath() != "/api/v5/users/alice%252Fadmin/following" || r.URL.RawQuery != "page=1&per_page=30" {
						t.Fatalf("request=%s %s", r.Method, r.URL)
					}
					want := "Bearer token"
					if anon {
						want = ""
					}
					if r.Header.Get("Authorization") != want {
						t.Fatalf("authorization mismatch")
					}
					return userResponse(200, `[{"id":"9007199254740993","object_id":"hex","login":"bob","name":"B\tO\nB","type":"User","following_url":"/user/bob/following"},{"id":"org","login":"team","name":null,"type":"Organization"}]`), nil
				})
				f.RepositoryResolver = func() (cmdutil.Repository, error) { t.Fatal("inferred Git context"); return cmdutil.Repository{}, nil }
				args := []string{"alice%2Fadmin"}
				if jsonOutput {
					args = append(args, "--json")
				}
				out, err := runFollowing(t, f, args...)
				if err != nil || calls != 1 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
				if jsonOutput {
					var items []map[string]any
					if err := json.Unmarshal([]byte(out), &items); err != nil {
						t.Fatal(err)
					}
					if len(items) != 2 || len(items[0]) != 10 || len(items[1]) != 10 || items[0]["id"] != "9007199254740993" || items[1]["type"] != "Organization" || items[1]["following_url"] != "" || items[1]["html_url"] != "" {
						t.Fatalf("output=%s", out)
					}
				} else {
					for _, want := range []string{"LOGIN", "NAME", "TYPE", "URL", "User", "Organization", `B\tO\nB`} {
						if !strings.Contains(out, want) {
							t.Fatalf("missing %q output=%q", want, out)
						}
					}
				}
			})
		}
	}
}

func TestUserFollowingValidatesBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{nil, {"alice", "bob"}, {""}, {"a/b"}, {"a?b"}, {"a#b"}, {"a\nb"}, {"alice", "--limit=0"}, {"alice", "--limit=-1"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cfg := &namespaceCountingConfig{}
			f := &cmdutil.Factory{Config: cfg, HttpClient: func() (*http.Client, error) { t.Fatal("created HTTP client"); return nil, nil }}
			out, err := runFollowing(t, f, args...)
			if err == nil || out != "" || cfg.tokenCalls != 0 {
				t.Fatalf("out=%q err=%v tokenCalls=%d", out, err, cfg.tokenCalls)
			}
		})
	}
}

func TestUserFollowingCredentialFailureAndEmpty(t *testing.T) {
	cause := errors.New("credential store damaged")
	f := userFactory(userTestConfig{tokenErr: cause}, func(*http.Request) (*http.Response, error) { t.Fatal("ignored credential error"); return nil, nil })
	out, err := runFollowing(t, f, "alice")
	if !errors.Is(err, cause) || out != "" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	for _, jsonOutput := range []bool{false, true} {
		f = userFactory(userTestConfig{}, func(*http.Request) (*http.Response, error) { return userResponse(200, `[]`), nil })
		args := []string{"alice"}
		want := "No followed users or organizations found.\n"
		if jsonOutput {
			args = append(args, "--json")
			want = "[]\n"
		}
		out, err := runFollowing(t, f, args...)
		if err != nil || out != want {
			t.Fatalf("out=%q err=%v", out, err)
		}
	}
}

func TestUserFollowingContextAndNoPartialOutput(t *testing.T) {
	calls := 0
	f := userFactory(userTestConfig{}, func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 2 {
			return userResponse(404, `{"message":"hidden"}`), nil
		}
		items := make([]map[string]string, 100)
		for i := range items {
			items[i] = map[string]string{"type": "User"}
		}
		b, _ := json.Marshal(items)
		return userResponse(200, string(b)), nil
	})
	out, err := runFollowing(t, f, "alice", "--limit=101")
	if err == nil || out != "" || calls != 2 {
		t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
	}
	f = userFactory(userTestConfig{}, func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() != context.Canceled {
			t.Fatal("lost context")
		}
		return nil, context.Canceled
	})
	cmd := newCmdUserFollowing(f)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cmd.SetContext(ctx)
	if err := cmd.RunE(cmd, []string{"alice"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
