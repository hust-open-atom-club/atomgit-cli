package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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
		want       []UserFollower
	}{
		{"empty", "[]", 200, []UserFollower{}},
		{"nullable fields", `[{"id":"9007199254740993123","login":null,"name":null,"type":null,"html_url":null,"avatar_url":null}]`, 200, []UserFollower{{ID: "9007199254740993123"}}},
		{"nullable ID", `[{"id":null,"login":"bob"}]`, 200, []UserFollower{{Login: "bob"}}},
		{"field mapping", `[{"id":"000496","login":"bob","name":"Bob","type":"User","html_url":"https://atomgit.com/bob","avatar_url":"https://cdn.example/bob.png"}]`, 200, []UserFollower{{ID: "000496", Login: "bob", Name: "Bob", Type: "User", HTMLURL: "https://atomgit.com/bob", AvatarURL: "https://cdn.example/bob.png"}}},
		{"null", "null", 200, nil},
		{"null item", "[null]", 200, nil},
		{"object", "{}", 200, nil},
		{"numeric ID", `[{"id":123}]`, 200, nil},
		{"malformed", "[", 200, nil},
		{"created", "[]", 201, nil},
		{"unauthorized", "{}", 401, nil},
		{"forbidden", "{}", 403, nil},
		{"not found", "{}", 404, nil},
		{"rate limited", "{}", 429, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			items, err := ListUserFollowers(client, "alice", 30)
			if tt.want == nil {
				if err == nil || items != nil {
					t.Fatalf("items=%#v error=%v", items, err)
				}
				if tt.status != http.StatusOK {
					if !IsHTTPStatus(err, tt.status) {
						t.Fatalf("lost HTTP status %d: %v", tt.status, err)
					}
				} else if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("expected invalid response error: %v", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(items, tt.want) {
				t.Fatalf("items=%#v want=%#v error=%v", items, tt.want, err)
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
