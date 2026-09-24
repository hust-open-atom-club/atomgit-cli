package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type identityTransport func(*http.Request) (*http.Response, error)

func (f identityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCurrentUserLoginRejectsRedirects(t *testing.T) {
	for _, target := range []string{
		"https://api.atomgit.com/api/v5/other",
		"https://other.example/user",
		"http://api.atomgit.com/api/v5/user",
	} {
		t.Run(target, func(t *testing.T) {
			requests, redirects := 0, 0
			httpClient := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
				Transport: identityTransport(func(req *http.Request) (*http.Response, error) {
					requests++
					if requests == 1 {
						return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{target}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"login":"alice"}`)), Request: req}, nil
				}),
			}
			login, err := NewClientWithHTTPClient("synthetic-access", httpClient).CurrentUserLogin()
			var httpErr *HTTPError
			if login != "" || !errors.As(err, &httpErr) || httpErr.StatusCode != 302 || requests != 1 || redirects != 0 {
				t.Fatalf("login=%q err=%v requests=%d redirects=%d", login, err, requests, redirects)
			}
			// The verification policy must not mutate the shared HTTP client.
			if err := httpClient.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatal("shared redirect policy changed")
			}
		})
	}
}
