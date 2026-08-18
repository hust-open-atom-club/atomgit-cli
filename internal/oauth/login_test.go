package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withDefaultHTTPClient(t *testing.T, fn roundTripFunc) {
	t.Helper()
	old := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: fn}
	t.Cleanup(func() { http.DefaultClient = old })
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestOAuthEnvironmentDefaultsAndOverrides(t *testing.T) {
	t.Setenv("AG_OAUTH_CLIENT_ID", "")
	t.Setenv("AG_OAUTH_CLIENT_SECRET", "")
	t.Setenv("AG_OAUTH_REDIRECT_PORT", "")
	if clientID() != defaultClientID || clientSecret() != defaultClientSecret || redirectPort() != defaultRedirectPort {
		t.Fatal("default OAuth settings were not used")
	}

	t.Setenv("AG_OAUTH_CLIENT_ID", " custom-id ")
	t.Setenv("AG_OAUTH_CLIENT_SECRET", " custom-secret ")
	t.Setenv("AG_OAUTH_REDIRECT_PORT", " 9999 ")
	if clientID() != "custom-id" || clientSecret() != "custom-secret" || redirectPort() != "9999" {
		t.Fatal("OAuth environment overrides were not trimmed")
	}
	if got := redirectURI(); got != "http://127.0.0.1:9999/callback" {
		t.Fatalf("redirectURI() = %q", got)
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	raw := buildAuthorizeURL("client", "http://127.0.0.1/callback", "state-value")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "https" || parsed.Host != "atomgit.com" || parsed.Path != "/oauth/authorize" {
		t.Fatalf("URL = %s", raw)
	}
	want := map[string]string{
		"client_id":     "client",
		"redirect_uri":  "http://127.0.0.1/callback",
		"response_type": "code",
		"state":         "state-value",
		"scope":         scopes,
	}
	for key, value := range want {
		if got := parsed.Query().Get(key); got != value {
			t.Fatalf("query %s = %q, want %q", key, got, value)
		}
	}
}

func TestRandomState(t *testing.T) {
	first, err := randomState()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomState()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 || len(second) != 32 {
		t.Fatalf("state lengths = %d, %d", len(first), len(second))
	}
	if first == second {
		t.Fatal("two random states unexpectedly matched")
	}
}

func TestExchangeCode(t *testing.T) {
	withDefaultHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != tokenURL {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q", got)
		}
		if err := req.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"client_id":     "id",
			"client_secret": "secret",
			"code":          "code",
			"redirect_uri":  "redirect",
			"grant_type":    "authorization_code",
		}
		for key, value := range want {
			if req.Form.Get(key) != value {
				t.Fatalf("form %s = %q", key, req.Form.Get(key))
			}
		}
		return response(http.StatusOK, `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":60}`), nil
	})

	got, err := exchangeCode(context.Background(), "id", "secret", "redirect", "code")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.ExpiresIn != 60 {
		t.Fatalf("token = %#v", got)
	}
}

func TestExchangeCodeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		want string
	}{
		{name: "endpoint", code: http.StatusUnauthorized, body: "invalid code", want: "401"},
		{name: "invalid JSON", code: http.StatusOK, body: "{", want: "parse token JSON"},
		{name: "missing token", code: http.StatusOK, body: `{}`, want: "missing access_token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDefaultHTTPClient(t, func(*http.Request) (*http.Response, error) {
				return response(tt.code, tt.body), nil
			})
			_, err := exchangeCode(context.Background(), "id", "secret", "redirect", "code")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRefreshAccessToken(t *testing.T) {
	t.Setenv("AG_OAUTH_CLIENT_ID", "id")
	t.Setenv("AG_OAUTH_CLIENT_SECRET", "secret")
	withDefaultHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if err := req.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if req.Form.Get("refresh_token") != "refresh" || req.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("form = %#v", req.Form)
		}
		return response(http.StatusOK, `{"access_token":"new","expires_in":120}`), nil
	})

	got, err := RefreshAccessToken(context.Background(), "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "new" || got.ExpiresIn != 120 || got.TokenType != "Bearer" {
		t.Fatalf("tokens = %#v", got)
	}
}

func TestRefreshAccessTokenRejectsEmptyToken(t *testing.T) {
	if _, err := RefreshAccessToken(context.Background(), "  "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v", err)
	}
}

func TestFetchUser(t *testing.T) {
	withDefaultHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != userURL {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer access" {
			t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
		}
		return response(http.StatusOK, `{"login":"alice","name":"Alice"}`), nil
	})

	got, err := FetchUser(context.Background(), "access")
	if err != nil {
		t.Fatal(err)
	}
	if got.Login != "alice" || got.Name != "Alice" {
		t.Fatalf("user = %#v", got)
	}
}

func TestFetchUserErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		want string
	}{
		{name: "endpoint", code: http.StatusUnauthorized, body: "denied", want: "401"},
		{name: "invalid JSON", code: http.StatusOK, body: "{", want: "parse user JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDefaultHTTPClient(t, func(*http.Request) (*http.Response, error) {
				return response(tt.code, tt.body), nil
			})
			_, err := FetchUser(context.Background(), "access")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestFetchUserWithURLAgainstTestServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer piped-token" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"login":"alice","name":"Alice","email":"alice@example.com"}`)
	}))
	t.Cleanup(server.Close)

	user, err := FetchUserWithURL(context.Background(), server.URL, "piped-token")
	if err != nil {
		t.Fatal(err)
	}
	if user.Login != "alice" || user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Fatalf("user = %#v", user)
	}

	if _, err := FetchUserWithURL(context.Background(), server.URL, "wrong-token"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v", err)
	}
}
