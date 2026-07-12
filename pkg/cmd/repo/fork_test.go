package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type forkTestConfig struct{}

func (forkTestConfig) GetToken() (string, error) { return "test-token", nil }
func (forkTestConfig) GetUser() (string, error)  { return "alice", nil }
func (forkTestConfig) GetHost() string           { return "atomgit.com" }

type forkRoundTripFunc func(*http.Request) (*http.Response, error)

func (f forkRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func forkResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRunForkUpdatesAndVerifiesDescription(t *testing.T) {
	const description = "Temporary CLI audit fork"
	var requests []string

	httpClient := &http.Client{Transport: forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		requests = append(requests, req.Method+" "+req.URL.Path)

		switch len(requests) {
		case 1:
			if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/openEuler/kernel/forks" {
				t.Fatalf("fork request = %s %s", req.Method, req.URL.Path)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["name"] != "kernel-audit" || body["private"] != false {
				t.Fatalf("fork body = %#v", body)
			}
			if _, exists := body["description"]; exists {
				t.Fatalf("fork request unexpectedly relied on description: %#v", body)
			}
			return forkResponse(http.StatusCreated, `{"name":"kernel-audit","web_url":"https://atomgit.com/alice/kernel-audit"}`), nil

		case 2:
			if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/alice/kernel-audit" {
				t.Fatalf("update request = %s %s", req.Method, req.URL.Path)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["name"] != "kernel-audit" || body["description"] != description {
				t.Fatalf("update body = %#v", body)
			}
			return forkResponse(http.StatusOK, `{"name":"kernel-audit","description":"Temporary CLI audit fork"}`), nil

		case 3:
			if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/kernel-audit" {
				t.Fatalf("verification request = %s %s", req.Method, req.URL.Path)
			}
			return forkResponse(http.StatusOK, `{"name":"kernel-audit","description":"Temporary CLI audit fork"}`), nil
		default:
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})}

	factory := &cmdutil.Factory{
		Config: forkTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return httpClient, nil
		},
	}
	opts := &ForkOptions{Name: "kernel-audit", Description: description, Public: true}
	if err := runFork(factory, opts, "openEuler/kernel"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"POST /api/v5/repos/openEuler/kernel/forks",
		"PATCH /api/v5/repos/alice/kernel-audit",
		"GET /api/v5/repos/alice/kernel-audit",
	}
	if fmt.Sprint(requests) != fmt.Sprint(want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
}

func TestSetAndVerifyForkDescriptionDetectsMismatch(t *testing.T) {
	call := 0
	httpClient := &http.Client{Transport: forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		call++
		if call == 1 {
			return forkResponse(http.StatusOK, `{"description":"wanted"}`), nil
		}
		return forkResponse(http.StatusOK, `{"description":"different"}`), nil
	})}
	client := api.NewClientWithHTTPClient("token", httpClient)

	err := setAndVerifyForkDescription(client, "alice", "repo", "wanted")
	if err == nil || !strings.Contains(err.Error(), "description mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunForkWithoutDescriptionOnlyForks(t *testing.T) {
	requests := 0
	httpClient := &http.Client{Transport: forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return forkResponse(http.StatusCreated, `{"name":"kernel"}`), nil
	})}
	factory := &cmdutil.Factory{
		Config: forkTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return httpClient, nil
		},
	}

	if err := runFork(factory, &ForkOptions{}, "openEuler/kernel"); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("request count = %d", requests)
	}
}
