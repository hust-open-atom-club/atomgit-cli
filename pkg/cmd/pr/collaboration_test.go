package pr

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

func TestPRCreateCollaborationMetadataMappingAndOrder(t *testing.T) {
	var requests []string
	factory := collaborationTestFactory(t, func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.RequestURI())
		switch {
		case strings.HasPrefix(req.URL.Path, "/api/v5/users/"):
			login := strings.TrimPrefix(req.URL.Path, "/api/v5/users/")
			return prResponse(http.StatusOK, fmt.Sprintf(`{"login":%q}`, login)), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/labels":
			return prResponse(http.StatusOK, `[{"name":"Bug"}]`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/milestones":
			return prResponse(http.StatusOK, `[{"number":7,"title":"v1.0"}]`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/v5/repos/alice/demo/pulls":
			var body map[string]interface{}
			decodePRRequest(t, req, &body)
			assertJSONValue(t, body, "assignees", "ann")
			assertJSONValue(t, body, "testers", "tess")
			assertJSONValue(t, body, "labels", "Bug")
			assertJSONValue(t, body, "milestone_number", float64(7))
			if _, exists := body["reviewers"]; exists {
				t.Fatal("create body unexpectedly contains reviewers")
			}
			return prResponse(http.StatusCreated, `{"number":42,"web_url":"https://atomgit.com/alice/demo/pulls/42"}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/reviewers":
			var body map[string]interface{}
			decodePRRequest(t, req, &body)
			assertJSONValue(t, body, "reviewers", "ruth")
			assertJSONValue(t, body, "add", true)
			return prResponse(http.StatusNoContent, ""), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
			return nil, nil
		}
	})

	cmd := newCmdPRCreate(factory)
	cmd.SetArgs([]string{"alice/demo", "--title", "Change", "--base", "main", "--head", "feature", "--assignee", "ann", "--reviewer", "ruth", "--tester", "tess", "--label", "bug", "--milestone", "v1.0"})
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created PR #42: https://atomgit.com/alice/demo/pull/42\n" {
		t.Fatalf("output = %q", got)
	}
	if got := requests[len(requests)-2:]; strings.Join(got, "|") != "POST /api/v5/repos/alice/demo/pulls|POST /api/v5/repos/alice/demo/pulls/42/reviewers" {
		t.Fatalf("final request order = %#v", got)
	}
}

func TestPRCreateReportsCreatedPRWhenReviewerUpdateFails(t *testing.T) {
	created := 0
	factory := collaborationTestFactory(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/v5/users/ruth":
			return prResponse(http.StatusOK, `{"login":"ruth"}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls":
			created++
			return prResponse(http.StatusCreated, `{"number":42,"web_url":"https://atomgit.com/alice/demo/pulls/42"}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/reviewers":
			return prResponse(http.StatusForbidden, `{"message":"forbidden"}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
			return nil, nil
		}
	})
	cmd := newCmdPRCreate(factory)
	cmd.SetArgs([]string{"alice/demo", "--title", "Change", "--base", "main", "--head", "feature", "--reviewer", "ruth"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "created PR #42 at https://atomgit.com/alice/demo/pull/42") || !strings.Contains(err.Error(), "403 Forbidden") {
		t.Fatalf("error = %v", err)
	}
	if created != 1 {
		t.Fatalf("create requests = %d, want 1", created)
	}
}

func TestPREditCollaborationMetadataOrderAndBodies(t *testing.T) {
	var mutations []string
	factory := collaborationTestFactory(t, func(req *http.Request) (*http.Response, error) {
		if strings.HasPrefix(req.URL.Path, "/api/v5/users/") {
			login := strings.TrimPrefix(req.URL.Path, "/api/v5/users/")
			return prResponse(http.StatusOK, fmt.Sprintf(`{"login":%q}`, login)), nil
		}
		switch {
		case req.URL.Path == "/api/v5/repos/alice/demo/labels" && req.Method == http.MethodGet:
			return prResponse(http.StatusOK, `[{"name":"Bug"},{"name":"Ready"}]`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" && req.Method == http.MethodGet:
			return prResponse(http.StatusOK, `{"number":42,"assignees":[{"login":"ann"},{"login":"bob"}],"approval_reviewers":[{"login":"ruth"}],"testers":[{"login":"tess"}],"milestone":{"number":1,"title":"old"}}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/labels" && req.Method == http.MethodGet:
			return prResponse(http.StatusOK, `[{"name":"Bug"}]`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/milestones" && req.Method == http.MethodGet:
			return prResponse(http.StatusOK, `[{"number":2,"title":"next"}]`), nil
		default:
			mutations = append(mutations, req.Method+" "+req.URL.EscapedPath())
			assertEditMutationBody(t, req)
			return prResponse(http.StatusNoContent, ""), nil
		}
	})

	cmd := newCmdPREdit(factory)
	cmd.SetArgs([]string{"alice/demo", "42", "--add-assignee", "cara", "--remove-assignee", "bob", "--add-reviewer", "vera", "--remove-reviewer", "ruth", "--add-tester", "tom", "--remove-tester", "tess", "--add-label", "ready", "--remove-label", "bug", "--milestone", "next"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"PATCH /api/v5/repos/alice/demo/pulls/42",
		"DELETE /api/v5/repos/alice/demo/pulls/42/assignees",
		"POST /api/v5/repos/alice/demo/pulls/42/assignees",
		"DELETE /api/v5/repos/alice/demo/pulls/42/reviewers",
		"POST /api/v5/repos/alice/demo/pulls/42/reviewers",
		"DELETE /api/v5/repos/alice/demo/pulls/42/testers",
		"POST /api/v5/repos/alice/demo/pulls/42/testers",
		"DELETE /api/v5/repos/alice/demo/pulls/42/labels/Bug",
		"POST /api/v5/repos/alice/demo/pulls/42/labels",
	}
	if strings.Join(mutations, "|") != strings.Join(want, "|") {
		t.Fatalf("mutations = %#v, want %#v", mutations, want)
	}
}

func TestPREditCollaborationMetadataIsIdempotent(t *testing.T) {
	mutations := 0
	factory := collaborationTestFactory(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			mutations++
			return prResponse(http.StatusNoContent, ""), nil
		}
		switch {
		case req.URL.Path == "/api/v5/users/ann":
			return prResponse(http.StatusOK, `{"login":"ann"}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/labels":
			return prResponse(http.StatusOK, `[{"name":"Bug"}]`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls/42":
			return prResponse(http.StatusOK, `{"number":42,"assignees":[{"login":"ann"}],"labels":[{"name":"Bug"}],"milestone":{"number":2}}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/labels":
			return prResponse(http.StatusOK, `[{"name":"Bug"}]`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/milestones":
			return prResponse(http.StatusOK, `[{"number":2,"title":"next"}]`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.RequestURI())
			return nil, nil
		}
	})
	cmd := newCmdPREdit(factory)
	cmd.SetArgs([]string{"alice/demo", "42", "--add-assignee", "ann", "--remove-reviewer", "ann", "--add-label", "bug", "--remove-label", "Ready", "--milestone", "2"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `label "Ready" does not exist`) {
		// Invalid metadata must stop before all mutation. Use a second command for
		// the no-op assertion after verifying preflight behavior.
		t.Fatalf("error = %v", err)
	}
	if mutations != 0 {
		t.Fatalf("mutations after failed preflight = %d", mutations)
	}

	mutations = 0
	cmd = newCmdPREdit(factory)
	cmd.SetArgs([]string{"alice/demo", "42", "--add-assignee", "ann", "--add-label", "bug", "--milestone", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if mutations != 0 {
		t.Fatalf("no-op mutations = %d, want 0", mutations)
	}
}

func TestPREditRejectsConflictingRoleChangeBeforeMutation(t *testing.T) {
	mutations := 0
	factory := collaborationTestFactory(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			mutations++
		}
		return prResponse(http.StatusOK, `{"login":"ann"}`), nil
	})
	cmd := newCmdPREdit(factory)
	cmd.SetArgs([]string{"alice/demo", "42", "--add-tester", "ann", "--remove-tester", "ANN"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `tester "ann" cannot be both added and removed`) {
		t.Fatalf("error = %v", err)
	}
	if mutations != 0 {
		t.Fatalf("mutations = %d, want 0", mutations)
	}
}

func TestApplyAssigneeChangesReportsRestoreFailure(t *testing.T) {
	client := api.NewClientWithBaseURL("token", "https://api.atomgit.com/api/v5", &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			return prResponse(http.StatusNoContent, ""), nil
		}
		return prResponse(http.StatusForbidden, `{"message":"forbidden"}`), nil
	})})
	err := applyAssigneeChanges(client, "/repos/alice/demo/pulls/42", resolvedPREditMetadata{CurrentAssignees: []string{"ann", "bob"}, RemoveAssignees: []string{"bob"}})
	if err == nil || !strings.Contains(err.Error(), "removed all assignees but failed to restore desired assignees ann") {
		t.Fatalf("error = %v", err)
	}
}

func collaborationTestFactory(t *testing.T, roundTrip prRoundTripFunc) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTrip}, nil
		},
	}
}

func decodePRRequest(t *testing.T, req *http.Request, target interface{}) {
	t.Helper()
	if req.Body == nil {
		return
	}
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(target); err != nil && err != io.EOF {
		t.Fatal(err)
	}
}

func assertJSONValue(t *testing.T, body map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if got := body[key]; got != want {
		t.Fatalf("body[%q] = %#v, want %#v", key, got, want)
	}
}

func assertEditMutationBody(t *testing.T, req *http.Request) {
	t.Helper()
	var body interface{}
	decodePRRequest(t, req, &body)
	path := req.URL.Path
	switch {
	case req.Method == http.MethodPatch:
		assertBodyMapValue(t, body, "milestone_number", float64(2))
	case strings.HasSuffix(path, "/assignees") && req.Method == http.MethodDelete:
		if body != nil {
			t.Fatalf("assignee DELETE body = %#v, want nil", body)
		}
	case strings.HasSuffix(path, "/assignees"):
		assertBodyMapValue(t, body, "assignees", "ann,cara")
	case strings.HasSuffix(path, "/reviewers"):
		assertBodyMapValue(t, body, "reviewers", map[bool]string{true: "vera", false: "ruth"}[req.Method == http.MethodPost])
	case strings.HasSuffix(path, "/testers"):
		assertBodyMapValue(t, body, "testers", map[bool]string{true: "tom", false: "tess"}[req.Method == http.MethodPost])
	case strings.HasSuffix(path, "/labels"):
		values, ok := body.([]interface{})
		if !ok || len(values) != 1 || values[0] != "Ready" {
			t.Fatalf("label add body = %#v", body)
		}
	case strings.Contains(path, "/labels/"):
		if body != nil {
			t.Fatalf("label DELETE body = %#v, want nil", body)
		}
	}
}

func assertBodyMapValue(t *testing.T, body interface{}, key string, want interface{}) {
	t.Helper()
	values, ok := body.(map[string]interface{})
	if !ok {
		t.Fatalf("body = %#v, want object", body)
	}
	if got := values[key]; got != want {
		t.Fatalf("body[%q] = %#v, want %#v", key, got, want)
	}
}
