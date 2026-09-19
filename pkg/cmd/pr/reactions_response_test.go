package pr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestPRReactionsDocumentedResponse(t *testing.T) {
	const body = `[{"id":"6aadf74ba99efd72002e8dcd","emoji":"👀","emoji_name":"eyes","user":{"login":"alice"}},{"id":9007199254740993,"emoji":"👍","user":{"login":"bob"}}]`
	for _, mode := range []string{"text", "json"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			factory := &cmdutil.Factory{Config: prTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/pulls/42/user_reactions" {
						t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					return prResponse(http.StatusOK, body), nil
				})}, nil
			}}
			cmd := newCmdPRReactions(factory)
			if mode == "json" {
				_ = cmd.Flags().Set("json", "true")
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("requests = %d, want 1", calls)
			}
			if mode == "text" {
				if got, want := out.String(), "eyes 👀 by alice\n👍 by bob\n"; got != want {
					t.Fatalf("output = %q, want %q", got, want)
				}
				return
			}
			var rows []map[string]any
			if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 {
				t.Fatalf("rows = %#v", rows)
			}
			for i, want := range []map[string]any{
				{"id": "6aadf74ba99efd72002e8dcd", "author": "alice", "emoji": "👀", "emojiName": "eyes", "content": "eyes", "createdAt": ""},
				{"id": "9007199254740993", "author": "bob", "emoji": "👍", "emojiName": "", "content": "👍", "createdAt": ""},
			} {
				if len(rows[i]) != len(want) {
					t.Errorf("unexpected schema: %#v", rows[i])
				}
				for k, v := range want {
					if rows[i][k] != v {
						t.Errorf("row %d field %s = %v, want %v", i, k, rows[i][k], v)
					}
				}
			}
		})
	}
}

func TestPRReactionsInvalidRecordDoesNotEmitPartialOutput(t *testing.T) {
	for _, body := range []string{`[null]`, `[{"id":"good","emoji_name":"like"},null]`, `[{"id":"good"},{"emoji":"👀"}]`} {
		for _, mode := range []string{"text", "json"} {
			t.Run(body+"/"+mode, func(t *testing.T) {
				factory := &cmdutil.Factory{Config: prTestConfig{}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) { return prResponse(http.StatusOK, body), nil })}, nil
				}}
				cmd := newCmdPRReactions(factory)
				if mode == "json" {
					_ = cmd.Flags().Set("json", "true")
				}
				var out bytes.Buffer
				cmd.SetOut(&out)
				err := cmd.RunE(cmd, []string{"alice/demo", "42"})
				if err == nil || !strings.Contains(err.Error(), "invalid pull request reaction record") {
					t.Fatalf("error = %v", err)
				}
				if out.Len() != 0 {
					t.Fatalf("partial output = %q", out.String())
				}
			})
		}
	}
}
