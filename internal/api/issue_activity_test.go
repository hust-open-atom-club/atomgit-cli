package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIssueAuditHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, path, query, body string
		call                    func(*Client) error
	}{
		{"activity", "/repos/a%20b/issues/7/operate_logs", "repo=demo%26other", `[{"id":17,"user":{"login":"a"},"content":"x","issue_id":"42"}]`, func(c *Client) error {
			v, e := ListIssueActivity(c, "a b", "demo&other", "7", 1)
			if e == nil && (len(v) != 1 || v[0].ID != 17 || v[0].IssueID != "42") {
				t.Errorf("activity %v", v)
			}
			return e
		}},
		{"history", "/repos/a%20b/demo&other/issues/7/modify_history", "", `[{"id":"opaque","created":true,"deleted":false,"updated_user":{"login":"b"}}]`, func(c *Client) error {
			v, e := ListIssueHistory(c, "a b", "demo&other", "7", 1)
			if e == nil && (len(v) != 1 || v[0].ID != "opaque" || !v[0].Created || v[0].UpdatedUser.Login != "b") {
				t.Errorf("history %v", v)
			}
			return e
		}},
		{"reactions", "/repos/a%20b/demo&other/issues/7/user_reactions", "page=1&per_page=100", `[{"id":"opaque","emoji":"👍","emoji_name":"like","user":{"login":"a"}}]`, func(c *Client) error {
			v, e := ListIssueReactions(c, "a b", "demo&other", "7", 1)
			if e == nil && (len(v) != 1 || v[0].EmojiName != "like" || v[0].ID != "opaque") {
				t.Errorf("reactions %v", v)
			}
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.EscapedPath() != tc.path || r.URL.RawQuery != tc.query {
					t.Errorf("request %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing authorization")
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			client := NewClient("test-token")
			client.baseURL = server.URL
			if err := tc.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIssueReactionsPaginationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		limit, wantCalls, wantLen int
		failSecond                bool
	}{
		{"stop at limit", 1, 1, 1, false}, {"exact page limit", 100, 1, 100, false},
		{"empty tail", 101, 2, 100, false}, {"second page error", 101, 2, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("page") != fmt.Sprint(calls) {
					t.Error("wrong page")
				}
				if calls == 2 {
					if tc.failSecond {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
					} else {
						fmt.Fprint(w, "[]")
					}
					return
				}
				values := make([]string, 100)
				for i := range values {
					values[i] = fmt.Sprintf(`{"id":"%d","emoji_name":"like"}`, i)
				}
				fmt.Fprintf(w, "[%s]", strings.Join(values, ","))
			}))
			defer server.Close()
			c := NewClient("")
			c.baseURL = server.URL
			rows, err := ListIssueReactions(c, "alice", "demo", "7", tc.limit)
			if (err != nil) != tc.failSecond {
				t.Fatalf("error %v", err)
			}
			if calls != tc.wantCalls || len(rows) != tc.wantLen {
				t.Fatalf("calls=%d rows=%d", calls, len(rows))
			}
		})
	}
}

func TestIssueAuditRejectsInvalidIDs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		decode func([]byte) error
	}{
		{"activity zero", `{"id":0}`, func(b []byte) error { var v IssueActivity; return json.Unmarshal(b, &v) }},
		{"activity negative", `{"id":-1}`, func(b []byte) error { var v IssueActivity; return json.Unmarshal(b, &v) }},
		{"history empty", `{"id":""}`, func(b []byte) error { var v IssueHistory; return json.Unmarshal(b, &v) }},
		{"history whitespace", `{"id":" \t"}`, func(b []byte) error { var v IssueHistory; return json.Unmarshal(b, &v) }},
		{"reaction empty", `{"id":""}`, func(b []byte) error { var v IssueReaction; return json.Unmarshal(b, &v) }},
		{"reaction whitespace", `{"id":" \t"}`, func(b []byte) error { var v IssueReaction; return json.Unmarshal(b, &v) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.decode([]byte(tc.body)); err == nil {
				t.Fatal("invalid ID accepted")
			}
		})
	}
}

func TestIssueAuditValidatesBeforeLimit(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		call       func(*Client) error
	}{
		{"activity", `[{"id":1},null]`, func(c *Client) error { _, e := ListIssueActivity(c, "a", "b", "7", 1); return e }},
		{"history", `[{"id":"h"},null]`, func(c *Client) error { _, e := ListIssueHistory(c, "a", "b", "7", 1); return e }},
		{"reactions", `[{"id":"r"},null]`, func(c *Client) error { _, e := ListIssueReactions(c, "a", "b", "7", 1); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, tc.body) }))
			defer server.Close()
			c := NewClientWithBaseURL("", server.URL, server.Client())
			if err := tc.call(c); err == nil {
				t.Fatal("malformed record beyond limit was ignored")
			}
		})
	}
}

func TestIssueReactionsRejectsInvalidLaterPage(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 2 {
			fmt.Fprint(w, "[null]")
			return
		}
		rows := make([]string, 100)
		for i := range rows {
			rows[i] = fmt.Sprintf(`{"id":"r%d"}`, i)
		}
		fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
	}))
	defer server.Close()
	c := NewClientWithBaseURL("", server.URL, server.Client())
	rows, err := ListIssueReactions(c, "a", "b", "7", 101)
	if err == nil || len(rows) != 0 || calls != 2 {
		t.Fatalf("rows=%d calls=%d error=%v", len(rows), calls, err)
	}
}
