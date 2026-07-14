package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestGetPaginatedHonorsLimit(t *testing.T) {
	for _, limit := range []int{1, 30, 100, 150} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			requests := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if got := r.URL.Query().Get("per_page"); got != "100" {
					t.Fatalf("per_page = %q, want 100", got)
				}
				if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
					t.Fatalf("page = %q, want %d", got, requests)
				}

				start := (requests - 1) * maxPerPage
				items := make([]int, maxPerPage)
				for i := range items {
					items[i] = start + i
				}
				if err := json.NewEncoder(w).Encode(items); err != nil {
					t.Fatal(err)
				}
			})

			items, err := GetPaginated[int](client, limit, func(page, perPage int) string {
				return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != limit {
				t.Fatalf("len(items) = %d, want %d", len(items), limit)
			}
			for i, item := range items {
				if item != i {
					t.Fatalf("items[%d] = %d, want %d", i, item, i)
				}
			}
			wantRequests := (limit + maxPerPage - 1) / maxPerPage
			if requests != wantRequests {
				t.Fatalf("requests = %d, want %d", requests, wantRequests)
			}
		})
	}
}

func TestGetPaginatedStopsAtLastPage(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode([]int{1, 2})
	})

	items, err := GetPaginated[int](client, 30, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || requests != 1 {
		t.Fatalf("len(items) = %d, requests = %d", len(items), requests)
	}
}

func TestGetPaginatedRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			if _, err := GetPaginated[int](NewClient("token"), limit, nil); err == nil {
				t.Fatal("expected invalid limit error")
			}
		})
	}
}
