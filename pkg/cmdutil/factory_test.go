package cmdutil

import (
	"net/http"
	"testing"
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
