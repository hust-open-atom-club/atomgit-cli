package cmdutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
)

type stubConfig struct{}

func (stubConfig) GetToken() (string, error) { return "token", nil }
func (stubConfig) GetUser() (string, error)  { return "alice", nil }
func (stubConfig) GetHost() string           { return "atomgit.com" }

func TestFactoryStoresDependencies(t *testing.T) {
	wantClient := &http.Client{}
	factory := &Factory{
		Config: stubConfig{},
		HttpClient: func() (*http.Client, error) {
			return wantClient, nil
		},
	}

	if token, err := factory.Config.GetToken(); err != nil || token != "token" {
		t.Fatalf("GetToken() = %q, %v", token, err)
	}
	if user, err := factory.Config.GetUser(); err != nil || user != "alice" {
		t.Fatalf("GetUser() = %q, %v", user, err)
	}
	if host := factory.Config.GetHost(); host != "atomgit.com" {
		t.Fatalf("GetHost() = %q", host)
	}
	client, err := factory.HttpClient()
	if err != nil {
		t.Fatal(err)
	}
	if client != wantClient {
		t.Fatal("HttpClient returned a different client")
	}
}

func TestFactoryTokenForCommand(t *testing.T) {
	tests := []struct {
		name     string
		factory  *Factory
		fallback string
		want     string
		override bool
	}{
		{name: "nil factory", fallback: "stored-token", want: "stored-token"},
		{name: "no override", factory: &Factory{}, fallback: "stored-token", want: "stored-token"},
		{name: "blank override", factory: &Factory{TokenOverride: "  "}, fallback: "stored-token", want: "stored-token"},
		{name: "explicit blank override", factory: &Factory{TokenOverride: "  ", TokenOverrideSet: true}, fallback: "stored-token", want: "", override: true},
		{name: "command override", factory: &Factory{TokenOverride: "command-token"}, fallback: "stored-token", want: "command-token", override: true},
		{name: "trimmed command override", factory: &Factory{TokenOverride: "  command-token\n"}, fallback: "stored-token", want: "command-token", override: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, overridden := tt.factory.TokenForCommand(tt.fallback)
			if got != tt.want || overridden != tt.override {
				t.Fatalf("TokenForCommand() = %q, %t; want %q, %t", got, overridden, tt.want, tt.override)
			}
		})
	}
}

func TestFactoryOverridePresence(t *testing.T) {
	tests := []struct {
		name    string
		factory *Factory
		token   bool
		baseURL bool
	}{
		{name: "defaults", factory: &Factory{BaseURL: "https://api.atomgit.com"}},
		{name: "normalized default", factory: &Factory{BaseURL: "https://api.atomgit.com/"}},
		{name: "explicit empty token", factory: &Factory{BaseURL: "https://api.atomgit.com", TokenOverrideSet: true}, token: true},
		{name: "direct token", factory: &Factory{BaseURL: "https://api.atomgit.com", TokenOverride: "token"}, token: true},
		{name: "explicit default base URL", factory: &Factory{BaseURL: "https://api.atomgit.com", BaseURLOverrideSet: true}, baseURL: true},
		{name: "direct custom base URL", factory: &Factory{BaseURL: "https://api.gitcode.com"}, baseURL: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.factory.HasTokenOverride(); got != tt.token {
				t.Fatalf("HasTokenOverride() = %t, want %t", got, tt.token)
			}
			if got := tt.factory.HasBaseURLOverride(); got != tt.baseURL {
				t.Fatalf("HasBaseURLOverride() = %t, want %t", got, tt.baseURL)
			}
		})
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		value string
		want  string
		err   bool
	}{
		{value: "", err: true},
		{value: "https://api.atomgit.com", want: "https://api.atomgit.com"},
		{value: " https://api.gitcode.com/ ", want: "https://api.gitcode.com"},
		{value: "http://localhost:8080", want: "http://localhost:8080"},
		{value: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{value: "http://[::1]:8080", want: "http://[::1]:8080"},
		{value: "http://api.gitcode.com", err: true},
		{value: "https://api.atomgit.com/api/v5", err: true},
		{value: "https://user@api.atomgit.com", err: true},
		{value: "https://api.gitcode.com?x=1", err: true},
		{value: "https://api.gitcode.com#fragment", err: true},
		{value: "file:///tmp/service", err: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := NormalizeBaseURL(tt.value)
			if (err != nil) != tt.err || got != tt.want {
				t.Fatalf("NormalizeBaseURL(%q) = %q, %v", tt.value, got, err)
			}
		})
	}
}

func TestFactoryBaseURLSelectsAPIVersion(t *testing.T) {
	var requestURLs []string
	factory := &Factory{
		BaseURL: "https://api.gitcode.com/",
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requestURLs = append(requestURLs, req.URL.String())
				body := `{}`
				if strings.Contains(req.URL.Path, "/actions/workflows") {
					body = `{"workflows":[]}`
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}, nil
		},
	}

	apiClient, err := factory.NewAPIClient("secret")
	if err != nil {
		t.Fatal(err)
	}
	var user map[string]any
	if err := apiClient.Get("/user", &user); err != nil {
		t.Fatal(err)
	}
	actionsClient, err := factory.NewActionsClient("secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actionsClient.ListWorkflows("owner", "repo", actions.ListWorkflowsOptions{Page: 1, PerPage: 1}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"https://api.gitcode.com/api/v5/user",
		"https://api.gitcode.com/api/v8/repos/owner/repo/actions/workflows?page=1&per_page=1",
	}
	if len(requestURLs) != len(want) || requestURLs[0] != want[0] || requestURLs[1] != want[1] {
		t.Fatalf("request URLs = %#v, want %#v", requestURLs, want)
	}
}

func TestApplyTokenOverrideResolvesUserAgainstSelectedOrigin(t *testing.T) {
	requests := 0
	factory := &Factory{
		BaseURL:       "https://api.gitcode.com",
		Config:        stubConfig{},
		TokenOverride: "command-token",
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got, want := req.URL.String(), "https://api.gitcode.com/api/v5/user"; got != want {
					t.Fatalf("request URL = %q, want %q", got, want)
				}
				if got, want := req.Header.Get("Authorization"), "Bearer command-token"; got != want {
					t.Fatalf("Authorization = %q, want %q", got, want)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"login":"bob"}`)),
					Request:    req,
				}, nil
			})}, nil
		},
	}

	factory.ApplyTokenOverride()
	factory.ApplyTokenOverride()
	for range 2 {
		user, err := factory.Config.GetUser()
		if err != nil || user != "bob" {
			t.Fatalf("GetUser() = %q, %v", user, err)
		}
	}
	if requests != 1 {
		t.Fatalf("identity requests = %d, want 1", requests)
	}
}

func TestApplyTokenOverrideDoesNotFallBackToStoredUser(t *testing.T) {
	const commandToken = "private-command-token"
	factory := &Factory{
		BaseURL:       "https://api.gitcode.com",
		Config:        stubConfig{},
		TokenOverride: commandToken,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"message":"denied"}`)),
					Request:    req,
				}, nil
			})}, nil
		},
	}

	factory.ApplyTokenOverride()
	user, err := factory.Config.GetUser()
	if err == nil || user != "" {
		t.Fatalf("GetUser() = %q, %v; want verification failure", user, err)
	}
	if strings.Contains(err.Error(), commandToken) {
		t.Fatalf("GetUser() error leaked the command token: %v", err)
	}
}
