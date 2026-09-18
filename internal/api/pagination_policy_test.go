package api

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGetPaginatedWithPolicyRejectsUnlistedStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `[1]`)
	})

	_, err := getPaginatedWithPolicy[int](client, 10, RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        true,
	}, func(page, perPage int) string {
		return pageQuery("/resources", page, perPage)
	})
	if !IsHTTPStatus(err, http.StatusCreated) {
		t.Fatalf("error = %v, want HTTP 201", err)
	}
}

func TestGetPaginatedWithPolicyDecodesExactSuccess(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `[1,2]`)
	})

	items, err := getPaginatedWithPolicy[int](client, 10, RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        true,
	}, func(page, perPage int) string {
		return pageQuery("/resources", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || fmt.Sprint(items) != "[1 2]" {
		t.Fatalf("items = %#v, requests = %d", items, requests)
	}
}
