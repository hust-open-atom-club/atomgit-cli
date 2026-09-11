package api_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/apicontract"
)

func contractFixtures(t *testing.T) []apicontract.Fixture {
	t.Helper()
	fixtures, err := apicontract.Load(os.DirFS("testdata/contracts"), "*.json")
	if err != nil {
		t.Fatal(err)
	}
	return fixtures
}

// Replay the fixtures through production clients, so a path/status/DTO change
// cannot pass just because the fixture validator still accepts its own input.
func TestAPIContracts(t *testing.T) {
	for _, f := range contractFixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "issue-list" {
				replay := &apicontract.Replay{Fixture: f, Examples: f.Examples}
				t.Cleanup(func() {
					if err := replay.Check(); err != nil {
						t.Error(err)
					}
				})
				client := api.NewClientWithHTTPClient("fixture-token", &http.Client{Transport: replay})
				issues, err := api.GetPaginatedWithPageSize[api.Issue](client, 5, 2, func(page, size int) string {
					return fmt.Sprintf("/repos/fixture-owner/fixture-repo/issues?page=%d&per_page=%d", page, size)
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(issues) != 4 {
					t.Fatalf("expected 4 issues across pages, got %d", len(issues))
				}
				for i, issue := range issues {
					if issue.ID != int64(i+1) || issue.GetNumber() != fmt.Sprint(i+1) || issue.Title != "Synthetic contract issue" || issue.State != "open" {
						t.Fatal("issue DTO or pagination drift")
					}
				}
				return
			}
			for _, ex := range f.Examples {
				t.Run(ex.Name, func(t *testing.T) {
					replay := &apicontract.Replay{Fixture: f, Examples: []apicontract.Example{ex}}
					t.Cleanup(func() {
						if err := replay.Check(); err != nil {
							t.Error(err)
						}
					})
					httpClient := &http.Client{Transport: replay}
					v5 := api.NewClientWithHTTPClient("fixture-token", httpClient)
					v8 := actions.NewClientWithHTTPClient("fixture-token", httpClient)
					switch f.Name {
					case "repository":
						var repo api.Repository
						if err := v5.Get("/repos/fixture-owner/fixture-repo", &repo); err != nil {
							t.Fatal(err)
						}
						if repo.ID != 1 || repo.Name != "fixture-repo" || repo.FullName != "fixture-owner/fixture-repo" || repo.DefaultBranch != "main" || repo.Owner.Login != "fixture-owner" {
							t.Fatal("repository DTO drift")
						}
					case "issue-create":
						issue, err := api.CreateIssueWithAssignee(v5, "fixture-owner", "fixture-repo", "Synthetic contract issue", "", "")
						if err != nil {
							t.Fatal(err)
						}
						if issue.ID != 1 || issue.GetNumber() != "1" || issue.Title != "Synthetic contract issue" || issue.State != "open" {
							t.Fatal("issue write DTO drift")
						}
					case "related-branches-update":
						if err := api.UpdateIssueRelatedBranches(v5, "fixture-owner", "fixture-repo", "1", []string{"main"}); err != nil {
							t.Fatal(err)
						}
					case "workflows":
						page := 1
						if ex.Name == "empty-page" {
							page = 2
						}
						result, err := v8.ListWorkflows("fixture-owner", "fixture-repo", actions.ListWorkflowsOptions{Page: page, PerPage: 2})
						if err != nil {
							t.Fatal(err)
						}
						if result.TotalCount != 1 {
							t.Fatal("workflow envelope drift")
						}
						if page == 1 {
							if len(result.Workflows) != 1 || result.Workflows[0].ID != "workflow-fixture" || result.Workflows[0].Name != "Synthetic CI" || result.Workflows[0].Path != ".gitcode/workflows/ci.yml" {
								t.Fatal("workflow DTO drift")
							}
						} else if len(result.Workflows) != 0 {
							t.Fatal("expected empty workflow page")
						}
					case "artifact-delete":
						if err := v8.DeleteArtifact("fixture-owner", "fixture-repo", "1"); err != nil {
							t.Fatal(err)
						}
					case "workflow-dispatch":
						if err := v8.CreateWorkflowDispatch("fixture-owner", "fixture-repo", "workflow-fixture", actions.WorkflowDispatchPayload{Ref: "main"}); err != nil {
							t.Fatal(err)
						}
					default:
						t.Fatal("fixture has no production-client replay")
					}
				})
			}
		})
	}
}
