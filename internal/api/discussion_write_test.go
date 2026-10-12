package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreateDeleteDiscussionContract(t *testing.T) {
	for _, method := range []string{"POST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				path := "/repos/org%2Fname/re%20po/discuss"
				if method == "DELETE" {
					path += "/7"
				}
				if r.Method != method || r.URL.EscapedPath() != path {
					t.Errorf("request=%s %s", r.Method, r.URL.EscapedPath())
				}
				if method == "POST" {
					if r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("Content-Type=%q", r.Header.Get("Content-Type"))
					}
					var fields map[string]string
					if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
						t.Error(err)
					}
					if len(fields) != 3 || fields["title"] != "Title" || fields["md_content"] != "# 中文\n" || fields["category_name"] != "Ideas" {
						t.Errorf("fields=%#v", fields)
					}
					_, _ = io.WriteString(w, `{"number":7,"id":"d1"}`)
				}
			})
			if method == "POST" {
				d, err := CreateDiscussion(client, "org/name", "re po", DiscussionCreateRequest{Title: "Title", MDContent: "# 中文\n", CategoryName: "Ideas"})
				if err != nil || d.Number != 7 {
					t.Fatalf("detail=%+v err=%v", d, err)
				}
			} else if err := DeleteDiscussion(client, "org/name", "re po", 7); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDiscussionWritesRejectUnexpectedStatusesAndRedirects(t *testing.T) {
	for _, method := range []string{"POST", "DELETE"} {
		for _, status := range []int{201, 204, 301, 302, 303, 307, 308} {
			t.Run(method+http.StatusText(status), func(t *testing.T) {
				calls, redirects := 0, 0
				hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://api.atomgit.com/api/v5/other"}}, Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
				})}
				client := NewClientWithHTTPClient("fake-token", hc)
				var err error
				if method == "POST" {
					_, err = CreateDiscussion(client, "o", "r", DiscussionCreateRequest{})
				} else {
					err = DeleteDiscussion(client, "o", "r", 7)
				}
				if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
					t.Fatalf("err=%v calls=%d redirects=%d", err, calls, redirects)
				}
				if err := hc.CheckRedirect(nil, nil); err != nil || redirects != 1 {
					t.Fatal("mutated shared HTTP client")
				}
			})
		}
	}
}
