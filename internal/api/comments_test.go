package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func commentReaders() []struct {
	kind string
	list func(*Client, string, string, int) ([]Comment, error)
} {
	return []struct {
		kind string
		list func(*Client, string, string, int) ([]Comment, error)
	}{{"pulls", ListPullRequestComments}, {"issues", ListIssueComments}}
}

func TestListAllCommentsPagination(t *testing.T) {
	for _, reader := range commentReaders() {
		for _, total := range []int{0, 20, 21, 100, 101} {
			for _, pageSize := range []int{20, 100} {
				t.Run(fmt.Sprintf("%s/total=%d/server-page-size=%d", reader.kind, total, pageSize), func(t *testing.T) {
					want := make([]Comment, total)
					for i := range want {
						want[i] = Comment{ID: int64(i + 1), Body: fmt.Sprintf("comment-%d", i+1)}
					}
					if total > 0 {
						want[total-1].CommentType = "diff_comment"
						want[total-1].DiscussionID = "review-thread"
						want[total-1].Reply = []Comment{{ID: 1001, Body: "nested reply", Reply: []Comment{{ID: 1002, Body: "further reply"}}}}
					}
					calls := 0
					client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						query := r.URL.Query()
						if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/v5/repos/team%20space/demo%3Fx/"+reader.kind+"/8/comments" || query.Get("page") != strconv.Itoa(calls) || query.Get("per_page") != "100" {
							t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
						}
						if (query.Get("view") == "all") != (reader.kind == "pulls") {
							t.Fatalf("view query: %q", query.Get("view"))
						}
						deadline, ok := r.Context().Deadline()
						if !ok || time.Until(deadline) > defaultMetadataTimeout {
							t.Fatal("missing traversal deadline")
						}
						start := min((calls-1)*pageSize, total)
						end := min(start+pageSize, total)
						body, err := json.Marshal(want[start:end])
						if err != nil {
							t.Fatal(err)
						}
						return discussionStateResponse(200, string(body)), nil
					})})
					got, err := reader.list(client, "team space", "demo?x", 8)
					wantCalls := (total+pageSize-1)/pageSize + 1
					if err != nil || !reflect.DeepEqual(got, want) || calls != wantCalls {
						t.Fatalf("got=%+v err=%v calls=%d wantCalls=%d", got, err, calls, wantCalls)
					}
				})
			}
		}
	}
}

func TestListAllCommentsDiscardsPartialResults(t *testing.T) {
	for _, reader := range commentReaders() {
		for _, tc := range []struct {
			name, body string
			status     int
		}{
			{"forbidden", `{}`, 403}, {"unexpected success", `[]`, 201},
			{"malformed", `[`, 200}, {"null page", `null`, 200}, {"null item", `[null]`, 200},
		} {
			t.Run(reader.kind+"/"+tc.name, func(t *testing.T) {
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return discussionStateResponse(200, `[{"id":1,"body":"partial"}]`), nil
					}
					return discussionStateResponse(tc.status, tc.body), nil
				})})
				got, err := reader.list(client, "team", "demo", 8)
				if err == nil || got != nil || calls != 2 {
					t.Fatalf("got=%v error=%v calls=%d", got, err, calls)
				}
				if tc.status != 200 {
					if !IsHTTPStatus(err, tc.status) {
						t.Fatalf("lost HTTP status: %v", err)
					}
				} else if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("lost invalid response classification: %v", err)
				}
			})
		}
	}
}

func TestListAllCommentsCancellationAndPageBound(t *testing.T) {
	for _, reader := range commentReaders() {
		t.Run(reader.kind+"/cancel", func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				cancel()
				return discussionStateResponse(200, `[{"id":1}]`), nil
			})}).WithContext(ctx)
			got, err := reader.list(client, "team", "demo", 8)
			if got != nil || !errors.Is(err, context.Canceled) || calls != 1 {
				t.Fatalf("got=%v error=%v calls=%d", got, err, calls)
			}
		})
		t.Run(reader.kind+"/deadline", func(t *testing.T) {
			ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
			defer cancel()
			client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("expired deadline sent a request")
				return nil, nil
			})}).WithContext(ctx)
			if got, err := reader.list(client, "team", "demo", 8); got != nil || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("got=%v error=%v", got, err)
			}
		})
		t.Run(reader.kind+"/page bound", func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return discussionStateResponse(200, `[{"id":1}]`), nil
			})})
			got, err := reader.list(client, "team", "demo", 8)
			if got != nil || err == nil || !strings.Contains(err.Error(), "maximum of 100 pages") || calls != maxUntilEmptyRequestPages {
				t.Fatalf("got=%v error=%v calls=%d", got, err, calls)
			}
		})
	}
}
