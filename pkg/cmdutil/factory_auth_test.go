package cmdutil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

type tokenConfig struct {
	token string
	err   error
}

func (c tokenConfig) GetToken() (string, error) { return c.token, c.err }
func (tokenConfig) GetUser() (string, error)    { return "alice", nil }
func (tokenConfig) GetHost() string             { return "atomgit.com" }

type factoryRoundTripFunc func(*http.Request) (*http.Response, error)

func (f factoryRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAuthenticatedAPIClientRequiresCredentials(t *testing.T) {
	tests := []struct {
		name      string
		config    tokenConfig
		wantIs    error
		wantError string
	}{
		{
			name:      "missing credentials",
			config:    tokenConfig{err: config.ErrNotAuthenticated},
			wantIs:    config.ErrNotAuthenticated,
			wantError: config.ErrNotAuthenticated.Error(),
		},
		{
			name:      "wrapped missing credentials",
			config:    tokenConfig{err: fmt.Errorf("load token: %w", config.ErrNotAuthenticated)},
			wantIs:    config.ErrNotAuthenticated,
			wantError: config.ErrNotAuthenticated.Error(),
		},
		{
			name:      "empty token",
			config:    tokenConfig{token: "  "},
			wantIs:    config.ErrNotAuthenticated,
			wantError: config.ErrNotAuthenticated.Error(),
		},
		{
			name:      "credential store failure",
			config:    tokenConfig{err: errors.New("cannot read token file: permission denied")},
			wantIs:    nil,
			wantError: "not authenticated: cannot read token file: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &Factory{Config: tt.config}
			_, err := factory.AuthenticatedAPIClient()
			if err == nil {
				t.Fatal("expected authentication error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("error = %v, want %v", err, tt.wantIs)
			}
			if err.Error() != tt.wantError {
				t.Fatalf("error = %q, want %q", err, tt.wantError)
			}
		})
	}
}

func TestOptionalAPIClientFallsBackOnlyForMissingLogin(t *testing.T) {
	requests := 0
	factory := &Factory{
		Config: tokenConfig{err: fmt.Errorf("wrapped: %w", config.ErrNotAuthenticated)},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.Header.Get("Authorization"); got != "" {
					t.Fatalf("Authorization = %q, want empty", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			})}, nil
		},
	}

	client, err := factory.OptionalAPIClient()
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := client.Get("/user", &payload); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestOptionalAPIClientDoesNotFallBackOnStoreFailure(t *testing.T) {
	wantErr := errors.New("cannot read token file: permission denied")
	factory := &Factory{Config: tokenConfig{err: wantErr}}
	_, err := factory.OptionalAPIClient()
	if err == nil || !errors.Is(err, wantErr) || err.Error() != "not authenticated: cannot read token file: permission denied" {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthenticatedAPIClientUsesInjectedTransport(t *testing.T) {
	requests := 0
	factory := &Factory{
		Config: tokenConfig{token: "secret"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.Header.Get("Authorization"); got != "Bearer secret" {
					t.Fatalf("Authorization = %q", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"login":"alice"}`)),
				}, nil
			})}, nil
		},
	}

	client, err := factory.AuthenticatedAPIClient()
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := client.Get("/user", &payload); err != nil {
		t.Fatal(err)
	}
	if payload["login"] != "alice" || requests != 1 {
		t.Fatalf("payload = %#v, requests = %d", payload, requests)
	}
}

func TestAuthenticatedActionsClientRequiresCredentials(t *testing.T) {
	factory := &Factory{Config: tokenConfig{err: config.ErrNotAuthenticated}}
	_, err := factory.AuthenticatedActionsClient()
	if !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("error = %v", err)
	}
}

func TestFactoryAuthHelpersRequireConfig(t *testing.T) {
	factory := &Factory{}
	if _, err := factory.AuthenticatedAPIClient(); err == nil || !strings.Contains(err.Error(), "configuration is unavailable") {
		t.Fatalf("AuthenticatedAPIClient error = %v", err)
	}
	if _, err := factory.OptionalAPIClient(); err == nil || !strings.Contains(err.Error(), "configuration is unavailable") {
		t.Fatalf("OptionalAPIClient error = %v", err)
	}
	if _, err := factory.AuthenticatedActionsClient(); err == nil || !strings.Contains(err.Error(), "configuration is unavailable") {
		t.Fatalf("AuthenticatedActionsClient error = %v", err)
	}
}

func TestAuthenticatedAPIClientReportsHTTPClientErrors(t *testing.T) {
	want := errors.New("dial failed")
	factory := &Factory{
		Config: tokenConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return nil, want
		},
	}
	_, err := factory.AuthenticatedAPIClient()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}
