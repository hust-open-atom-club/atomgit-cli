package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

func TestUserFollowersOutput(t *testing.T) {
	for _, anonymous := range []bool{false, true} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("anonymous=%t/json=%t", anonymous, asJSON), func(t *testing.T) {
				cfg := userTestConfig{}
				if anonymous {
					cfg.tokenErr = config.ErrNotAuthenticated
				}
				f := userFactory(cfg, func(r *http.Request) (*http.Response, error) {
					if r.Method != "GET" || r.URL.Path != "/api/v5/users/alice/followers" || r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "100" {
						t.Fatalf("request = %s %s", r.Method, r.URL)
					}
					if (r.Header.Get("Authorization") == "") != anonymous {
						t.Fatal("unexpected authorization")
					}
					return userResponse(200, `[{"id":"0009007199254740993","login":"bob","name":"B\tC\nD\rE\\F","type":"User","html_url":"","avatar_url":null}]`), nil
				})
				cmd := newCmdUserFollowers(f)
				var out bytes.Buffer
				cmd.SetOut(&out)
				args := []string{"alice"}
				if asJSON {
					args = append(args, "--json")
				}
				cmd.SetArgs(args)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				if asJSON {
					var items []map[string]any
					if err := json.Unmarshal(out.Bytes(), &items); err != nil {
						t.Fatal(err)
					}
					if len(items) != 1 || len(items[0]) != 6 || items[0]["id"] != "0009007199254740993" || items[0]["avatarUrl"] != "" || items[0]["url"] != "" {
						t.Fatalf("output=%s", out.String())
					}
				} else if out.String() != "ID\tLOGIN\tNAME\tTYPE\tURL\n0009007199254740993\tbob\tB\\tC\\nD\\rE\\\\F\tUser\t\n" {
					t.Fatalf("output=%q", out.String())
				}
			})
		}
	}
}

func TestUserFollowersDefaultAndExplicitLimit(t *testing.T) {
	for _, tt := range []struct {
		name  string
		flags []string
		want  int
	}{
		{"default", nil, 30}, {"explicit", []string{"-L", "45"}, 45},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := userFactory(userTestConfig{}, func(*http.Request) (*http.Response, error) {
				return userResponse(200, "["+strings.Repeat(`{"id":"1"},`, 99)+`{"id":"100"}]`), nil
			})
			cmd := newCmdUserFollowers(f)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(append([]string{"alice", "--json"}, tt.flags...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var items []followerJSON
			if err := json.Unmarshal(out.Bytes(), &items); err != nil {
				t.Fatal(err)
			}
			if len(items) != tt.want {
				t.Fatalf("got %d followers, want %d", len(items), tt.want)
			}
		})
	}
}

func TestUserFollowersValidationAndCredentialErrors(t *testing.T) {
	cause := errors.New("credential read failed")
	for _, args := range [][]string{nil, {"alice", "bob"}, {""}, {"a/b"}, {"a?b"}, {"a#b"}, {"a\\b"}, {"."}, {".."}, {"a\nb"}, {"a\u00a0b"}, {"alice", "--limit", "0"}, {"alice", "-L", "-1"}} {
		cmd := newCmdUserFollowers(userFactory(userTestConfig{tokenErr: cause}, func(*http.Request) (*http.Response, error) { t.Fatal("invalid args made request"); return nil, nil }))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil || errors.Is(err, cause) {
			t.Fatalf("args=%q error=%v", args, err)
		}
	}
	cmd := newCmdUserFollowers(userFactory(userTestConfig{tokenErr: cause}, func(*http.Request) (*http.Response, error) {
		t.Fatal("credential failure made request")
		return nil, nil
	}))
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"alice"})
	if err := cmd.Execute(); !errors.Is(err, cause) {
		t.Fatalf("error=%v", err)
	}
}

func TestUserFollowersEmptyAndNoPartialOutput(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			f := userFactory(userTestConfig{}, func(r *http.Request) (*http.Response, error) {
				if !fail {
					return userResponse(200, "[]"), nil
				}
				if r.URL.Query().Get("page") == "2" {
					return userResponse(404, `{"message":"missing"}`), nil
				}
				return userResponse(200, "["+strings.Repeat(`{"id":"1"},`, 99)+`{"id":"100"}]`), nil
			})
			cmd := newCmdUserFollowers(f)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			args := []string{"alice", "-L", "101"}
			if asJSON {
				args = append(args, "--json")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if (err != nil) != fail {
				t.Fatalf("error=%v", err)
			}
			want := "No followers found for user \"alice\".\n"
			if asJSON {
				want = "[]\n"
			}
			if fail {
				want = ""
			}
			if out.String() != want {
				t.Fatalf("output=%q want=%q", out.String(), want)
			}
		}
	}
}
