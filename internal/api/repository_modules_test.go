package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateRepositoryModulesPartialBooleans(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name    string
		request RepositoryModuleUpdate
		want    map[string]bool
	}{
		{"enable", RepositoryModuleUpdate{Wiki: &yes}, map[string]bool{"has_wiki": true}},
		{"disable", RepositoryModuleUpdate{Issue: &no}, map[string]bool{"has_issue": false}},
		{"all", RepositoryModuleUpdate{Wiki: &yes, Issue: &no, Security: &yes, MergeRequest: &no, Fork: &yes, Analysis: &no, Discussion: &yes}, map[string]bool{"has_wiki": true, "has_issue": false, "has_security": true, "has_merge_request": false, "has_fork": true, "has_analysis": false, "has_discussion": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type key struct{}
			ctx := context.WithValue(t.Context(), key{}, "module")
			calls := 0
			c := NewClientWithBaseURL("fixture", "https://example.test/api/v5", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "PUT" || r.URL.EscapedPath() != "/api/v5/repos/team%23one/repo%2Fname/module/setting" || r.URL.RawQuery != "" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture" || r.Header.Get("Content-Type") != "application/json" || r.Context().Value(key{}) != "module" {
					t.Fatal("headers or context lost")
				}
				var got map[string]bool
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("got=%v want=%v", got, tc.want)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":1,"msg":"success"}`)), Request: r}, nil
			})})
			if err := UpdateRepositoryModules(c.WithContext(ctx), "team#one", "repo/name", tc.request); err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestUpdateRepositoryModulesAcknowledgement(t *testing.T) {
	yes := true
	for _, tc := range []struct {
		body              string
		success, business bool
	}{
		{`{"code":1,"msg":"success"}`, true, false},
		{`{"code":0,"msg":"success"}`, false, true}, {`{"code":-1}`, false, true},
		{`{}`, false, false}, {`null`, false, false}, {`[]`, false, false}, {``, false, false}, {`{`, false, false},
		{`{"code":1}`, false, false}, {`{"code":null,"msg":"success"}`, false, false},
		{`{"code":"1","msg":"success"}`, false, false}, {`{"code":1.5,"msg":"success"}`, false, false},
		{`{"code":1,"msg":null}`, false, false}, {`{"code":1,"msg":42}`, false, false}, {`{"code":1,"msg":"private-message"}`, false, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, tc.body) })
			err := UpdateRepositoryModules(c, "team", "repo", RepositoryModuleUpdate{Wiki: &yes})
			if (err == nil) != tc.success {
				t.Fatalf("err=%v", err)
			}
			if err != nil {
				var uncertain *RepositoryModuleUpdateUncertainError
				if errors.As(err, &uncertain) == tc.business {
					t.Fatalf("incorrect uncertainty: %v", err)
				}
				if !tc.business && !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("lost invalid response: %v", err)
				}
				if strings.Contains(err.Error(), "private-message") {
					t.Fatal("exposed response message")
				}
			}
		})
	}
}

func TestUpdateRepositoryModulesNeverReplays(t *testing.T) {
	no := false
	for _, status := range []int{201, 204, 301, 302, 303, 307, 308, 400, 401, 403, 404, 429, 500, 502, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, redirects := 0, 0
			hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("write replayed")
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://redirect.test/"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"msg":"private-message"}`)), Request: r}, nil
			})}
			err := UpdateRepositoryModules(NewClientWithHTTPClient("fixture", hc), "team", "repo", RepositoryModuleUpdate{Issue: &no})
			var uncertain *RepositoryModuleUpdateUncertainError
			if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 || errors.As(err, &uncertain) != (status < 400 || status >= 500) {
				t.Fatalf("err=%v calls=%d redirects=%d", err, calls, redirects)
			}
			_ = hc.CheckRedirect(nil, nil)
			if redirects != 1 {
				t.Fatal("changed caller redirect policy")
			}
		})
	}
	cause := errors.New("private-transport")
	for _, readFailure := range []bool{false, true} {
		calls := 0
		c := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if !readFailure {
				return nil, cause
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(apiErrorReader{err: cause}), Request: r}, nil
		})})
		err := UpdateRepositoryModules(c, "team", "repo", RepositoryModuleUpdate{Issue: &no})
		var uncertain *RepositoryModuleUpdateUncertainError
		if !errors.As(err, &uncertain) || !errors.Is(err, cause) || calls != 1 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	}
}

func TestUpdateRepositoryModulesValidationBeforeRequest(t *testing.T) {
	if err := UpdateRepositoryModules(nil, "team", "repo", RepositoryModuleUpdate{}); err == nil {
		t.Fatal("empty update accepted")
	}
	yes := true
	for _, target := range [][2]string{{"", "repo"}, {"team", ""}} {
		if err := UpdateRepositoryModules(nil, target[0], target[1], RepositoryModuleUpdate{Wiki: &yes}); err == nil {
			t.Fatal("empty repository accepted")
		}
	}
}
