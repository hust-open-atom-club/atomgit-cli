package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestRepositoryInsightsEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		response string
		call     func(*Client) error
	}{
		{
			name: "languages", path: "/repos/team/demo/languages", response: `{"Go":98.5}`,
			call: func(client *Client) error {
				languages, err := GetRepositoryLanguages(client, "team", "demo")
				if err == nil && languages["Go"] != 98.5 {
					t.Fatalf("languages = %#v", languages)
				}
				return err
			},
		},
		{
			name: "contributors", path: "/repos/team/demo/contributors/statistic", response: `[{"name":"Ada","contributions":null,"overview":{"commit_count":4}}]`,
			call: func(client *Client) error {
				contributors, err := GetRepositoryContributorStatistics(client, "team", "demo")
				if err == nil && (len(contributors) != 1 || contributors[0].Contributions == nil || contributors[0].Overview.CommitCount != 4) {
					t.Fatalf("contributors = %#v", contributors)
				}
				return err
			},
		},
		{
			name: "events", path: "/repos/team/demo/events?page=2&per_page=7", response: `{"events":null,"has_next_page":1}`,
			call: func(client *Client) error {
				page, err := GetRepositoryEventsPage(client, "team", "demo", 2, 7)
				if err == nil && (page.Events == nil || !bool(page.HasNextPage)) {
					t.Fatalf("page = %#v", page)
				}
				return err
			},
		},
		{
			name: "downloads", path: "/repos/team/demo/download_statistics", response: `{"download_statistics_detail":[],"download_statistics_total":3,"download_statistics_history_total":9}`,
			call: func(client *Client) error {
				statistics, err := GetRepositoryDownloadStatistics(client, "team", "demo")
				if err == nil && (statistics.PeriodTotal != 3 || statistics.HistoryTotal != 9) {
					t.Fatalf("statistics = %#v", statistics)
				}
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RequestURI() != tt.path {
					t.Fatalf("request URI = %q, want %q", r.URL.RequestURI(), tt.path)
				}
				fmt.Fprint(w, tt.response)
			})
			if err := tt.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositoryInsightUsersPaginateAndHonorLimit(t *testing.T) {
	for _, tt := range []struct {
		name   string
		suffix string
		call   func(*Client) ([]RepositoryInsightUser, error)
	}{
		{name: "watchers", suffix: "/subscribers", call: func(client *Client) ([]RepositoryInsightUser, error) {
			return ListRepositoryWatchers(client, "team", "demo", 101)
		}},
		{name: "stargazers", suffix: "/stargazers", call: func(client *Client) ([]RepositoryInsightUser, error) {
			return ListRepositoryStargazers(client, "team", "demo", 101)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.URL.Path != "/repos/team/demo"+tt.suffix || r.URL.Query().Get("page") != fmt.Sprint(requests) || r.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request = %s", r.URL.RequestURI())
				}
				count := 100
				if requests == 2 {
					count = 2
				}
				fmt.Fprint(w, "[")
				for i := 0; i < count; i++ {
					if i > 0 {
						fmt.Fprint(w, ",")
					}
					fmt.Fprintf(w, `{"id":%d,"login":"user-%d"}`, (requests-1)*100+i, (requests-1)*100+i)
				}
				fmt.Fprint(w, "]")
			})
			users, err := tt.call(client)
			if err != nil {
				t.Fatal(err)
			}
			if len(users) != 101 || users[100].Login != "user-100" || requests != 2 {
				t.Fatalf("users = %d, last = %#v, requests = %d", len(users), users[len(users)-1], requests)
			}
		})
	}
}

func TestRepositoryInsightsMalformedAndErrorResponses(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "{") })
		_, err := GetRepositoryLanguages(client, "team", "demo")
		if err == nil {
			t.Fatal("expected malformed JSON error")
		}
	})

	t.Run("API error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, "upstream failed")
		})
		_, err := GetRepositoryDownloadStatistics(client, "team", "demo")
		if err == nil || !strings.Contains(err.Error(), "502") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGetRepositoryEventsPageRejectsInvalidPagination(t *testing.T) {
	if _, err := GetRepositoryEventsPage(NewClient("token"), "team", "demo", 0, 100); err == nil {
		t.Fatal("expected invalid page error")
	}
}
