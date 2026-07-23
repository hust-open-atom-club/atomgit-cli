package key

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type sshKeyTestConfig struct {
	token    string
	tokenErr error
}

func (c sshKeyTestConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c sshKeyTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c sshKeyTestConfig) GetHost() string           { return "atomgit.com" }

type sshKeyRoundTripFunc func(*http.Request) (*http.Response, error)

func (f sshKeyRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func sshKeyFactory(config sshKeyTestConfig, transport sshKeyRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func sshKeyResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
