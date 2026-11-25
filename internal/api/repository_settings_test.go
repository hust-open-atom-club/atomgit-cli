package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetRepositorySettingsPartialValues(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/repos/team%23name/demo%2Frepo/repo_settings" || r.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Accept") != "application/json" {
			t.Fatal("missing headers")
		}
		_, _ = io.WriteString(w, `{"action_pipeline_enabled":false,"disable_fork":null,"branch_name_regex":"","forbidden_developer_create_branch_users":[{"user_id":9007199254740993123,"user_name":"A\tB\nC"}],"future_field":true}`)
	})
	settings, err := GetRepositorySettings(client, "team#name", "demo/repo")
	if err != nil {
		t.Fatal(err)
	}
	if settings.ActionPipelineEnabled == nil || *settings.ActionPipelineEnabled || settings.DisableFork != nil || settings.ShowBranchCreator != nil || settings.BranchNameRegex == nil || *settings.BranchNameRegex != "" {
		t.Fatalf("settings=%+v", settings)
	}
	u := settings.ForbiddenDeveloperCreateBranchUsers[0]
	if u.UserID == nil || *u.UserID != 9007199254740993123 || u.ID != nil || u.UserName == nil || *u.UserName != "A\tB\nC" {
		t.Fatalf("user=%+v", u)
	}
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"action_pipeline_enabled":false`, `"show_branch_creator":null`, `"branch_name_regex":""`, `9007199254740993123`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s: %s", want, data)
		}
	}
}

func TestUpdateRepositorySettingsExplicitValues(t *testing.T) {
	no, empty := false, ""
	request := UpdateRepositorySettingsRequest{DisableFork: &no, ForbiddenDeveloperCreateBranch: &no, ForbiddenDeveloperCreateTag: &no, ForbiddenCommitterCreateBranch: &no, ForbiddenDeveloperCreateBranchUserIDs: &empty, BranchNameRegex: &empty, TagNameRegex: &empty, GeneratePreMergeRef: &no, RebaseDisableTriggerWebhook: &no, OpenGPGVerified: &no, IncludeLFSObjects: &no, ActionPipelineEnabled: &no, BlockCrossRepoPRTrigger: &no, ShowBranchCreator: &no}
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "settings")
	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" || r.URL.Path != "/repos/team/demo/repo_settings" || r.Header.Get("Content-Type") != "application/json" || r.Context().Value(contextKey{}) != "settings" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 14 {
			t.Fatalf("body=%s", body)
		}
		for key, value := range body {
			want := "false"
			if key == "branch_name_regex" || key == "tag_name_regex" || key == "forbidden_developer_create_branch_user_ids" {
				want = `""`
			}
			if string(value) != want {
				t.Fatalf("field=%s value=%s want=%s", key, value, want)
			}
		}
		_, _ = io.WriteString(w, `{"action_pipeline_enabled":false}`)
	})
	settings, err := UpdateRepositorySettings(client.WithContext(ctx), "team", "demo", request)
	if err != nil || calls != 1 || settings.ActionPipelineEnabled == nil || *settings.ActionPipelineEnabled {
		t.Fatalf("result=%+v error=%v calls=%d", settings, err, calls)
	}
	if _, err := UpdateRepositorySettings(nil, "team", "demo", UpdateRepositorySettingsRequest{}); err == nil {
		t.Fatal("accepted no changes")
	}
}

func TestRepositorySettingsResponseContract(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{}`, `{"unknown":true}`, `{"disable_fork":null}`, `{"action_pipeline_enabled":"false"}`, `{"action_pipeline_enabled":0}`, `{"tag_name_regex":false}`, `{"forbidden_developer_create_branch_users":[null]}`, `{"forbidden_developer_create_branch_users":[{"id":"123"}]}`, `{"action_pipeline_enabled":`} {
		t.Run(body, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) })
			if _, err := GetRepositorySettings(client, "team", "demo"); !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("error=%v", err)
			}
			value := false
			_, err := UpdateRepositorySettings(client, "team", "demo", UpdateRepositorySettingsRequest{DisableFork: &value})
			var uncertain *RepositorySettingsUpdateUncertainError
			if !errors.Is(err, ErrInvalidResponse) || !errors.As(err, &uncertain) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"forbidden_developer_create_branch_users":[]}`)
	})
	settings, err := GetRepositorySettings(client, "team", "demo")
	if err != nil || settings.ForbiddenDeveloperCreateBranchUsers == nil || len(settings.ForbiddenDeveloperCreateBranchUsers) != 0 {
		t.Fatalf("result=%+v error=%v", settings, err)
	}
}

func TestRepositorySettingsWriteDoesNotReplay(t *testing.T) {
	value := true
	for _, status := range []int{201, 204, 301, 302, 303, 304, 307, 308, 400, 401, 403, 404, 422, 429, 500, 502} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, redirects := 0, 0
			original := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("PUT replayed")
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://other.test/replayed"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"action_pipeline_enabled":true}`)), Request: r}, nil
			})}
			_, err := UpdateRepositorySettings(NewClientWithHTTPClient("token", original), "team", "demo", UpdateRepositorySettingsRequest{ActionPipelineEnabled: &value})
			var uncertain *RepositorySettingsUpdateUncertainError
			wantUncertain := status < 400 || status >= 500
			if !IsHTTPStatus(err, status) || errors.As(err, &uncertain) != wantUncertain || calls != 1 || redirects != 0 {
				t.Fatalf("error=%v calls=%d redirects=%d", err, calls, redirects)
			}
			_ = original.CheckRedirect(nil, nil)
			if redirects != 1 {
				t.Fatal("changed caller redirect policy")
			}
		})
	}
}

func TestRepositorySettingsReadHTTPErrors(t *testing.T) {
	for _, status := range []int{201, 401, 403, 404, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.WriteHeader(status)
			})
			if _, err := GetRepositorySettings(client, "team", "demo"); !IsHTTPStatus(err, status) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestRepositorySettingsReadDoesNotRedirect(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, redirects := 0, 0
			original := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					code := status
					if calls > 1 {
						code = http.StatusOK
					}
					return &http.Response{StatusCode: code,
						Header: http.Header{"Location": {"https://redirect.test/settings"}},
						Body:   io.NopCloser(strings.NewReader(`{"action_pipeline_enabled":true}`)), Request: r}, nil
				}),
			}
			_, err := GetRepositorySettings(NewClientWithHTTPClient("fixture", original), "team", "demo")
			if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
				t.Fatalf("error=%v calls=%d redirects=%d", err, calls, redirects)
			}
			if err := original.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatal("changed caller redirect policy")
			}
		})
	}
}

func TestRepositorySettingsWritePreservesFailures(t *testing.T) {
	value := false
	cause := errors.New("synthetic failure")
	for _, readFailure := range []bool{false, true} {
		calls := 0
		client := NewClientWithHTTPClient("token", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if !readFailure {
				return nil, cause
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(apiErrorReader{err: cause}), Request: r}, nil
		})})
		_, err := UpdateRepositorySettings(client, "team", "demo", UpdateRepositorySettingsRequest{DisableFork: &value})
		var uncertain *RepositorySettingsUpdateUncertainError
		if !errors.Is(err, cause) || !errors.As(err, &uncertain) || calls != 1 {
			t.Fatalf("error=%v calls=%d", err, calls)
		}
	}
}
