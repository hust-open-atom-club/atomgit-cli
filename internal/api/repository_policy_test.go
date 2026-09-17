package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCodeReviewPolicyFromValidatesReviewerObjects(t *testing.T) {
	for _, field := range []string{"approvers", "testers"} {
		for _, tc := range []struct {
			entry string
			valid bool
		}{
			{`{}`, true},
			{`{"username":"alice","unknown":{"nested":true}}`, true},
			{`false`, false}, {`42`, false}, {`null`, false}, {`[]`, false}, {`"alice"`, false},
		} {
			t.Run(field+"/"+tc.entry, func(t *testing.T) {
				count := 0
				s := &MergeRequestPolicySettings{
					ApprovalApprovers: []json.RawMessage{}, ApprovalTesters: []json.RawMessage{},
					ApprovalRequiredApprovers: &count, ApprovalRequiredTesters: &count,
				}
				entries := []json.RawMessage{json.RawMessage(tc.entry)}
				if field == "approvers" {
					s.ApprovalApprovers = entries
				} else {
					s.ApprovalTesters = entries
				}
				result, err := CodeReviewPolicyFrom(PullRequestPolicy{MergeRequestSetting: s})
				if (err == nil) != tc.valid {
					t.Fatalf("valid %t, error %v", tc.valid, err)
				}
				if tc.valid {
					got := result.ApprovalApprovers
					if field == "testers" {
						got = result.ApprovalTesters
					}
					if string(got[0]) != tc.entry {
						t.Fatalf("entry changed: %s", got[0])
					}
				}
			})
		}
	}
}

type policyTransport func(*http.Request) (*http.Response, error)

func (f policyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPolicyWritesAreNotRetried(t *testing.T) {
	mode, count, force := 2, 0, false
	for _, tc := range []struct {
		name string
		run  func(*Client) error
	}{
		{"permission", func(c *Client) error {
			return UpdatePermissionPolicy(c, "team", "demo", UpdatePermissionPolicyRequest{Mode: &mode})
		}},
		{"code-review", func(c *Client) error {
			return UpdateCodeReviewPolicy(c, "team", "demo", UpdateCodeReviewPolicyRequest{TestersNumber: &count})
		}},
		{"pull-request", func(c *Client) error {
			return UpdatePullRequestPolicy(c, "team", "demo", UpdatePullRequestPolicyRequest{CanForceMerge: &force})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := NewClientWithHTTPClient("synthetic", &http.Client{Transport: policyTransport(func(r *http.Request) (*http.Response, error) { calls++; return nil, errors.New("transport failed") })})
			if err := tc.run(c); err == nil {
				t.Fatal("expected error")
			}
			if calls != 1 {
				t.Fatalf("calls %d", calls)
			}
		})
	}
}
func TestPolicyPathEscapesSegments(t *testing.T) {
	c := NewClientWithHTTPClient("synthetic", &http.Client{Transport: policyTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.EscapedPath() != "/api/v5/repos/team%2Fsub/repo%3Fname/transition" {
			t.Fatalf("path %s", r.URL.EscapedPath())
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"memberMgntMode":1}`))}, nil
	})})
	if _, err := GetPermissionPolicy(c, "team/sub", "repo?name"); err != nil {
		t.Fatal(err)
	}
}
func TestPermissionPolicyRejectsFailureAcknowledgement(t *testing.T) {
	mode := 2
	c := NewClientWithHTTPClient("synthetic", &http.Client{Transport: policyTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":0,"msg":"failed"}`))}, nil
	})})
	if err := UpdatePermissionPolicy(c, "team", "demo", UpdatePermissionPolicyRequest{Mode: &mode}); err == nil {
		t.Fatal("expected failure")
	}
}
func TestPolicyRejectsMalformedFieldTypes(t *testing.T) {
	for _, body := range []string{`{"memberMgntMode":"2"}`, `{"memberMgntMode":false}`} {
		t.Run(body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
			if _, err := GetPermissionPolicy(c, "team", "demo"); err == nil {
				t.Fatal("expected type error")
			}
		})
	}
	for _, body := range []string{
		`{"merge_request_setting":[],"merge_method":"merge","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false}`,
		`{"merge_request_setting":{"can_force_merge":"false"},"merge_method":"merge","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
			if _, err := GetPullRequestPolicy(c, "team", "demo"); err == nil {
				t.Fatal("expected type error")
			}
		})
	}
}
