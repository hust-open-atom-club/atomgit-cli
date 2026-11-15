package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListUserFollowingMixedAccounts(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/users/alice%252Fadmin/following" || r.URL.RawQuery != "page=1&per_page=30" {
			t.Errorf("request=%s %s", r.Method, r.URL)
		}
		fmt.Fprint(w, `[{"id":"9007199254740993","object_id":"hex-id","login":"bob","type":"User","following_url":"/users/bob/following"},{"id":"org-id","login":null,"name":null,"type":"Organization","avatar_url":null}]`)
	})
	items, err := ListUserFollowing(c, "alice%2Fadmin", 30)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].ID != "9007199254740993" || items[0].ObjectID != "hex-id" || items[1].FollowingURL != "" || items[1].Type != "Organization" {
		t.Fatalf("items=%+v", items)
	}
}

func TestListUserFollowingPagination(t *testing.T) {
	for _, limit := range []int{1, 30, 101, 150} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			calls := 0
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				size := min(limit, 100)
				if r.URL.Query().Get("page") != fmt.Sprint(calls) || r.URL.Query().Get("per_page") != fmt.Sprint(size) {
					t.Errorf("query=%s", r.URL.RawQuery)
				}
				items := make([]FollowedAccount, size)
				for i := range items {
					items[i] = FollowedAccount{ID: fmt.Sprint((calls-1)*size + i), Type: "User"}
				}
				_ = json.NewEncoder(w).Encode(items)
			})
			items, err := ListUserFollowing(c, "alice", limit)
			if err != nil || len(items) != limit || calls != (limit+99)/100 {
				t.Fatalf("len=%d calls=%d err=%v", len(items), calls, err)
			}
		})
	}
}

func TestListUserFollowingInvalidResponses(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"message":"error"}`, `[null]`, `[{"type":"Other"}]`, `[{"type":"User","id":123}]`, `[`, ``} {
		t.Run(body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) })
			items, err := ListUserFollowing(c, "alice", 30)
			if err == nil || items != nil {
				t.Fatalf("items=%v err=%v", items, err)
			}
		})
	}
	for _, status := range []int{201, 204, 400, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status); fmt.Fprint(w, `[]`) })
			c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": {"9999"}}, Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
			})
			items, err := ListUserFollowing(c, "alice", 30)
			if err == nil || items != nil || !IsHTTPStatus(err, status) {
				t.Fatalf("items=%v err=%v", items, err)
			}
		})
	}
}

func TestListUserFollowingNoPartialResults(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 2 {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"hidden"}`)
			return
		}
		items := make([]FollowedAccount, 100)
		for i := range items {
			items[i].Type = "User"
		}
		_ = json.NewEncoder(w).Encode(items)
	})
	items, err := ListUserFollowing(c, "alice", 101)
	if !IsHTTPStatus(err, 404) || items != nil || calls != 2 {
		t.Fatalf("items=%v err=%v calls=%d", items, err, calls)
	}
}

func TestListUserFollowingEmptyAndInvalid(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { calls++; fmt.Fprint(w, `[]`) })
	for _, limit := range []int{0, -1} {
		if _, err := ListUserFollowing(c, "alice", limit); err == nil {
			t.Fatal("accepted invalid limit")
		}
	}
	if _, err := ListUserFollowing(c, "", 30); err == nil {
		t.Fatal("accepted empty username")
	}
	if calls != 0 {
		t.Fatal("invalid inputs made requests")
	}
	items, err := ListUserFollowing(c, "alice", 30)
	if err != nil || items == nil || len(items) != 0 {
		t.Fatalf("items=%v err=%v", items, err)
	}
}
