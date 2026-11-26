package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestUnifiedSearchRejectsHTTP201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"page_count":1,"content":[{"iid":17}]}`)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("", server.URL, server.Client())
	results, err := client.SearchPullRequests(SearchOptions{Query: "cli", Limit: 1})
	var httpErr *HTTPError
	if results != nil || !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusCreated {
		t.Fatalf("results = %#v, error = %v, want HTTP 201 error", results, err)
	}
}

func TestUnifiedSearchDeduplicatesOverlappingPages(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var content string
		switch r.URL.Query().Get("p") {
		case "1":
			content = `[{"id":"first","title":"original"},{"path_with_namespace":"org/one","iid":7,"title":"fallback"}]`
		case "2":
			content = `[{"id":"first","title":"duplicate"},{"id":"","path_with_namespace":"org/one","iid":7}]`
		case "3":
			content = `[]`
		case "4":
			content = `[{"path_with_namespace":"org/two","iid":7,"title":"different repository"},{"iid":7,"title":"unknown A"},{"iid":7,"title":"unknown B"},{"id":"last","title":"last"},{"id":"extra","title":"extra"}]`
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("p"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, `{"page_count":5,"content":%s}`, content)
	}))
	defer server.Close()

	client := NewClientWithBaseURL("", server.URL, server.Client())
	results, err := client.SearchPullRequests(SearchOptions{Query: "cli", Limit: 6})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, item := range results {
		if item.Title == nil {
			t.Fatalf("result without a title: %+v", item)
		}
		got = append(got, *item.Title)
	}
	want := []string{"original", "fallback", "different repository", "unknown A", "unknown B", "last"}
	if !slices.Equal(got, want) || requests.Load() != 4 {
		t.Fatalf("titles = %v, requests = %d, want %v after 4 requests", got, requests.Load(), want)
	}
}

func TestUnifiedSearchPageBudget(t *testing.T) {
	for _, tt := range []struct {
		name         string
		pageCount    int
		lastPageItem bool
		wantResults  int
		wantError    bool
	}{
		{name: "last server page at budget", pageCount: maxUnifiedSearchPages, wantResults: 1},
		{name: "limit reached at budget", pageCount: maxUnifiedSearchPages + 1, lastPageItem: true, wantResults: 2},
		{name: "server has more pages", pageCount: maxUnifiedSearchPages + 1, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := int(requests.Add(1))
				if got := r.URL.Query().Get("p"); got != strconv.Itoa(page) || page > maxUnifiedSearchPages {
					t.Errorf("unexpected page %q on request %d", got, page)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				content := `[]`
				if page == 1 {
					content = `[{"id":"first"}]`
				}
				if tt.lastPageItem && page == maxUnifiedSearchPages {
					content = `[{"id":"last"}]`
				}
				fmt.Fprintf(w, `{"page_count":%d,"content":%s}`, tt.pageCount, content)
			}))
			defer server.Close()

			client := NewClientWithBaseURL("", server.URL, server.Client())
			results, err := client.SearchUsers(SearchOptions{Query: "cli", Limit: 2})
			if tt.wantError {
				want := fmt.Sprintf("pagination reached the maximum of %d pages after collecting 1 of 2 requested results", maxUnifiedSearchPages)
				if results != nil || err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("results = %#v, error = %v, want no partial results and error containing %q", results, err, want)
				}
			} else if err != nil || len(results) != tt.wantResults {
				t.Fatalf("results = %#v, error = %v, want %d results", results, err, tt.wantResults)
			}
			if got := int(requests.Load()); got != maxUnifiedSearchPages {
				t.Fatalf("requests = %d, want %d", got, maxUnifiedSearchPages)
			}
		})
	}
}
