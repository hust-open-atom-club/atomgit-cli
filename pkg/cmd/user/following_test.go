package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
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
					return userResponse(200, `[{"id":"9007199254740993","object_id":"hex","login":"bob","name":"B\tO\nB","type":"User","avatar_url":"https://example.test/avatar/bob","followers_url":"/user/bob/followers","html_url":"https://example.test/bob","url":"https://example.test/api/users/bob","following_url":"/user/bob/following"},{"id":"org","login":"team","name":null,"type":"Organization","url":"https://example.test/api/orgs/team"}]`), nil
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
					want := []map[string]any{
						{"id": "9007199254740993", "object_id": "hex", "login": "bob", "name": "B\tO\nB", "type": "User",
							"html_url": "https://example.test/bob", "url": "https://example.test/api/users/bob",
							"avatar_url": "https://example.test/avatar/bob", "followers_url": "/user/bob/followers", "following_url": "/user/bob/following"},
						{"id": "org", "object_id": "", "login": "team", "name": "", "type": "Organization", "html_url": "",
							"url": "https://example.test/api/orgs/team", "avatar_url": "", "followers_url": "", "following_url": ""},
					}
					if !reflect.DeepEqual(items, want) {
						t.Fatalf("output = %s, want %+v", out, want)
					}
				} else {
					lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
					if len(lines) != 3 {
						t.Fatalf("unexpected output rows: %q", out)
					}
					wantUser := []string{"bob", `B\tO\nB`, "User", "https://example.test/bob"}
					if !reflect.DeepEqual(strings.Fields(lines[1]), wantUser) {
						t.Fatalf("user row = %q, want %v", lines[1], wantUser)
					}
					if !reflect.DeepEqual(strings.Fields(lines[2]), []string{"team", "Organization"}) {
						t.Fatalf("organization row should have no synthesized URL: %q", lines[2])
					}
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
	for _, args := range [][]string{nil, {"alice", "bob"}, {""}, {"a/b"}, {"a?b"}, {"a#b"}, {"a\nb"}, {"a\\b"}, {"."}, {".."}, {"a\u00a0b"}, {"a\u0085b"}, {"a\u009fb"}, {"alice", "--limit=0"}, {"alice", "--limit=-1"}} {
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
		if r.URL.EscapedPath() != "/api/v5/users/alice/following" || r.URL.RawQuery != fmt.Sprintf("page=%d&per_page=100", calls) {
			t.Fatalf("unexpected request: %s", r.URL)
		}
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

func TestUserFollowingPaginationAndLimit(t *testing.T) {
	for _, limit := range []int{1, 101} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("limit=%d/json=%t", limit, asJSON), func(t *testing.T) {
				calls := 0
				f := userFactory(userTestConfig{}, func(r *http.Request) (*http.Response, error) {
					calls++
					if calls > (limit+99)/100 {
						t.Fatal("requested another page after reaching limit")
					}
					if r.URL.EscapedPath() != "/api/v5/users/alice/following" || r.URL.RawQuery != fmt.Sprintf("page=%d&per_page=%d", calls, min(limit, 100)) {
						t.Fatalf("unexpected request: %s", r.URL)
					}
					// Return a full page even when fewer entries remain in the output budget.
					items := make([]map[string]string, 100)
					for i := range items {
						items[i] = map[string]string{"id": fmt.Sprint((calls-1)*100 + i), "login": fmt.Sprintf("user-%d", (calls-1)*100+i), "type": "User"}
					}
					body, err := json.Marshal(items)
					if err != nil {
						t.Fatal(err)
					}
					return userResponse(200, string(body)), nil
				})
				args := []string{" alice ", "--limit", fmt.Sprint(limit)}
				if asJSON {
					args = append(args, "--json")
				}
				out, err := runFollowing(t, f, args...)
				if err != nil || calls != (limit+99)/100 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
				if asJSON {
					var items []map[string]any
					if err := json.Unmarshal([]byte(out), &items); err != nil {
						t.Fatal(err)
					}
					if len(items) != limit || items[0]["id"] != "0" || items[limit-1]["id"] != fmt.Sprint(limit-1) {
						t.Fatalf("unexpected output: %s", out)
					}
				} else {
					lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
					if len(lines) != limit+1 || strings.Fields(lines[limit])[0] != fmt.Sprintf("user-%d", limit-1) {
						t.Fatalf("unexpected output: %s", out)
					}
				}
			})
		}
	}
}
