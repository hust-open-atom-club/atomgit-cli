package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

func TestRedirectCredentialBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, start     string
		targets         []string
		wantCredentials []bool
		wantError       bool
	}{
		{"same origin", "https://api.example.test/start", []string{"https://api.example.test/end"}, []bool{true, true}, false},
		{"default port", "https://api.example.test/start", []string{"https://api.example.test:443/end"}, []bool{true, true}, false},
		{"port change", "https://api.example.test/start", []string{"https://api.example.test:8443/end"}, []bool{true, false}, false},
		{"subdomain", "https://api.example.test/start", []string{"https://sub.api.example.test/end"}, []bool{true, false}, false},
		{"host change", "https://api.example.test/start", []string{"https://other.example.test/end"}, []bool{true, false}, false},
		{"redirect user info", "https://api.example.test/start", []string{"https://user:fixture@other.example.test/end"}, []bool{true, false}, false},
		{"return to origin", "https://api.example.test/start", []string{"https://other.example.test/away", "https://api.example.test/back"}, []bool{true, false, false}, false},
		{"return after port change", "https://api.example.test/start", []string{"https://api.example.test:8443/away", "https://api.example.test/back"}, []bool{true, false, false}, false},
		{"downgrade", "https://api.example.test/start", []string{"http://api.example.test/end"}, []bool{true}, true},
		{"loopback downgrade", "https://localhost/start", []string{"http://localhost/end"}, []bool{true}, true},
		{"http loopback", "http://localhost:8080/start", []string{"http://localhost:8080/end"}, []bool{true, true}, false},
		{"loopback port change", "http://localhost:8080/start", []string{"http://localhost:8081/end"}, []bool{true, false}, false},
		{"loopback to remote http", "http://localhost:8080/start", []string{"http://remote.example.test/end"}, []bool{true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if calls >= len(tc.wantCredentials) {
					t.Fatal("unexpected redirect request")
				}
				for _, header := range []string{"Authorization", "Cookie", "Cookie2", "Proxy-Authorization"} {
					if got := r.Header.Get(header) != ""; got != tc.wantCredentials[calls] {
						t.Errorf("request %d %s present=%t", calls, header, got)
					}
				}
				resp := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}
				if calls < len(tc.targets) {
					resp.StatusCode = 302
					resp.Header.Set("Location", tc.targets[calls])
				}
				calls++
				return resp, nil
			})}
			req, err := http.NewRequest("GET", tc.start, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, h := range []string{"Authorization", "Cookie", "Cookie2", "Proxy-Authorization"} {
				req.Header.Set(h, "fake-credential")
			}
			resp, err := RedirectSafeHTTPClient(hc).Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if (err != nil) != tc.wantError || calls != len(tc.wantCredentials) {
				t.Fatalf("error=%v requests=%d", err, calls)
			}
			if hc.CheckRedirect != nil {
				t.Fatal("modified caller client")
			}
		})
	}
}

func TestRedirectOriginComparison(t *testing.T) {
	for _, tc := range []struct {
		target string
		same   bool
	}{
		{"https://API.example.test:443/end", true},
		{"https://api.example.test:0443/end", true},
		{"https://api.example.test:8443/end", false},
		{"https://sub.api.example.test/end", false},
		{"http://api.example.test:443/end", false},
	} {
		origin, err := url.Parse("https://api.example.test/start")
		if err != nil {
			t.Fatal(err)
		}
		target, err := url.Parse(tc.target)
		if err != nil {
			t.Fatal(err)
		}
		if got := redirectSameOrigin(origin, target); got != tc.same {
			t.Fatalf("target=%s same=%t", tc.target, got)
		}
	}
}

func TestRedirectRechecksCallbackDestination(t *testing.T) {
	calls := 0
	hc := &http.Client{CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		changed, err := url.Parse("http://api.example.test/changed")
		if err != nil {
			return err
		}
		req.URL = changed
		req.Header.Set("Authorization", "restored")
		return nil
	}, Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"https://api.example.test/same"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})}
	resp, err := RedirectSafeHTTPClient(hc).Get("https://api.example.test/start")
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil || calls != 1 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestRedirectStripsJarCookiesAndCallbackHeaders(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	origin, err := url.Parse("https://api.example.test/start")
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(origin, []*http.Cookie{{Name: "session", Value: "fixture", Domain: "example.test", Path: "/"}})
	calls, callbacks := 0, 0
	hc := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		callbacks++
		if req.Header.Get("Authorization") != "" {
			t.Error("callback saw cross-origin authorization")
		}
		req.Header.Set("Authorization", "reintroduced")
		req.Header.Set("Cookie", "reintroduced=1")
		req.Header["proxy-authorization"] = []string{"reintroduced"}
		return nil
	}, Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if calls == 0 && req.Header.Get("Cookie") == "" {
			t.Error("same-origin jar cookie missing")
		}
		if calls > 0 {
			for h := range req.Header {
				if strings.EqualFold(h, "Authorization") || strings.EqualFold(h, "Cookie") || strings.EqualFold(h, "Proxy-Authorization") {
					t.Errorf("redirect credential header survived: %s", h)
				}
			}
		}
		resp := &http.Response{StatusCode: 302, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}
		switch calls {
		case 0:
			resp.Header.Set("Location", "https://other.example.test/away")
		case 1:
			resp.Header.Set("Location", origin.String())
		default:
			resp.StatusCode = 200
		}
		calls++
		return resp, nil
	})}
	req, _ := http.NewRequest("GET", origin.String(), nil)
	req.Header.Set("Authorization", "Bearer fixture")
	resp, err := RedirectSafeHTTPClient(hc).Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil || calls != 3 || callbacks != 2 {
		t.Fatalf("err=%v calls=%d callbacks=%d", err, calls, callbacks)
	}
	if hc.Jar != jar {
		t.Fatal("changed caller jar")
	}
}

func TestRedirectPreservesRefusalAndLimit(t *testing.T) {
	refused := errors.New("caller refused")
	wrappedLastResponse := fmt.Errorf("wrapped stop: %w", http.ErrUseLastResponse)
	for _, reason := range []error{refused, http.ErrUseLastResponse, wrappedLastResponse, nil} {
		calls := 0
		hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return reason }, Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"/loop"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		})}
		resp, err := RedirectSafeHTTPClient(hc).Get("https://api.example.test/loop")
		if resp != nil {
			_ = resp.Body.Close()
		}
		if reason == nil {
			if err == nil || calls != 10 {
				t.Fatalf("limit error=%v calls=%d", err, calls)
			}
		} else if reason == http.ErrUseLastResponse { //nolint:errorlint // net/http requires this exact sentinel.
			if err != nil || calls != 1 || resp.StatusCode != 302 {
				t.Fatalf("last response error=%v calls=%d", err, calls)
			}
		} else if !errors.Is(err, reason) || calls != 1 {
			t.Fatalf("refusal error=%v calls=%d", err, calls)
		}
	}
}
