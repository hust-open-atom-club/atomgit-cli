package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type eventCountingConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *eventCountingConfig) GetToken() (string, error) {
	c.tokenCalls++
	return "token", c.tokenErr
}

func (c *eventCountingConfig) GetUser() (string, error) { return "alice", nil }
func (c *eventCountingConfig) GetHost() string          { return "atomgit.com" }

func runEventsCommand(t *testing.T, factory *cmdutil.Factory, args []string, flags map[string]string, out *bytes.Buffer) error {
	t.Helper()
	cmd := newCmdUserEvents(factory)
	cmd.SetArgs(args)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	cmd.SetOut(out)
	return cmd.RunE(cmd, args)
}

func TestNewCmdUserRegistersEvents(t *testing.T) {
	cmd := NewCmdUser(&cmdutil.Factory{})
	events, _, err := cmd.Find([]string{"events"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"year", "limit", "json"} {
		if events.Flags().Lookup(flag) == nil {
			t.Fatalf("events flag %q was not registered", flag)
		}
	}
	if got := events.Flags().Lookup("year").DefValue; got != "0" {
		t.Fatalf("default year = %q, want 0", got)
	}
	if got := events.Flags().Lookup("limit").DefValue; got != "30" {
		t.Fatalf("default limit = %q, want 30", got)
	}
	if err := events.Args(events, []string{"one", "two"}); err == nil {
		t.Fatal("user events accepted more than one argument")
	}
}

func TestUserEventsValidationBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{name: "zero limit", flags: map[string]string{"limit": "0"}, want: "must be positive"},
		{name: "negative limit", flags: map[string]string{"limit": "-1"}, want: "must be positive"},
		{name: "year before epoch", flags: map[string]string{"year": "1969"}, want: "invalid year"},
		{name: "negative year", flags: map[string]string{"year": "-1"}, want: "invalid year"},
		{name: "year too large", flags: map[string]string{"year": "10000"}, want: "invalid year"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &eventCountingConfig{}
			var out bytes.Buffer
			err := runEventsCommand(t, &cmdutil.Factory{Config: config}, nil, tt.flags, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if config.tokenCalls != 0 {
				t.Fatalf("GetToken calls = %d, want 0", config.tokenCalls)
			}
		})
	}
}

func TestUserEventsRequiresAuthentication(t *testing.T) {
	config := &eventCountingConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")}
	var out bytes.Buffer
	err := runEventsCommand(t, &cmdutil.Factory{Config: config}, nil, nil, &out)
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
	if config.tokenCalls != 1 {
		t.Fatalf("GetToken calls = %d, want 1", config.tokenCalls)
	}
}

func TestUserEventsExplicitAndCurrentUsername(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{name: "explicit username", args: []string{"bob"}, wantPath: "/api/v5/users/bob/events"},
		{name: "current username", args: nil, wantPath: "/api/v5/users/alice/events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != tt.wantPath {
					t.Errorf("request = %s %s, want GET %s", req.Method, req.URL.Path, tt.wantPath)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"events":{"2026-08-27":[]},"next":""}`)
			})
			var out bytes.Buffer
			if err := runEventsCommand(t, factory, tt.args, nil, &out); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserEventsOnePageTextOrdering(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/users/alice/events" {
			t.Errorf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.RawQuery != "" {
			t.Errorf("query = %q, want empty", req.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"events":{
				"2026-08-25":[{"action":1,"action_name":"pushed to","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-25T10:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"main","target_type":"Branch","target_type_format":"branch"}],
				"2026-08-27":[{"action":6,"action_name":"commented on","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-27T09:00:00Z","project_id":20,"project_name":"alice/notes","target_id":21,"target_iid":22,"target_title":"hello","target_type":"Note","target_type_format":"note"}]
			},
			"next":""
		}`)
	})
	var out bytes.Buffer
	if err := runEventsCommand(t, factory, nil, nil, &out); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	first := strings.Index(output, "commented on")
	second := strings.Index(output, "pushed to")
	if first == -1 || second == -1 || first > second {
		t.Fatalf("output not ordered newest first:\n%s", output)
	}
	for _, want := range []string{"DATE", "ACTION", "PROJECT", "TITLE", "alice/notes", "hello", "alice/demo", "main"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestUserEventsMultipleCursorsAndYearQuery(t *testing.T) {
	requests := 0
	var queries []string
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		queries = append(queries, req.URL.RawQuery)
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/users/alice/events" {
			t.Errorf("request = %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			fmt.Fprint(w, `{"events":{"2026-08-27":[{"action":1,"action_name":"pushed to","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-27T09:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"main","target_type":"Branch","target_type_format":"branch"}]},"next":"cursor-1"}`)
		case 2:
			fmt.Fprint(w, `{"events":{"2026-08-26":[{"action":6,"action_name":"commented on","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-26T09:00:00Z","project_id":20,"project_name":"alice/notes","target_id":21,"target_iid":22,"target_title":"hello","target_type":"Note","target_type_format":"note"}]},"next":""}`)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	})
	var out bytes.Buffer
	err := runEventsCommand(t, factory, nil, map[string]string{"year": "2026"}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if queries[0] != "year=2026" {
		t.Fatalf("first query = %q, want year=2026", queries[0])
	}
	if queries[1] != "next=cursor-1&year=2026" {
		t.Fatalf("second query = %q, want next=cursor-1&year=2026", queries[1])
	}
	for _, want := range []string{"2026-08-27", "2026-08-26", "pushed to", "commented on"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestUserEventsLimitStopsAfterSatisfied(t *testing.T) {
	requests := 0
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			fmt.Fprint(w, `{"events":{"2026-08-27":[
				{"action":1,"action_name":"pushed to","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-27T09:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"main","target_type":"Branch","target_type_format":"branch"},
				{"action":6,"action_name":"commented on","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-27T08:00:00Z","project_id":20,"project_name":"alice/notes","target_id":21,"target_iid":22,"target_title":"hello","target_type":"Note","target_type_format":"note"}
			]},"next":"cursor-1"}`)
		case 2:
			fmt.Fprint(w, `{"events":{"2026-08-26":[
				{"action":1,"action_name":"pushed to","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-26T09:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"main","target_type":"Branch","target_type_format":"branch"},
				{"action":6,"action_name":"commented on","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-26T08:00:00Z","project_id":20,"project_name":"alice/notes","target_id":21,"target_iid":22,"target_title":"hello","target_type":"Note","target_type_format":"note"}
			]},"next":"cursor-2"}`)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	})
	var out bytes.Buffer
	if err := runEventsCommand(t, factory, nil, map[string]string{"limit": "3"}, &out); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (must not fetch after limit is satisfied)", requests)
	}
	lines := strings.Count(strings.TrimSpace(out.String()), "\n") + 1
	if lines != 4 { // header + 3 events
		t.Fatalf("output lines = %d, want 4:\n%s", lines, out.String())
	}
}

func TestUserEventsEmptyResults(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"events":{},"next":""}`)
		})
		var out bytes.Buffer
		if err := runEventsCommand(t, factory, nil, nil, &out); err != nil {
			t.Fatal(err)
		}
		if got := out.String(); got != "No events found.\n" {
			t.Fatalf("output = %q, want %q", got, "No events found.\n")
		}
	})

	t.Run("json", func(t *testing.T) {
		factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"events":{},"next":""}`)
		})
		var out bytes.Buffer
		if err := runEventsCommand(t, factory, nil, map[string]string{"json": "true"}, &out); err != nil {
			t.Fatal(err)
		}
		var events []eventJSON
		if err := json.Unmarshal(out.Bytes(), &events); err != nil {
			t.Fatalf("output is not a JSON array: %v\n%s", err, out.String())
		}
		if len(events) != 0 {
			t.Fatalf("events = %d, want 0", len(events))
		}
	})
}

func TestUserEventsRepeatedCursorDetected(t *testing.T) {
	requests := 0
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"events":{"2026-08-27":[{"action":1,"action_name":"pushed to","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":1,"author_username":"alice","created_at":"2026-08-27T09:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"main","target_type":"Branch","target_type_format":"branch"}]},"next":"same-cursor"}`)
	})
	var out bytes.Buffer
	err := runEventsCommand(t, factory, nil, map[string]string{"limit": "10"}, &out)
	if err == nil || !strings.Contains(err.Error(), "repeated cursor") {
		t.Fatalf("error = %v, want repeated cursor", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestUserEventsAPIError(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	var out bytes.Buffer
	err := runEventsCommand(t, factory, []string{"alice"}, nil, &out)
	if err == nil || !strings.Contains(err.Error(), "list events for user \"alice\"") {
		t.Fatalf("error = %v, want list events context", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want API body", err)
	}
}

func TestUserEventsJSONStableArray(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"events":{"2026-08-27":[{"action":6,"action_name":"commented on","author":{"name":"Alice","username":"alice","web_url":"https://atomgit.com/alice"},"author_id":7,"author_username":"alice","created_at":"2026-08-27T09:00:00Z","project_id":10,"project_name":"alice/demo","target_id":11,"target_iid":12,"target_title":"hello","target_type":"Note","target_type_format":"note"}]},"next":""}`)
	})
	var out bytes.Buffer
	if err := runEventsCommand(t, factory, nil, map[string]string{"json": "true"}, &out); err != nil {
		t.Fatal(err)
	}

	var events []eventJSON
	if err := json.Unmarshal(out.Bytes(), &events); err != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", err, out.String())
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	got := events[0]
	if got.Date != "2026-08-27" || got.Action != 6 || got.ActionName != "commented on" || got.AuthorID != 7 || got.AuthorUsername != "alice" || got.AuthorName != "Alice" || got.AuthorURL != "https://atomgit.com/alice" || got.CreatedAt != "2026-08-27T09:00:00Z" || got.ProjectID != 10 || got.ProjectName != "alice/demo" || got.TargetID != 11 || got.TargetIID != 12 || got.TargetTitle != "hello" || got.TargetType != "Note" || got.TargetTypeFormat != "note" {
		t.Fatalf("event = %+v", got)
	}
}
