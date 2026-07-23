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

				start := (requests - 1) * defaultMaxPerPage
				items := make([]int, defaultMaxPerPage)
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
			wantRequests := (limit + defaultMaxPerPage - 1) / defaultMaxPerPage
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

func TestGetPaginatedWithPageSizeUsesEndpointLimit(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Fatalf("per_page = %q, want 50", got)
		}
		items := make([]int, 50)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			t.Fatal(err)
		}
	})

	items, err := GetPaginatedWithPageSize[int](client, 60, 50, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 60 || requests != 2 {
		t.Fatalf("len(items) = %d, requests = %d, want 60 and 2", len(items), requests)
	}
}

func TestGetPaginatedUntilEmptyContinuesAfterShortPage(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Fatalf("per_page = %q, want 50", got)
		}

		var items []int
		switch requests {
		case 1:
			items = make([]int, 48)
			for i := range items {
				items[i] = i
			}
		case 2:
			items = make([]int, 44)
			for i := range items {
				items[i] = 48 + i
			}
		default:
			t.Fatalf("unexpected request %d", requests)
		}

		if err := json.NewEncoder(w).Encode(items); err != nil {
			t.Fatal(err)
		}
	})

	items, err := GetPaginatedUntilEmptyWithPageSize[int](client, 51, 50, func(item int) string {
		return fmt.Sprint(item)
	}, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 51 {
		t.Fatalf("len(items) = %d, want 51", len(items))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if got := items[48]; got != 48 {
		t.Fatalf("items[48] = %d, want first item from second page", got)
	}
}

func TestGetPaginatedUntilEmptyDeduplicatesOverlappingPages(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		var items []int
		switch requests {
		case 1:
			items = []int{1, 2, 3}
		case 2:
			items = []int{3, 4, 5}
		case 3:
			items = []int{5, 6, 7}
		default:
			t.Fatalf("unexpected request %d", requests)
		}
		if err := json.NewEncoder(w).Encode(items); err != nil {
			t.Fatal(err)
		}
	})

	items, err := GetPaginatedUntilEmptyWithPageSize[int](client, 7, 3, func(item int) string {
		return fmt.Sprint(item)
	}, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(items), "[1 2 3 4 5 6 7]"; got != want {
		t.Fatalf("items = %s, want %s", got, want)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestGetPaginatedUntilEmptyRejectsNilKey(t *testing.T) {
	if _, err := GetPaginatedUntilEmptyWithPageSize[int](NewClient("token"), 1, 1, nil, nil); err == nil {
		t.Fatal("expected nil key error")
	}
}

func TestGetPaginatedUntilEmptyStopsWhenPageMakesNoProgress(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}
		if err := json.NewEncoder(w).Encode([]int{1}); err != nil {
			t.Fatal(err)
		}
	})

	items, err := GetPaginatedUntilEmptyWithPageSize[int](client, 2, 1, func(item int) string {
		return fmt.Sprint(item)
	}, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err == nil || err.Error() != "pagination made no progress on page 2: received 1 items but found no new unique items" {
		t.Fatalf("error = %v, want no-progress error", err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil on incomplete pagination", items)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestGetPaginatedUntilEmptyErrorsAtMaximumPageBeforeLimit(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewEncoder(w).Encode([]int{requests}); err != nil {
			t.Fatal(err)
		}
	})

	items, err := GetPaginatedUntilEmptyWithPageSize[int](client, maxUntilEmptyRequestPages+1, 1, func(item int) string {
		return fmt.Sprint(item)
	}, func(page, perPage int) string {
		return fmt.Sprintf("/resources?page=%d&per_page=%d", page, perPage)
	})
	if err == nil || err.Error() != "pagination reached the maximum of 100 pages with 100 unique items, fewer than requested limit 101" {
		t.Fatalf("error = %v, want maximum-page error", err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil on incomplete pagination", items)
	}
	if requests != maxUntilEmptyRequestPages {
		t.Fatalf("requests = %d, want %d", requests, maxUntilEmptyRequestPages)
	}
}

func TestGetPaginatedWithPageSizeRejectsInvalidPageSize(t *testing.T) {
	for _, pageSize := range []int{0, -1} {
		t.Run(fmt.Sprint(pageSize), func(t *testing.T) {
			if _, err := GetPaginatedWithPageSize[int](NewClient("token"), 1, pageSize, nil); err == nil {
				t.Fatal("expected invalid maximum page size error")
			}
		})
	}
}
