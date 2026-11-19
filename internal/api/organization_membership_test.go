package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func membershipOperations() map[string]func(*Client, string, string) error {
	return map[string]func(*Client, string, string) error{
		"invite": func(c *Client, org, user string) error {
			return InviteOrganizationMember(c, org, user, OrganizationInvitationRequest{Permission: "push"})
		},
		"remove": RemoveOrganizationMember,
	}
}

func TestOrganizationMembershipDoesNotReplay(t *testing.T) {
	transportErr := errors.New("connection interrupted")
	for action, call := range membershipOperations() {
		for _, status := range []int{0, 201, 202, 204, 301, 302, 303, 307, 308, 400, 401, 403, 404, 429, 500, 503} {
			t.Run(fmt.Sprintf("%s/%d", action, status), func(t *testing.T) {
				calls, redirects := 0, 0
				hc := &http.Client{
					CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						if calls > 1 {
							t.Fatal("write replayed")
						}
						if status == 0 {
							return nil, transportErr
						}
						return &http.Response{StatusCode: status, Request: req, Header: http.Header{"Location": {"https://other.test/memberships"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"message":"failure"}`))}, nil
					}),
				}
				err := call(NewClientWithBaseURL("token", "https://example.test/api/v5", hc), "team", "alice")
				if err == nil || calls != 1 || redirects != 0 {
					t.Fatalf("err=%v calls=%d redirects=%d", err, calls, redirects)
				}
				if status == 0 && !errors.Is(err, transportErr) {
					t.Fatalf("lost transport error: %v", err)
				}
				if status != 0 && !IsHTTPStatus(err, status) {
					t.Fatalf("lost status: %v", err)
				}
				if err := hc.CheckRedirect(nil, nil); err != nil || redirects != 1 {
					t.Fatal("shared redirect policy changed")
				}
			})
		}
	}
}

func TestOrganizationMembershipPathAndContext(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(t.Context(), key{}, "request")
	for action, call := range membershipOperations() {
		t.Run(action, func(t *testing.T) {
			calls := 0
			hc := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Context().Value(key{}) != "request" || req.URL.EscapedPath() != "/custom/v5/orgs/team%2Fname/memberships/user%3Fname" || req.Header.Get("Accept") != "application/json" {
					t.Fatalf("unexpected request: %s", req.URL)
				}
				if err := req.Context().Err(); err != nil {
					return nil, err
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
			})}
			client := NewClientWithBaseURL("token", "https://example.test/custom/v5", hc)
			if err := call(client.WithContext(ctx), "team/name", "user?name"); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("calls=%d", calls)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if err := call(client.WithContext(cancelled), "team/name", "user?name"); !errors.Is(err, context.Canceled) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
