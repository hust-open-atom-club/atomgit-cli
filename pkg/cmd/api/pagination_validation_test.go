package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestPaginationValidationHasNoSideEffects(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "non GET", args: []string{"/items", "--paginate", "-X", "POST"}},
		{name: "raw input", args: []string{"/items", "--paginate", "--input", "-"}},
		{name: "duplicate page", args: []string{"/items?page=1&page=2", "--paginate"}},
		{name: "duplicate per page", args: []string{"/items?per_page=1&per_page=2", "--paginate"}},
		{name: "zero page", args: []string{"/items?page=0", "--paginate"}},
		{name: "negative per page", args: []string{"/items?per_page=-1", "--paginate"}},
		{name: "non integer", args: []string{"/items?page=x", "--paginate"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newAPITestFixture("secret", func(*http.Request) (*http.Response, error) {
				t.Fatal("unexpected request")
				return nil, nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
			if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
				t.Fatal("validation caused side effects")
			}
		})
	}
}

func TestPaginationDefaultsAndPreservesQuery(t *testing.T) {
	request, err := prepare("/items?state=open&tag=a&tag=b", options{method: "GET", accept: "application/json", paginate: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.pagination.page != 1 || request.pagination.perPage != 100 {
		t.Fatalf("pagination = %#v", request.pagination)
	}
	parsed, err := url.Parse(request.path)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("state") != "open" || len(query["tag"]) != 2 || query.Get("page") != "1" || query.Get("per_page") != "100" {
		t.Fatalf("query = %#v", query)
	}
}
