package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	internalapi "atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

func TestAPIPaginationWithTotalPages(t *testing.T) {
	requests := 0
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		requests++
		query := req.URL.Query()
		if query.Get("page") != strconv.Itoa(requests) || query.Get("per_page") != "2" || query.Get("state") != "open" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		resp := apiTestResponse(req, http.StatusOK, fmt.Sprintf(" [ %d ] ", requests))
		resp.Header.Set("total_page", "3")
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/items?state=open&per_page=2", "--paginate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 3 || fixture.stdout.String() != "[1]\n[2]\n[3]\n" {
		t.Fatalf("requests = %d stdout = %q", requests, fixture.stdout.String())
	}
}

func TestAPIPaginationArrayFallback(t *testing.T) {
	pages := []string{`[{"id":1},{"id":2}]`, `[{"id":3}]`}
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		return apiTestResponse(req, http.StatusOK, pages[page-1]), nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/items?per_page=2", "--paginate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fixture.stdout.String() != "[{\"id\":1},{\"id\":2}]\n[{\"id\":3}]\n" {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
}

func TestAPIPaginationCompactsObjectWithMetadata(t *testing.T) {
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		resp := apiTestResponse(req, http.StatusOK, " { \"items\" : [1] } ")
		resp.Header.Set("total_page", "1")
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/items", "--paginate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fixture.stdout.String() != "{\"items\":[1]}\n" {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
}

func TestAPIPaginationStopsOnEmptyFirstPage(t *testing.T) {
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		return apiTestResponse(req, http.StatusOK, "[]"), nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/items", "--paginate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fixture.requests != 1 || fixture.stdout.String() != "[]\n" {
		t.Fatalf("requests = %d stdout = %q", fixture.requests, fixture.stdout.String())
	}
}

func TestPaginationRejectsPreviouslyVisitedRequest(t *testing.T) {
	request, err := prepare("/items", options{method: "GET", accept: "application/json", paginate: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.pagination.visited[request.path] = true
	client := internalapi.NewClientWithHTTPClient("secret", &http.Client{Transport: apiRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected repeated request")
		return nil, nil
	})})
	err = executePagination(io.Discard, client, request, "secret")
	if err == nil || !strings.Contains(err.Error(), "repeated request") {
		t.Fatalf("error = %v", err)
	}
}

func TestPaginationFailureDoesNotEmitCurrentPage(t *testing.T) {
	tests := []struct {
		name string
		fail func(*http.Request) (*http.Response, error)
	}{
		{name: "status", fail: func(req *http.Request) (*http.Response, error) {
			return apiTestResponse(req, http.StatusInternalServerError, "partial"), nil
		}},
		{name: "transport", fail: func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") }},
		{name: "read", fail: func(req *http.Request) (*http.Response, error) {
			resp := apiTestResponse(req, http.StatusOK, "")
			resp.Body = failingReader{err: errors.New("read")}
			return resp, nil
		}},
		{name: "JSON", fail: func(req *http.Request) (*http.Response, error) { return apiTestResponse(req, http.StatusOK, "{"), nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					resp := apiTestResponse(req, http.StatusOK, `[1]`)
					resp.Header.Set("total_page", "2")
					return resp, nil
				}
				return tt.fail(req)
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs([]string{"/items", "--paginate"})
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
			wantRequests := 2
			if tt.name == "transport" {
				wantRequests = 3
			}
			if fixture.stdout.String() != "[1]\n" || requests != wantRequests {
				t.Fatalf("requests = %d stdout = %q", requests, fixture.stdout.String())
			}
		})
	}
}

func TestPaginationRejectsInconsistentMetadataAndNonArray(t *testing.T) {
	for _, test := range []struct {
		name     string
		response func(int, *http.Request) *http.Response
	}{
		{name: "inconsistent", response: func(page int, req *http.Request) *http.Response {
			resp := apiTestResponse(req, http.StatusOK, `[]`)
			resp.Header.Set("total_page", strconv.Itoa(3-page))
			return resp
		}},
		{name: "non array", response: func(_ int, req *http.Request) *http.Response { return apiTestResponse(req, http.StatusOK, `{}`) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := 0
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) { page++; return test.response(page, req), nil })
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs([]string{"/items?per_page=1", "--paginate"})
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPaginationSupportsOneThousandItems(t *testing.T) {
	requests := 0
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		requests++
		items := make([]string, 100)
		for index := range items {
			items[index] = strconv.Itoa((requests-1)*100 + index)
		}
		resp := apiTestResponse(req, http.StatusOK, "["+strings.Join(items, ",")+"]")
		resp.Header.Set("total_page", "10")
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"/items", "--paginate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 10 {
		t.Fatalf("requests = %d", requests)
	}
}
