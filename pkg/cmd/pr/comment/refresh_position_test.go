package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runRefreshPosition(t *testing.T, f *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdComment(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(append([]string{"refresh-position"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestRefreshPositionScopesAndOutput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		want  map[string]any
	}{
		{"paths", []string{"--path", "src/a,b.go", "--path", "docs/a b.md"}, map[string]any{"refresh_by_paths": []any{"src/a,b.go", "docs/a b.md"}}},
		{"comment", []string{"--comment-id", "hex-id"}, map[string]any{"refresh_by_note_id": "hex-id"}},
		{"all", []string{"--all"}, map[string]any{}},
	} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: "/json", false: "/text"}[jsonOutput], func(t *testing.T) {
				calls := 0
				f := &cmdutil.Factory{Config: &prCommentRecordingConfig{}, RepositoryResolver: func() (cmdutil.Repository, error) {
					t.Fatal("explicit repository lost priority")
					return cmdutil.Repository{}, nil
				}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prCommentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Method != "POST" || r.URL.Path != "/api/v5/repos/team/repo/pulls/12/refresh_position" {
							t.Fatalf("request=%s %s", r.Method, r.URL)
						}
						var got map[string]any
						if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(got, tc.want) {
							t.Fatalf("body=%v", got)
						}
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"note_ids":[0,123]}`))}, nil
					})}, nil
				}}
				args := append([]string{"team/repo", "12"}, tc.flags...)
				if jsonOutput {
					args = append(args, "--json")
				}
				out, err := runRefreshPosition(t, f, args...)
				if err != nil || calls != 1 {
					t.Fatalf("err=%v calls=%d", err, calls)
				}
				if jsonOutput {
					if out != "{\n  \"note_ids\": [\n    0,\n    123\n  ]\n}\n" {
						t.Fatalf("output=%q", out)
					}
				} else if out != "Refreshed comment positions for PR #12 (2 comments).\n0\n123\n" {
					t.Fatalf("output=%q", out)
				}
			})
		}
	}
}

func TestRefreshPositionRejectsInvalidBeforeAuth(t *testing.T) {
	for _, args := range [][]string{nil, {"team/repo", "12"}, {"team/repo", "12", "--all=false"}, {"team/repo", "12", "--path="}, {"team/repo", "12", "--path", " \t"}, {"team/repo", "12", "--comment-id="}, {"team/repo", "12", "--comment-id", " \t"}, {"team/repo", "12", "--all", "--path", "a"}, {"team/repo", "12", "--all", "--comment-id", "1"}, {"team/repo", "12", "--path", "a", "--comment-id", "1"}, {"team/repo", "0", "--all"}, {"team/repo", "bad", "--all"}, {"team/repo", "12", "extra", "--all"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cfg := &prCommentRecordingConfig{}
			f := &cmdutil.Factory{Config: cfg, HttpClient: func() (*http.Client, error) { t.Fatal("created HTTP client"); return nil, nil }}
			out, err := runRefreshPosition(t, f, args...)
			if err == nil || out != "" || cfg.getTokenCalls != 0 {
				t.Fatalf("out=%q err=%v authCalls=%d", out, err, cfg.getTokenCalls)
			}
		})
	}
}

type refreshConfig struct{ err error }

func (c refreshConfig) GetToken() (string, error) { return "", c.err }
func (refreshConfig) GetUser() (string, error)    { return "alice", nil }
func (refreshConfig) GetHost() string             { return "atomgit.com" }

func TestRefreshPositionAuthenticationContextAndErrors(t *testing.T) {
	f := &cmdutil.Factory{Config: refreshConfig{config.ErrNotAuthenticated}}
	out, err := runRefreshPosition(t, f, "team/repo", "12", "--all")
	if !errors.Is(err, config.ErrNotAuthenticated) || out != "" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	for _, status := range []int{403, 404, 429, 500} {
		f = &cmdutil.Factory{Config: &prCommentRecordingConfig{}, HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prCommentRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})}, nil
		}}
		out, err := runRefreshPosition(t, f, "team/repo", "12", "--all")
		if err == nil || out != "" {
			t.Fatalf("status=%d out=%q err=%v", status, out, err)
		}
	}
	f = &cmdutil.Factory{Config: &prCommentRecordingConfig{}, RepositoryResolver: func() (cmdutil.Repository, error) { return cmdutil.Repository{Owner: "team", Name: "repo"}, nil }, HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: prCommentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Context().Err() != context.Canceled {
				t.Fatal("lost command context")
			}
			return nil, context.Canceled
		})}, nil
	}}
	cmd := newCmdRefreshPosition(f)
	_ = cmd.Flags().Set("all", "true")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cmd.SetContext(ctx)
	if err := cmd.RunE(cmd, []string{"12"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
