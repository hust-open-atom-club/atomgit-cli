package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreateOrganizationDoesNotReplay(t *testing.T) {
	transportErr := errors.New("connection lost")
	for _, tt := range []struct {
		name   string
		status int
		err    error
	}{
		{"transport", 0, transportErr},
		{"rate limit", 429, nil},
		{"server error", 503, nil},
		{"moved", 301, nil},
		{"found", 302, nil},
		{"see other", 303, nil},
		{"temporary redirect", 307, nil},
		{"permanent redirect", 308, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls, redirects := 0, 0
			httpClient := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						t.Fatal("creation replayed")
					}
					if tt.err != nil {
						return nil, tt.err
					}
					return &http.Response{
						StatusCode: tt.status,
						Header:     http.Header{"Location": {"https://other.test/create"}, "Retry-After": {"0"}},
						Body:       io.NopCloser(strings.NewReader(`{"message":"failure"}`)),
						Request:    req,
					}, nil
				}),
			}
			client := NewClientWithBaseURL("token", "https://example.test/api/v5", httpClient)
			_, err := CreateOrganization(client, OrganizationCreateRequest{Path: "team"})
			if err == nil || calls != 1 || redirects != 0 {
				t.Fatalf("error=%v calls=%d redirects=%d", err, calls, redirects)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Fatalf("lost transport error: %v", err)
			}
			if tt.err != nil && !strings.Contains(err.Error(), "creation may have succeeded") {
				t.Fatalf("missing uncertain write guidance: %v", err)
			}
			// The caller's redirect policy must remain intact.
			if err := httpClient.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatal("modified shared HTTP client")
			}
		})
	}
}

func TestCreateOrganizationResponseReadFailure(t *testing.T) {
	cause := errors.New("response interrupted")
	calls := 0
	client := NewClientWithHTTPClient("token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(apiErrorReader{err: cause})}, nil
	})})
	_, err := CreateOrganization(client, OrganizationCreateRequest{Path: "team"})
	var readErr *ResponseReadError
	if !errors.As(err, &readErr) || !errors.Is(err, cause) || calls != 1 || !strings.Contains(err.Error(), "creation may have succeeded") {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestCreateOrganizationContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "request")
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(contextKey{}) != "request" {
			t.Error("request context not propagated")
		}
		w.Write([]byte(`{"id":42,"path":"team","web_url":"https://example.test/team","visibility":"private"}`))
	})
	result, err := CreateOrganization(client.WithContext(ctx), OrganizationCreateRequest{Path: "team"})
	if err != nil || result.ID != 42 || result.Visibility != "private" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestCreateOrganizationInvalidIdentity(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"id":0,"path":"team"}`, `{"id":1,"path":"another"}`} {
		t.Run(body, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(body)) })
			_, err := CreateOrganization(client, OrganizationCreateRequest{Path: "team"})
			if !errors.Is(err, ErrInvalidResponse) || !strings.Contains(err.Error(), "creation may have succeeded") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
