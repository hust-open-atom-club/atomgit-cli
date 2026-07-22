package api

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestPrepareRejectsInvalidRequestBeforeDependencies(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		opts     options
		want     string
	}{
		{name: "unsupported method", endpoint: "/user", opts: options{method: "TRACE", accept: "application/json"}, want: "unsupported HTTP method"},
		{name: "absolute URL", endpoint: "https://api.atomgit.com/api/v5/user", opts: options{method: "GET", accept: "application/json"}, want: "relative API v5"},
		{name: "protocol relative URL", endpoint: "//evil.test/user", opts: options{method: "GET", accept: "application/json"}, want: "relative API v5"},
		{name: "fragment", endpoint: "/user#secret", opts: options{method: "GET", accept: "application/json"}, want: "fragment"},
		{name: "malformed escape", endpoint: "/bad%zz", opts: options{method: "GET", accept: "application/json"}, want: "invalid endpoint"},
		{name: "empty path", endpoint: "?page=1", opts: options{method: "GET", accept: "application/json"}, want: "relative API v5"},
		{name: "parent traversal", endpoint: "/../user", opts: options{method: "GET", accept: "application/json"}, want: "API v5 base"},
		{name: "escaped parent traversal", endpoint: "/%2e%2e/user", opts: options{method: "GET", accept: "application/json"}, want: "API v5 base"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := prepare(tt.endpoint, tt.opts, bytes.NewBuffer(nil))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("prepare() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvalidCommandHasNoAuthenticationOrRequest(t *testing.T) {
	fixture := newAPITestFixture("secret", func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected request")
		return nil, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetArgs([]string{"https://evil.test/user"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
	if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
		t.Fatalf("side effects = token %d client %d requests %d", fixture.config.tokenReads, fixture.clientReads, fixture.requests)
	}
}

func TestAPIRequiresExactlyOneEndpoint(t *testing.T) {
	for _, args := range [][]string{nil, {"/one", "/two"}} {
		cmd := NewCmdAPI(newAPITestFixture("secret", nil).factory)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("args %q unexpectedly succeeded", args)
		}
	}
}
