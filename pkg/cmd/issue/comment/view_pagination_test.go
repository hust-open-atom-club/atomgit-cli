package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestIssueCommentViewPaginates(t *testing.T) {
	for _, total := range []int{0, 20, 21, 100, 101} {
		t.Run(strconv.Itoa(total), func(t *testing.T) {
			items := make([]api.Comment, total)
			for i := range items {
				items[i] = api.Comment{ID: int64(i + 1), Body: fmt.Sprintf("entry-%03d", i+1), CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(total-i) * time.Minute).Format(time.RFC3339), User: api.User{Login: "alice"}}
			}

			calls := 0
			f := &cmdutil.Factory{Config: issueCommentTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueCommentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != http.MethodGet || r.URL.Path != "/api/v5/repos/alice/demo/issues/8/comments" || r.URL.Query().Get("page") != strconv.Itoa(calls) || r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("view") != "" {
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
					}
					start := min((calls-1)*20, total)
					body, err := json.Marshal(items[start:min(start+20, total)])
					if err != nil {
						t.Fatal(err)
					}
					return issueJSONResponse(200, string(body)), nil
				})}, nil
			}}
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("explicit repository was not used")
				return cmdutil.Repository{}, nil
			}
			cmd := newCmdView(f)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"alice/demo", "8"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			wantCalls := (total+19)/20 + 1
			if calls != wantCalls {
				t.Fatalf("calls=%d want=%d", calls, wantCalls)
			}
			if total == 0 {
				if out.String() != "No comments found.\n" {
					t.Fatalf("output=%q", out.String())
				}
				return
			}
			if !strings.Contains(out.String(), fmt.Sprintf("Issue #8 的评论 (共 %d 条):", total)) || strings.Count(out.String(), "entry-") != total {
				t.Fatalf("incomplete list: %s", out.String())
			}
			if total > 1 && strings.Index(out.String(), fmt.Sprintf("entry-%03d", total)) > strings.Index(out.String(), "entry-001") {
				t.Fatal("comments were not sorted across pages")
			}

		})
	}
}

func TestIssueCommentViewFailurePrintsNoPartialList(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", canceled), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			f := &cmdutil.Factory{Config: issueCommentTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueCommentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						if canceled {
							cancel()
						}
						return issueJSONResponse(200, `[{"id":1,"body":"partial result"}]`), nil
					}
					return issueJSONResponse(403, `{}`), nil
				})}, nil
			}}
			cmd := newCmdView(f)
			cmd.SetContext(ctx)
			cmd.SilenceUsage, cmd.SilenceErrors = true, true
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"alice/demo", "8"})
			err := cmd.Execute()
			if err == nil || out.Len() != 0 {
				t.Fatalf("error=%v output=%q", err, out.String())
			}
			if canceled {
				if !errors.Is(err, context.Canceled) || calls != 1 {
					t.Fatalf("error=%v calls=%d", err, calls)
				}
			} else if !api.IsHTTPStatus(err, 403) || calls != 2 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
}
