package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"
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

func TestGetPaginatedWithPolicyRetriesOnlyCurrentPage(t *testing.T) {
	firstPage := make([]int, 100)
	for i := range firstPage {
		firstPage[i] = i
	}
	firstPageJSON, err := json.Marshal(firstPage)
	if err != nil {
		t.Fatal(err)
	}

	var pages []string
	pageTwoAttempts := 0
	client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			page := req.URL.Query().Get("page")
			pages = append(pages, page)
			switch page {
			case "1":
				return runRawResponse(req, http.StatusOK, string(firstPageJSON)), nil
			case "2":
				pageTwoAttempts++
				if pageTwoAttempts == 1 {
					resp := runRawResponse(req, http.StatusTooManyRequests, "limited")
					resp.Header.Set("Retry-After", "0")
					return resp, nil
				}
				return runRawResponse(req, http.StatusOK, `[100]`), nil
			default:
				t.Fatalf("unexpected page %q", page)
				return nil, nil
			}
		}),
	})
	client.rateLimitPolicy = rateLimitRetryPolicy{
		budget:      time.Second,
		maxAttempts: 3,
		now:         time.Now,
		wait:        func(context.Context, time.Duration) error { return nil },
		jitter:      func(time.Duration) time.Duration { return 0 },
	}

	items, err := getPaginatedWithPolicy[int](client, 101, RequestPolicy{
		AllowedStatuses: []int{http.StatusOK},
		CanRetry:        true,
	}, func(page, perPage int) string {
		return pageQuery("/resources", page, perPage)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 101 || !reflect.DeepEqual(pages, []string{"1", "2", "2"}) {
		t.Fatalf("items = %d, requested pages = %v", len(items), pages)
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
