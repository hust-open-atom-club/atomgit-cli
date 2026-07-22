package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type apiTestConfig struct {
	token      string
	tokenErr   error
	tokenReads int
}

func (c *apiTestConfig) GetToken() (string, error) {
	c.tokenReads++
	return c.token, c.tokenErr
}
func (c *apiTestConfig) GetUser() (string, error) { return "tester", nil }
func (c *apiTestConfig) GetHost() string          { return "atomgit.com" }

type apiRoundTripFunc func(*http.Request) (*http.Response, error)

func (f apiRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type apiTestFixture struct {
	config      *apiTestConfig
	factory     *cmdutil.Factory
	clientReads int
	requests    int
	stdin       *bytes.Buffer
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

func newAPITestFixture(token string, transport apiRoundTripFunc) *apiTestFixture {
	fixture := &apiTestFixture{
		config: &apiTestConfig{token: token},
		stdin:  &bytes.Buffer{},
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	fixture.factory = &cmdutil.Factory{Config: fixture.config}
	fixture.factory.HttpClient = func() (*http.Client, error) {
		fixture.clientReads++
		return &http.Client{Transport: apiRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			fixture.requests++
			return transport(req)
		})}, nil
	}
	return fixture
}

func apiTestResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func assertNoTokenLeak(t *testing.T, token string, values ...string) {
	t.Helper()
	for _, value := range values {
		if token != "" && strings.Contains(value, token) {
			t.Fatalf("token leaked in %q", value)
		}
	}
}
