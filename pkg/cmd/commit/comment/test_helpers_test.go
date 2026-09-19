package comment

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type testConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *testConfig) GetToken() (string, error) {
	c.tokenCalls++
	return "token", c.tokenErr
}

func (*testConfig) GetUser() (string, error) { return "alice", nil }
func (*testConfig) GetHost() string          { return "atomgit.com" }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newFactory(t *testing.T, cfg *testConfig, transport http.RoundTripper) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{Config: cfg, HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: transport}, nil
	}}
}

func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

var errNoRequests = errors.New("unexpected HTTP request")
