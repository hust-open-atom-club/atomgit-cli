package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSetPullRequestDraftSendsFalse(t *testing.T) {
	requests := 0
	client := NewClientWithBaseURL("fixture", "https://example.test/api/v5", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/api/v5/repos/team/repo/pulls/42" || req.Header.Get("Authorization") != "Bearer fixture" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || body["draft"] != false {
			t.Fatalf("body = %v", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
	})})
	if err := SetPullRequestDraft(client, "team", "repo", "42", false); err != nil || requests != 1 {
		t.Fatalf("err=%v requests=%d", err, requests)
	}
}

func TestSetPullRequestDraftDoesNotReplay(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requests := 0
			client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://other.test/"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
			})})
			if err := SetPullRequestDraft(client, "team", "repo", "42", true); !IsHTTPStatus(err, status) || requests != 1 {
				t.Fatalf("err=%v requests=%d", err, requests)
			}
		})
	}
}
