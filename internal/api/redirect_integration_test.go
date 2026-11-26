package api_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type boundaryTransport func(*http.Request) (*http.Response, error)

func (t boundaryTransport) RoundTrip(req *http.Request) (*http.Response, error) { return t(req) }

func TestFactoryClientsEnforceRedirectBoundary(t *testing.T) {
	for _, version := range []string{"v5", "v8"} {
		for _, target := range []string{"https://api.example.test:443/end", "https://api.example.test:8443/end", "https://other.example.test/end", "http://api.example.test/end"} {
			t.Run(version+"/"+target, func(t *testing.T) {
				calls, redirected := 0, 0
				f := &cmdutil.Factory{BaseURL: "https://api.example.test", HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: boundaryTransport(func(req *http.Request) (*http.Response, error) {
						calls++
						status := 200
						header := make(http.Header)
						if req.URL.Path != "/end" {
							status = 302
							header.Set("Location", target)
						} else {
							redirected++
							if got := req.Header.Get("Authorization") != ""; got != (target == "https://api.example.test:443/end") {
								t.Errorf("redirect auth=%t", got)
							}
						}
						return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
					})}, nil
				}}
				var err error
				if version == "v5" {
					c, e := f.NewAPIClient("fixture")
					if e != nil {
						t.Fatal(e)
					}
					var result map[string]any
					err = c.Get("/probe", &result)
				} else {
					c, e := f.NewActionsClient("fixture")
					if e != nil {
						t.Fatal(e)
					}
					_, err = c.ListRuns("team", "demo", actions.ListRunsOptions{})
				}
				if strings.HasPrefix(target, "http:") {
					if err == nil || redirected != 0 {
						t.Fatalf("downgrade error=%v requests=%d", err, calls)
					}
				} else if err != nil || redirected != 1 || calls != 2 {
					t.Fatalf("error=%v calls=%d redirected=%d", err, calls, redirected)
				}
			})
		}
	}
}
