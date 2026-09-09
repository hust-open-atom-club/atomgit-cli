package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRedirectAuthorizationExactOrigin(t *testing.T) {
	tests := []struct {
		name       string
		location   string
		wantSecond bool
	}{
		{name: "same origin", location: "https://api.atomgit.com/api/v5/final", wantSecond: true},
		{name: "host change", location: "https://other.atomgit.com/api/v5/final"},
		{name: "scheme change", location: "http://api.atomgit.com/api/v5/final"},
		{name: "port change", location: "https://api.atomgit.com:444/api/v5/final"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					resp := apiTestResponse(req, http.StatusFound, "")
					resp.Header.Set("Location", tt.location)
					return resp, nil
				}
				got := req.Header.Get("Authorization") != ""
				if got != tt.wantSecond {
					t.Fatalf("redirect Authorization present = %v", got)
				}
				return apiTestResponse(req, http.StatusOK, "ok"), nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs([]string{"/start"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRedirectNeverRestoresAuthorization(t *testing.T) {
	calls := 0
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		calls++
		resp := apiTestResponse(req, http.StatusFound, "")
		switch calls {
		case 1:
			resp.Header.Set("Location", "https://evil.test/away")
		case 2:
			if req.Header.Get("Authorization") != "" {
				t.Fatal("credential followed cross-origin redirect")
			}
			resp.Header.Set("Location", "https://api.atomgit.com/api/v5/back")
		default:
			if req.Header.Get("Authorization") != "" {
				t.Fatal("credential was restored after returning to original origin")
			}
			return apiTestResponse(req, http.StatusOK, "ok"), nil
		}
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(fixture.stdout)
	cmd.SetArgs([]string{"/start"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRedirectPreservesClientAndCallback(t *testing.T) {
	callbackErr := errors.New("redirect refused")
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return callbackErr }}
	clone := cloneRedirectSafeClient(client)
	if clone == client || clone.Timeout != client.Timeout || client.CheckRedirect == nil {
		t.Fatal("client was not cloned without mutation")
	}
	err := clone.CheckRedirect(&http.Request{URL: mustURL(t, "https://api.atomgit.com/api/v5/next"), Header: make(http.Header)}, []*http.Request{{URL: mustURL(t, "https://api.atomgit.com/api/v5/start")}})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("callback error = %v", err)
	}
}

func TestRedirectEnforcesOriginAfterExistingCallback(t *testing.T) {
	client := &http.Client{CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		req.URL = mustURL(t, "https://evil.test/changed")
		req.Header.Set("Authorization", "Bearer restored")
		return nil
	}}
	clone := cloneRedirectSafeClient(client)
	req := &http.Request{URL: mustURL(t, "https://api.atomgit.com/api/v5/next"), Header: make(http.Header)}
	via := []*http.Request{{URL: mustURL(t, "https://api.atomgit.com/api/v5/start"), Header: http.Header{"Authorization": {"Bearer secret"}}}}
	if err := clone.CheckRedirect(req, via); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
	}
}

func TestRedirectStripsAuthorizationBeforeExistingCallback(t *testing.T) {
	client := &http.Client{CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		if req.Header.Get("Authorization") != "" {
			t.Fatalf("callback observed Authorization %q", req.Header.Get("Authorization"))
		}
		return nil
	}}
	clone := cloneRedirectSafeClient(client)
	req := &http.Request{URL: mustURL(t, "https://evil.test/next"), Header: http.Header{"Authorization": {"Bearer secret"}}}
	via := []*http.Request{{URL: mustURL(t, "https://api.atomgit.com/api/v5/start"), Header: http.Header{"Authorization": {"Bearer secret"}}}}
	if err := clone.CheckRedirect(req, via); err != nil {
		t.Fatal(err)
	}
}

func TestRedirectLoopUsesDefaultLimit(t *testing.T) {
	requests := 0
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		requests++
		resp := apiTestResponse(req, http.StatusFound, "")
		resp.Header.Set("Location", "/api/v5/loop")
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetArgs([]string{"/loop"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "10 redirects") {
		t.Fatalf("error = %v", err)
	}
	if requests != 20 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestAPIHonorsInjectedClientTimeout(t *testing.T) {
	fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	fixture.factory.HttpClient = func() (*http.Client, error) {
		fixture.clientReads++
		return &http.Client{
			Timeout: time.Millisecond,
			Transport: apiRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				fixture.requests++
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		}, nil
	}
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetArgs([]string{"/user"})
	err := cmd.Execute()
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if fixture.requests != 2 {
		t.Fatalf("requests = %d, want retry with preserved timeout", fixture.requests)
	}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
