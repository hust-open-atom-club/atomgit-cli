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

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		value string
		want  string
		err   bool
	}{
		{value: "https://test.gitcode.net", want: "https://test.gitcode.net"},
		{value: " https://test.gitcode.net/ ", want: "https://test.gitcode.net"},
		{value: "http://localhost:8080", want: "http://localhost:8080"},
		{value: "https://test.gitcode.net/api/v5", err: true},
		{value: "https://user@test.gitcode.net", err: true},
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
	var paths []string
	factory := &Factory{
		BaseURL: "https://test.gitcode.net/",
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				paths = append(paths, req.URL.Path)
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

	if len(paths) != 2 || paths[0] != "/api/v5/user" || paths[1] != "/api/v8/repos/owner/repo/actions/workflows" {
		t.Fatalf("request paths = %#v", paths)
	}
}
