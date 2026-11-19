package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type archiveRoundTrip func(*http.Request) (*http.Response, error)

func (f archiveRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func archiveResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestOrganizationRepositoryArchiveRequest(t *testing.T) {
	for _, status := range []int{OrganizationRepositoryArchived, OrganizationRepositoryActive} {
		for _, withPassword := range []bool{true, false} {
			t.Run(fmt.Sprintf("status=%d/password=%t", status, withPassword), func(t *testing.T) {
				const password = " secret password &?\\\" "
				calls := 0
				client := NewClientWithHTTPClient("fixture-token", &http.Client{Transport: archiveRoundTrip(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodPut || req.URL.EscapedPath() != "/api/v5/org/team%20name/repo/demo%20repo/status" || req.URL.RawQuery != "" || req.Header.Get("Authorization") != "Bearer fixture-token" || req.Header.Get("Content-Type") != "application/json" {
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
					}
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if string(body["status"]) != fmt.Sprint(status) {
						t.Fatalf("status missing or wrong: %s", body["status"])
					}
					if withPassword {
						var got string
						if err := json.Unmarshal(body["password"], &got); err != nil || got != password {
							t.Fatal("password did not round-trip")
						}
						if len(body) != 2 {
							t.Fatal("unexpected body fields")
						}
					} else if len(body) != 1 {
						t.Fatal("omitted password was serialized")
					}
					return archiveResponse(200, `{"code":0,"msg":"success"}`), nil
				})})
				var pw *string
				if withPassword {
					value := password
					pw = &value
				}
				if err := SetOrganizationRepositoryArchiveStatus(client, "team name", "demo repo", status, pw); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatalf("calls=%d", calls)
				}
			})
		}
	}
	if err := SetOrganizationRepositoryArchiveStatus(nil, "team", "demo", 1, nil); err == nil {
		t.Fatal("accepted unsupported status")
	}
}

func TestOrganizationRepositoryArchiveRejectsBusinessAndMalformedResponses(t *testing.T) {
	for _, body := range []string{`{"code":1,"msg":"rejected"}`, `{}`, `null`, `[]`, ``, "{", `{"code":null,"msg":"success"}`, `{"code":"0","msg":"success"}`, `{"code":0.1,"msg":"success"}`, `{"code":0}`, `{"code":0,"msg":null}`, `{"code":0,"msg":42}`} {
		t.Run(body, func(t *testing.T) {
			client := NewClientWithHTTPClient("token", &http.Client{Transport: archiveRoundTrip(func(*http.Request) (*http.Response, error) { return archiveResponse(200, body), nil })})
			if err := SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 2, nil); err == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
}

func TestOrganizationRepositoryArchiveHTTPAndTransportErrorsAreNotReplayedOrExposed(t *testing.T) {
	const password = "private-password-fixture"
	for _, status := range []int{201, 204, 301, 304, 400, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("token", &http.Client{Transport: archiveRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return archiveResponse(status, `{"code":0,"msg":"`+password+`"}`), nil
			})})
			pw := password
			err := SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 2, &pw)
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != status || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if strings.Contains(err.Error(), password) {
				t.Fatal("password exposed in error")
			}
		})
	}
	cause := errors.New(password)
	calls := 0
	client := NewClientWithHTTPClient("token", &http.Client{Transport: archiveRoundTrip(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })})
	err := SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 2, nil)
	if !errors.Is(err, cause) || calls != 1 || strings.Contains(err.Error(), password) {
		t.Fatalf("calls=%d or unsafe/lost error", calls)
	}
}

func TestOrganizationRepositoryArchiveDoesNotFollowRedirect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Location", "/replayed")
		w.WriteHeader(307)
	}))
	defer server.Close()
	client := NewClientWithBaseURL("token", server.URL+APIVersion, server.Client())
	err := SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 0, nil)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 307 || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRepositoryArchiveBusinessRejectionPrecedesMissingMessage(t *testing.T) {
	for _, body := range []string{`{"code":9}`, `{"code":9,"msg":null}`, `{"code":9,"msg":"private-password"}`} {
		client := NewClientWithHTTPClient("token", &http.Client{Transport: archiveRoundTrip(func(*http.Request) (*http.Response, error) { return archiveResponse(200, body), nil })})
		err := SetOrganizationRepositoryArchiveStatus(client, "team", "demo", 2, nil)
		if err == nil || errors.Is(err, ErrInvalidResponse) || !strings.Contains(err.Error(), "business code 9") || strings.Contains(err.Error(), "private-password") {
			t.Fatalf("unexpected rejection: %v", err)
		}
	}
}
