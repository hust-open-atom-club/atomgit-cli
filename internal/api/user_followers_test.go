package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func TestListUserFollowersPagination(t *testing.T) {
	for _, limit := range []int{30, 100, 145, 200} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.EscapedPath() != "/users/alice%25dev/followers" || r.URL.Query().Get("page") != strconv.Itoa(calls) || r.URL.Query().Get("per_page") != "100" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				count := 100
				if calls == 2 {
					count = 70
				}
				items := make([]UserFollower, count)
				for i := range items {
					items[i] = UserFollower{ID: fmt.Sprintf("000%d", (calls-1)*100+i), Login: "alice"}
				}
				_ = json.NewEncoder(w).Encode(items)
			})
			items, err := ListUserFollowers(client, "alice%dev", limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != min(limit, 170) || items[0].ID != "0000" {
				t.Fatalf("items = %#v", items)
			}
			wantCalls := 1
			if limit > 100 {
				wantCalls = 2
			}
			if calls != wantCalls {
				t.Fatalf("calls = %d, want %d", calls, wantCalls)
			}
		})
	}
}

func TestListUserFollowersResponseContract(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"empty", "[]", 200, false},
		{"nullable fields", `[{"id":"9007199254740993123","login":null,"name":null,"avatar_url":null}]`, 200, false},
		{"null", "null", 200, true},
		{"null item", "[null]", 200, true},
		{"object", "{}", 200, true},
		{"numeric ID", `[{"id":123}]`, 200, true},
		{"malformed", "[", 200, true},
		{"created", "[]", 201, true},
		{"unauthorized", "{}", 401, true},
		{"forbidden", "{}", 403, true},
		{"not found", "{}", 404, true},
		{"rate limited", "{}", 429, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			items, err := ListUserFollowers(client, "alice", 30)
			if (err != nil) != tt.fail {
				t.Fatalf("items=%#v error=%v", items, err)
			}
			if !tt.fail && items == nil {
				t.Fatal("empty result must be a non-nil slice")
			}
			if tt.name == "nullable fields" && items[0].ID != "9007199254740993123" {
				t.Fatal("ID changed")
			}
		})
	}
}

func TestListUserFollowersDiscardsPartialResults(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(403)
			return
		}
		_ = json.NewEncoder(w).Encode(make([]UserFollower, 100))
	})
	items, err := ListUserFollowers(client, "alice", 101)
	if err == nil || items != nil {
		t.Fatalf("items=%#v error=%v", items, err)
	}
	client = newTestClient(t, func(http.ResponseWriter, *http.Request) { t.Fatal("invalid limit made a request") })
	if _, err := ListUserFollowers(client, "alice", 0); err == nil {
		t.Fatal("accepted zero limit")
	}
}
