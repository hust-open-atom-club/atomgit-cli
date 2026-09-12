package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runInsightCommand(t *testing.T, name string, args []string, transport forkRoundTripFunc) (string, error) {
	t.Helper()
	group := newCmdRepoInsights(repoFactory(repoCommandConfig{token: "token"}, transport))
	cmd, _, err := group.Find([]string{name})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if err := cmd.Flags().Parse(args); err != nil {
		return out.String(), err
	}
	err = cmd.RunE(cmd, cmd.Flags().Args())
	return out.String(), err
}

func TestRepoInsightsLanguagesOrderingEmptyAndSanitization(t *testing.T) {
	t.Run("ordered and sanitized", func(t *testing.T) {
		transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v5/repos/team/demo/languages" {
				t.Fatalf("path = %q", req.URL.Path)
			}
			return forkResponse(http.StatusOK, `{"Zig":1,"Go":90,"Bad\tName\nNext":90}`), nil
		})
		out, err := runInsightCommand(t, "languages", []string{"team/demo"}, transport)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, `Bad\tName\nNext`) || strings.Index(out, `Bad\tName\nNext`) > strings.Index(out, "Go") || strings.Index(out, "Go") > strings.Index(out, "Zig") {
			t.Fatalf("output = %q", out)
		}
	})

	t.Run("empty JSON", func(t *testing.T) {
		out, err := runInsightCommand(t, "languages", []string{"team/demo", "--json"}, func(*http.Request) (*http.Response, error) {
			return forkResponse(http.StatusOK, `{}`), nil
		})
		if err != nil || strings.TrimSpace(out) != "[]" {
			t.Fatalf("output = %q, error = %v", out, err)
		}
	})
}

func TestRepoInsightsContributorsUsesStatisticsAndHonorsLimit(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/team/demo/contributors/statistic" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[
			{"name":"Few","overview":{"commit_count":1},"contributions":[]},
			{"name":"Many","email":"many@example.test","overview":{"additions":7,"deletions":2,"total_changes":9,"commit_count":5},"contributions":[{"date":"2026-01-01","commit_count":1},{"date":"2026-02-01","commit_count":4}]}
		]`), nil
	})
	out, err := runInsightCommand(t, "contributors", []string{"team/demo", "--limit", "1", "--json"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Few") || !strings.Contains(out, `"name": "Many"`) || strings.Index(out, "2026-02-01") > strings.Index(out, "2026-01-01") {
		t.Fatalf("output = %s", out)
	}
}

func TestRepoInsightsEventsPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/api/v5/repos/team/demo/events" || req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("request = %s", req.URL.RequestURI())
		}
		if requests == 1 {
			return forkResponse(http.StatusOK, `{"events":[{"action":1,"created_at":"a","target_id":1},{"action":2,"created_at":"b","target_id":2}],"has_next_page":true}`), nil
		}
		return forkResponse(http.StatusOK, `{"events":[{"action":3,"created_at":"c","target_id":3},{"action":4,"created_at":"d","target_id":4}],"has_next_page":false}`), nil
	})
	out, err := runInsightCommand(t, "events", []string{"team/demo", "--limit", "3", "--json"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || strings.Count(out, `"action":`) != 3 || strings.Contains(out, `"action": 4`) {
		t.Fatalf("requests = %d, output = %s", requests, out)
	}
}

func TestRepoInsightsEventsRejectsPaginationWithoutProgress(t *testing.T) {
	response := `{"events":[],"has_next_page":true}`
	_, err := runInsightCommand(t, "events", []string{"team/demo"}, func(*http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, response), nil
	})
	if err == nil || !strings.Contains(err.Error(), "made no progress") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoInsightsWatchersAndStargazers(t *testing.T) {
	for _, tt := range []struct {
		name      string
		path      string
		timestamp string
		jsonField string
	}{
		{name: "watchers", path: "/api/v5/repos/team/demo/subscribers", timestamp: "watch_at", jsonField: "watchedAt"},
		{name: "stargazers", path: "/api/v5/repos/team/demo/stargazers", timestamp: "starred_at", jsonField: "starredAt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != tt.path || req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request = %s", req.URL.RequestURI())
				}
				return forkResponse(http.StatusOK, fmt.Sprintf(`[{"id":7,"login":"ada","name":"Ada","%s":"2026-01-02"}]`, tt.timestamp)), nil
			})
			out, err := runInsightCommand(t, tt.name, []string{"team/demo", "--json"}, transport)
			if err != nil || !strings.Contains(out, `"login": "ada"`) || !strings.Contains(out, `"`+tt.jsonField+`": "2026-01-02"`) {
				t.Fatalf("output = %q, error = %v", out, err)
			}
		})
	}
}

func TestRepoInsightsDownloadsOrderingAndEmpty(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/team/demo/download_statistics" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{"download_statistics_detail":[{"pdate":"2026-01-01","today_dl_cnt":1},{"pdate":"2026-02-01","today_dl_cnt":2}],"download_statistics_total":3,"download_statistics_history_total":10}`), nil
	})
	out, err := runInsightCommand(t, "downloads", []string{"team/demo", "--json"}, transport)
	if err != nil || strings.Index(out, "2026-02-01") > strings.Index(out, "2026-01-01") || !strings.Contains(out, `"historyDownloads": 10`) {
		t.Fatalf("output = %q, error = %v", out, err)
	}

	out, err = runInsightCommand(t, "downloads", []string{"team/demo"}, func(*http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{}`), nil
	})
	if err != nil || out != "Period downloads: 0\nHistory downloads: 0\n" {
		t.Fatalf("empty output = %q, error = %v", out, err)
	}
}

func TestRepoInsightsListCommandsRejectInvalidLimitBeforeAuth(t *testing.T) {
	for _, name := range []string{"contributors", "events", "watchers", "stargazers"} {
		t.Run(name, func(t *testing.T) {
			cfg := &repoRecordingConfig{}
			group := newCmdRepoInsights(&cmdutil.Factory{Config: cfg})
			cmd, _, err := group.Find([]string{name})
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Flags().Set("limit", "0"); err != nil {
				t.Fatal(err)
			}
			err = cmd.RunE(cmd, []string{"team/demo"})
			if err == nil || !strings.Contains(err.Error(), "invalid limit") || cfg.getTokenCalls != 0 {
				t.Fatalf("error = %v, token calls = %d", err, cfg.getTokenCalls)
			}
		})
	}
}

func TestRepoInsightsAPIErrorsIncludeRepositoryContext(t *testing.T) {
	_, err := runInsightCommand(t, "languages", []string{"team/demo"}, func(*http.Request) (*http.Response, error) {
		return forkResponse(http.StatusBadGateway, "failed"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "get languages for team/demo") {
		t.Fatalf("error = %v", err)
	}
}
