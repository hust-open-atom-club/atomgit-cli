package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestListOrganizationDiscussionsPagination(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit int
		pages []string
		want  []int
	}{
		{"empty", 30, []string{`[]`}, []int{}},
		{"short pages and overlap", 30, []string{`[{"number":1}]`, `[{"number":1},{"number":2}]`, `[]`}, []int{1, 2}},
		{"limit", 2, []string{`[{"number":1}]`, `[{"number":2},{"number":3}]`}, []int{1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.URL.EscapedPath() != "/orgs/org%2Fname%20%3F%25/discuss" {
					t.Fatalf("request = %s %s", r.Method, r.URL)
				}
				if r.URL.Query().Get("page") != strconv.Itoa(calls) || r.URL.Query().Get("per_page") != "100" {
					t.Fatalf("query = %s", r.URL.RawQuery)
				}
				if r.Header.Get("Accept") != "application/json" {
					t.Fatalf("Accept = %s", r.Header.Get("Accept"))
				}
				if calls > len(tc.pages) {
					t.Fatal("unexpected extra page")
				}
				_, _ = io.WriteString(w, tc.pages[calls-1])
			})
			items, err := ListOrganizationDiscussions(client, "org/name ?%", tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]int, 0, len(items))
			for _, item := range items {
				got = append(got, item.Number)
			}
			if !reflect.DeepEqual(got, tc.want) || calls != len(tc.pages) || items == nil {
				t.Fatalf("items = %#v, calls = %d", items, calls)
			}
		})
	}
}

func TestListOrganizationDiscussionsFullPageLimit(t *testing.T) {
	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		items := make([]Discussion, 100)
		for i := range items {
			items[i].Number = (page-1)*100 + i + 1
		}
		_ = json.NewEncoder(w).Encode(items)
	})
	items, err := ListOrganizationDiscussions(client, "team", 101)
	if err != nil || len(items) != 101 || items[100].Number != 101 || calls != 2 {
		t.Fatalf("len = %d, calls = %d, err = %v", len(items), calls, err)
	}
}

func TestListOrganizationDiscussionsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		if _, err := ListOrganizationDiscussions(nil, "team", limit); err == nil {
			t.Fatalf("limit %d accepted", limit)
		}
	}
}

func TestGetOrganizationDiscussion(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/orgs/org%2Fname/discuss/7" || r.URL.RawQuery != "" {
			t.Fatalf("request = %s %s", r.Method, r.URL)
		}
		_, _ = io.WriteString(w, `{"number":7,"title":"Roadmap","md_content":"# Plan\nBody","author":{"login":"alice"},"category":{"name":"Ideas"},"is_closed":1,"is_lock":1}`)
	})
	detail, err := GetOrganizationDiscussion(client, "org/name", 7)
	if err != nil || detail.MDContent != "# Plan\nBody" || detail.Author.Login != "alice" || detail.Category.Name != "Ideas" || !bool(detail.IsClosed) || !bool(detail.IsLocked) {
		t.Fatalf("detail = %+v, err = %v", detail, err)
	}
}

func TestOrganizationDiscussionThreads(t *testing.T) {
	commentsPath := "/orgs/org%2Fname/discuss/7/comment"
	responses := map[string][]string{
		commentsPath:                          {`[{"id":"c /?%","reply_total":0}]`, `[{"id":"c2","is_hide":1}]`, `[]`},
		commentsPath + "/c%20%2F%3F%25/reply": {`[{"id":"r1","md_content":"**reply**"}]`, `[{"id":"r1"},{"id":"r2","is_deleted":1}]`, `[]`},
		commentsPath + "/c2/reply":            {`[]`},
	}
	calls := make(map[string]int)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		calls[path]++
		pages, ok := responses[path]
		if !ok || calls[path] > len(pages) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		if r.Method != http.MethodGet || r.URL.Query().Get("page") != strconv.Itoa(calls[path]) || r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("request = %s %s", r.Method, r.URL)
		}
		_, _ = io.WriteString(w, pages[calls[path]-1])
	})
	threads, err := ListOrganizationDiscussionThreads(client, "org/name", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 2 || len(threads[0].Replies) != 2 || threads[0].Replies[0].MDContent != "**reply**" || !bool(threads[0].Replies[1].IsDeleted) || !bool(threads[1].Comment.IsHidden) || threads[1].Replies == nil {
		t.Fatalf("threads = %+v", threads)
	}
	for path, pages := range responses {
		if calls[path] != len(pages) {
			t.Fatalf("%s: calls = %d", path, calls[path])
		}
	}
}

func TestOrganizationDiscussionReadFailures(t *testing.T) {
	for _, stage := range []string{"list", "detail", "comments", "replies"} {
		for _, tc := range []struct {
			name   string
			status int
			body   string
		}{
			{"unauthorized", 401, `{}`},
			{"forbidden", 403, `{}`},
			{"not found", 404, `{}`},
			{"unexpected success", 201, `{}`},
			{"no content", 204, ""},
			{"invalid JSON", 200, `{`},
			{"null", 200, `null`},
			{"wrong shape", 200, `"unexpected"`},
			{"missing identity", 200, ""},
		} {
			t.Run(stage+"/"+tc.name, func(t *testing.T) {
				client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					if stage == "replies" && strings.HasSuffix(r.URL.Path, "/comment") {
						if r.URL.Query().Get("page") == "1" {
							_, _ = io.WriteString(w, `[{"id":"c1"}]`)
						} else {
							_, _ = io.WriteString(w, `[]`)
						}
						return
					}
					w.WriteHeader(tc.status)
					body := tc.body
					if tc.name == "missing identity" {
						body = `[{}]`
						if stage == "detail" {
							body = `{}`
						}
					}
					_, _ = io.WriteString(w, body)
				})
				var err error
				switch stage {
				case "list":
					_, err = ListOrganizationDiscussions(client, "team", 30)
				case "detail":
					_, err = GetOrganizationDiscussion(client, "team", 7)
				default:
					var threads []DiscussionCommentThread
					threads, err = ListOrganizationDiscussionThreads(client, "team", 7)
					if threads != nil {
						t.Fatal("failed request returned partial threads")
					}
				}
				if err == nil || !strings.Contains(err.Error(), "organization discussion") {
					t.Fatalf("error = %v", err)
				}
				if tc.status != 200 && !IsHTTPStatus(err, tc.status) {
					t.Fatalf("HTTP status lost: %v", err)
				}
				if tc.status == 200 && !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("invalid response classification lost: %v", err)
				}
			})
		}
	}
}

func TestOrganizationDiscussionPaginationGuards(t *testing.T) {
	for _, stage := range []string{"list", "comments", "replies"} {
		for _, repeat := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/repeat=%t", stage, repeat), func(t *testing.T) {
				calls := 0
				client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					if stage == "replies" && strings.HasSuffix(r.URL.Path, "/comment") {
						if r.URL.Query().Get("page") == "1" {
							_, _ = io.WriteString(w, `[{"id":"c1"}]`)
						} else {
							_, _ = io.WriteString(w, `[]`)
						}
						return
					}
					calls++
					id := calls
					if repeat {
						id = 1
					}
					_, _ = fmt.Fprintf(w, `[{"id":"c%d","number":%d}]`, id, id)
				})
				var err error
				if stage == "list" {
					_, err = ListOrganizationDiscussions(client, "team", 200)
				} else {
					_, err = ListOrganizationDiscussionThreads(client, "team", 7)
				}
				wantCalls, wantError := maxUntilEmptyRequestPages, "maximum of 100 pages"
				if repeat {
					wantCalls, wantError = 2, "no progress"
				}
				if err == nil || !strings.Contains(err.Error(), wantError) || calls != wantCalls {
					t.Fatalf("calls = %d, err = %v", calls, err)
				}
			})
		}
	}
}
