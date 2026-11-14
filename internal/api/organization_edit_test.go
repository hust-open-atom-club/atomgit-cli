package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestEditOrganizationRequestAndContext(t *testing.T) {
	name, empty, website := "Display Name", "", "www.example.com"
	request := OrganizationEditRequest{Name: &name, Description: &empty, Email: &empty, Location: &empty, Website: &website}
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "org-edit")
	calls := 0
	client := NewClientWithBaseURL("fixture", "https://example.test/api/v5", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "PATCH" || r.URL.EscapedPath() != "/api/v5/orgs/team%20space" || r.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if r.Context().Value(contextKey{}) != "org-edit" {
			t.Fatal("lost request context")
		}
		if r.Header.Get("Authorization") != "Bearer fixture" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected headers")
		}
		var got map[string]string
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"name": name, "description": "", "email": "", "location": "", "html_url": website}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("body=%v want=%v", got, want)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":42,"path":"team space"}`)), Request: r}, nil
	})})
	got, err := EditOrganization(client.WithContext(ctx), "team space", request)
	if err != nil || calls != 1 || got.ID != 42 || got.Path != "team space" {
		t.Fatalf("result=%+v error=%v calls=%d", got, err, calls)
	}
}

func TestEditOrganizationDoesNotReplay(t *testing.T) {
	description := "new"
	for _, tc := range []struct {
		status    int
		uncertain bool
	}{
		{201, true}, {204, true}, {301, true}, {302, true}, {303, true}, {307, true}, {308, true},
		{400, false}, {401, false}, {403, false}, {404, false}, {422, false}, {429, false}, {500, true}, {502, true}, {503, true},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			calls, redirects := 0, 0
			original := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("PATCH replayed")
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": {"https://other.test/profile"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"private":"fixture"}`)), Request: r}, nil
			})}
			_, err := EditOrganization(NewClientWithBaseURL("fixture", "https://example.test/api/v5", original), "team", OrganizationEditRequest{Description: &description})
			var uncertain *OrganizationEditUncertainError
			if err == nil || !IsHTTPStatus(err, tc.status) || errors.As(err, &uncertain) != tc.uncertain || calls != 1 || redirects != 0 {
				t.Fatalf("error=%v calls=%d redirects=%d", err, calls, redirects)
			}
			_ = original.CheckRedirect(nil, nil)
			if redirects != 1 {
				t.Fatal("modified the caller's redirect policy")
			}
		})
	}
}

func TestEditOrganizationResponseIdentity(t *testing.T) {
	value := "new"
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"identity only", `{"id":42,"path":"team"}`, true},
		{"extra profile fields", `{"id":42,"path":"team","name":"server value","html_url":"www.example.com"}`, true},
		{"null", `null`, false}, {"empty object", `{}`, false}, {"zero ID", `{"id":0,"path":"team"}`, false},
		{"mismatched path", `{"id":42,"path":"other"}`, false}, {"missing path", `{"id":42}`, false},
		{"malformed", `{"id":`, false}, {"empty body", ``, false}, {"wrong ID type", `{"id":"42","path":"team"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, tc.body) })
			_, err := EditOrganization(client, "team", OrganizationEditRequest{Name: &value})
			if (err == nil) != tc.valid {
				t.Fatalf("error=%v", err)
			}
			if !tc.valid {
				var uncertain *OrganizationEditUncertainError
				if !errors.As(err, &uncertain) || !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("lost uncertain invalid response: %v", err)
				}
			}
		})
	}
}

func TestEditOrganizationPreservesExecutionAndReadFailures(t *testing.T) {
	value := "new"
	cause := errors.New("private failure")
	for _, readFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "execution", true: "read"}[readFailure], func(t *testing.T) {
			calls := 0
			c := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if !readFailure {
					return nil, cause
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(apiErrorReader{err: cause}), Request: r}, nil
			})})
			_, err := EditOrganization(c, "team", OrganizationEditRequest{Description: &value})
			var uncertain *OrganizationEditUncertainError
			if !errors.As(err, &uncertain) || !errors.Is(err, cause) || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
}

func TestEditOrganizationValidationBeforeRequest(t *testing.T) {
	empty, blank := "", " \n"
	c := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("sent invalid request"); return nil, nil })})
	for _, r := range []OrganizationEditRequest{{}, {Name: &empty}, {Name: &blank}} {
		if _, err := EditOrganization(c, "team", r); err == nil {
			t.Fatal("accepted invalid request")
		}
	}
	if _, err := EditOrganization(c, "", OrganizationEditRequest{Description: &empty}); err == nil {
		t.Fatal("accepted empty organization")
	}
}
