package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }
func (r failingReader) Close() error             { return nil }

func TestAPIDefaultGETStreamsAllSuccessfulResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "OK", status: http.StatusOK, body: `{"ok":true}`},
		{name: "created binary", status: http.StatusCreated, body: "\x00\xff"},
		{name: "no content", status: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/user" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return apiTestResponse(req, tt.status, tt.body), nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetErr(fixture.stderr)
			cmd.SetArgs([]string{"user"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if got := fixture.stdout.String(); got != tt.body {
				t.Fatalf("stdout = %q", got)
			}
			assertNoTokenLeak(t, "secret", fixture.stdout.String(), fixture.stderr.String())
		})
	}
}

func TestAPIErrorsAreBoundedAndRedacted(t *testing.T) {
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		return apiTestResponse(req, http.StatusForbidden, `{"message":"denied secret`+strings.Repeat("x", maxErrorBody)+`"}`), nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/user"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "403 Forbidden") || len(err.Error()) > maxErrorBody+200 {
		t.Fatalf("error = %v", err)
	}
	if fixture.stdout.Len() != 0 {
		t.Fatalf("stdout = %q", fixture.stdout.String())
	}
	assertNoTokenLeak(t, "secret", err.Error())
}

func TestAPIReportsAuthTransportAndReadErrors(t *testing.T) {
	t.Run("auth", func(t *testing.T) {
		fixture := newAPITestFixture("secret", nil)
		cause := errors.New("credential secret unavailable")
		fixture.config.tokenErr = cause
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetArgs([]string{"/user"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not authenticated") {
			t.Fatalf("error = %v", err)
		}
		assertNoTokenLeak(t, "secret", err.Error())
		if !errors.Is(err, cause) {
			t.Fatalf("error does not preserve cause: %v", err)
		}
		if fixture.clientReads != 0 || fixture.requests != 0 {
			t.Fatal("authentication failure acquired client or sent request")
		}
	})
	t.Run("transport", func(t *testing.T) {
		fixture := newAPITestFixture("secret", func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		})
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetArgs([]string{"/user"})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "request GET") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("read", func(t *testing.T) {
		fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
			resp := apiTestResponse(req, http.StatusOK, "")
			resp.Body = failingReader{err: errors.New("read failed")}
			return resp, nil
		})
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetArgs([]string{"/user"})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "read response") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestAPINon2xxPreservesBodyReadError(t *testing.T) {
	cause := errors.New("read error body")
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		resp := apiTestResponse(req, http.StatusBadGateway, "")
		resp.Body = failingReader{err: cause}
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetArgs([]string{"/user"})
	err := cmd.Execute()
	if err == nil || !errors.Is(err, cause) || !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Fatalf("error = %v", err)
	}
}

func TestAPIBodyIsNotAddedToGET(t *testing.T) {
	request, err := prepare("/user", options{method: "get", accept: "application/json"}, bytes.NewBuffer(nil))
	if err != nil {
		t.Fatal(err)
	}
	if request.body != nil || request.method != http.MethodGet {
		t.Fatalf("request = %#v", request)
	}
}

var _ io.ReadCloser = failingReader{}
