package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreatePullRequestReplySingleAttempt(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204, 301, 302, 303, 307, 308, 400, 401, 403, 404, 422, 429, 500, 502} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.EscapedPath() != "/api/v5/repos/team/demo/pulls/8/discussions/thread%2F%23%3F/comments" || r.URL.RawQuery != "" {
					t.Fatalf("path=%s", r.URL)
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://other.test/"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"id":"thread/#?","noteId":42}`)), Request: r}, nil
			})}
			client := NewClientWithHTTPClient("fixture", hc)
			_, err := CreatePullRequestReply(client, "team", "demo", "8", "thread/#?", "body")
			if calls != 1 || hc.CheckRedirect != nil {
				t.Fatalf("calls=%d; changed caller redirect policy=%t", calls, hc.CheckRedirect != nil)
			}
			if status == http.StatusOK || status == http.StatusCreated {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var uncertain *PullRequestReplyUncertainError
			wantUncertain := status < 400 || status >= 500
			if !IsHTTPStatus(err, status) || errors.As(err, &uncertain) != wantUncertain {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestCreatePullRequestReplyExecutionFailure(t *testing.T) {
	cause := errors.New("fixture transport failure")
	calls := 0
	client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })})
	_, err := CreatePullRequestReply(client, "team", "demo", "8", "thread", "body")
	var uncertain *PullRequestReplyUncertainError
	if calls != 1 || !errors.Is(err, cause) || !errors.As(err, &uncertain) {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}
