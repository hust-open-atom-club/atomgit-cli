package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestReplyCommentIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		id         int64
	}{
		{"official", `{"id":"thread","noteId":42}`, 42},
		{"legacy", `{"id":"thread","note_id":41}`, 41},
		{"official wins", `{"id":"thread","noteId":42,"note_id":41}`, 42},
		{"null fallback", `{"id":"thread","noteId":null,"note_id":41}`, 41},
		{"large exact ID", `{"id":"thread","noteId":9007199254740993}`, 9007199254740993},
		{"missing", `{"id":"thread"}`, 0},
		{"zero", `{"id":"thread","noteId":0}`, 0},
		{"negative", `{"id":"thread","noteId":-1}`, 0},
		{"zero cannot fall back", `{"id":"thread","noteId":0,"note_id":41}`, 0},
		{"negative cannot fall back", `{"id":"thread","noteId":-2,"note_id":41}`, 0},
		{"null", `{"id":"thread","noteId":null}`, 0},
		{"zero legacy", `{"id":"thread","note_id":0}`, 0},
		{"negative legacy", `{"id":"thread","note_id":-1}`, 0},
		{"string", `{"id":"thread","noteId":"42"}`, 0},
		{"fraction", `{"id":"thread","noteId":42.5}`, 0},
		{"overflow", `{"id":"thread","noteId":9223372036854775808}`, 0},
		{"boolean", `{"id":"thread","noteId":true}`, 0},
		{"object ID", `{"id":"thread","noteId":{}}`, 0},
		{"invalid legacy", `{"id":"thread","note_id":"41"}`, 0},
		{"different discussion", `{"id":"other","noteId":42}`, 0},
		{"missing discussion", `{"noteId":42}`, 0},
		{"discussion is not reply ID", `{"id":"42"}`, 0},
		{"null response", `null`, 0},
		{"array", `[]`, 0},
		{"malformed", `{`, 0},
		{"empty response", ``, 0},
	} {
		for _, status := range []int{http.StatusOK, http.StatusCreated} {
			t.Run(tc.name+"/"+http.StatusText(status), func(t *testing.T) {
				for _, inferred := range []bool{false, true} {
					calls := 0
					ctx := context.WithValue(t.Context(), replyContextKey{}, "reply-context")
					f := &cmdutil.Factory{Config: prCommentTestConfig{}, HTTPClient: func() (*http.Client, error) {
						return &http.Client{Transport: prCommentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
							calls++
							if r.Method != http.MethodPost || r.URL.EscapedPath() != "/api/v5/repos/alice/demo/pulls/8/discussions/thread/comments" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer token" || r.Context().Value(replyContextKey{}) != "reply-context" {
								t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
							}
							var body map[string]string
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || body["body"] != " reply\ntext " {
								t.Fatalf("body=%v error=%v", body, err)
							}
							return prJSONResponse(status, tc.body), nil
						})}, nil
					}}
					f.RepositoryResolver = func() (cmdutil.Repository, error) {
						if !inferred {
							t.Fatal("explicit repository should win")
						}
						return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
					}
					cmd := newCmdReply(f)
					cmd.SilenceUsage, cmd.SilenceErrors = true, true
					args := []string{"8", "thread", "--body", " reply\ntext "}
					if !inferred {
						args = append([]string{"alice/demo"}, args...)
					}
					cmd.SetArgs(args)
					var out bytes.Buffer
					cmd.SetOut(&out)
					cmd.SetErr(io.Discard)
					err := cmd.ExecuteContext(ctx)
					if calls != 1 {
						t.Fatalf("calls=%d", calls)
					}
					if tc.id > 0 {
						want := "Created reply #" + strconv.FormatInt(tc.id, 10) + " in discussion thread\n"
						if err != nil || out.String() != want {
							t.Fatalf("error=%v output=%q", err, out.String())
						}
					} else {
						var uncertain *api.PullRequestReplyUncertainError
						if !errors.As(err, &uncertain) || !errors.Is(err, api.ErrInvalidResponse) || out.Len() != 0 || !strings.Contains(err.Error(), "may have been created") {
							t.Fatalf("error=%v output=%q", err, out.String())
						}
					}
				}
			})
		}
	}
}

type replyContextKey struct{}
