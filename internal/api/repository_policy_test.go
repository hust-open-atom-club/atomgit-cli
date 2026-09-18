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
		`{"merge_request_setting":{"can_force_merge":"invalid"},"merge_method":"merge","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
			if _, err := GetPullRequestPolicy(c, "team", "demo"); err == nil {
				t.Fatal("expected type error")
			}
		})
	}
}

func TestPolicyBooleanResponseNormalization(t *testing.T) {
	for _, field := range []string{
		"approval_required_reviewers_enable", "disable_merge_by_self", "can_force_merge",
		"add_notes_after_merged", "mark_auto_merged_mr_as_closed", "can_reopen",
		"delete_source_branch_when_merged", "disable_squash_merge", "auto_squash_merge",
		"squash_merge_with_no_merge_commit", "is_check_cla", "is_allow_lite_merge_request",
		"close_issue_when_mr_merged", "forbidden_pr_related_issue_closed",
		"only_allow_merge_if_pipeline_succeeds", "only_allow_merge_if_all_discussions_are_resolved",
	} {
		for _, tc := range []struct {
			encoded string
			want    bool
		}{
			{"0", false}, {"1", true}, {"false", false}, {"true", true},
			{`"0"`, false}, {`"1"`, true}, {`"false"`, false}, {`"true"`, true},
		} {
			t.Run(field+"/"+tc.encoded, func(t *testing.T) {
				payload := `{"` + field + `":` + tc.encoded + `}`
				topLevel := strings.HasPrefix(field, "only_allow_merge_if_")
				if !topLevel {
					payload = `{"merge_request_setting":` + payload + `}`
				} else {
					payload = strings.TrimSuffix(payload, "}") + `,"merge_request_setting":{}}`
				}
				var response PullRequestPolicy
				if err := json.Unmarshal([]byte(payload), &response); err != nil {
					t.Fatal(err)
				}
				var output any = response.MergeRequestSetting
				if topLevel {
					output = response
				}
				data, err := json.Marshal(output)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]any
				if err := json.Unmarshal(data, &fields); err != nil {
					t.Fatal(err)
				}
				if fields[field] != tc.want {
					t.Fatalf("normalized %s = %#v, want %t", field, fields[field], tc.want)
				}
				for _, requested := range []bool{false, true} {
					var request UpdatePullRequestPolicyRequest
					if err := json.Unmarshal(fmt.Appendf(nil, `{"%s":%t}`, field, requested), &request); err != nil {
						t.Fatal(err)
					}
					err := validatePolicyUpdate(request, response)
					if (err == nil) != (requested == tc.want) {
						t.Fatalf("requested %t, returned %s: %v", requested, tc.encoded, err)
					}
				}
			})
		}
	}
}

func TestPolicyBooleanResponseMissingAndNull(t *testing.T) {
	for _, payload := range []string{
		`{"merge_request_setting":{}}`,
		`{"merge_request_setting":{"can_force_merge":null,"forbidden_pr_related_issue_closed":null},"only_allow_merge_if_pipeline_succeeds":null}`,
	} {
		t.Run(payload, func(t *testing.T) {
			var response PullRequestPolicy
			if err := json.Unmarshal([]byte(payload), &response); err != nil {
				t.Fatal(err)
			}
			if response.MergeRequestSetting.CanForceMerge != nil || response.MergeRequestSetting.ForbiddenPRRelatedIssueClosed != nil || response.OnlyAllowMergeIfPipelineSucceeds != nil {
				t.Fatal("unavailable fields must remain nil")
			}
			value := true
			if err := validatePolicyUpdate(UpdatePullRequestPolicyRequest{CanForceMerge: &value, ForbiddenPRRelatedIssueClosed: &value}, response); err != nil {
				t.Fatalf("unavailable fields must not be compared as false: %v", err)
			}
		})
	}
}
