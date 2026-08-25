package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type namespaceCountingConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *namespaceCountingConfig) GetToken() (string, error) {
	c.tokenCalls++
	return "token", c.tokenErr
}

func (c *namespaceCountingConfig) GetUser() (string, error) { return "alice", nil }
func (c *namespaceCountingConfig) GetHost() string          { return "atomgit.com" }

func namespaceTestFactory(t *testing.T, handler http.HandlerFunc) *cmdutil.Factory {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport := server.Client().Transport
	return userFactory(userTestConfig{}, userRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
		cloned.Host = target.Host
		return serverTransport.RoundTrip(cloned)
	}))
}

func runNamespacesCommand(t *testing.T, factory *cmdutil.Factory, flags map[string]string, out *bytes.Buffer) error {
	t.Helper()
	cmd := newCmdUserNamespaces(factory)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	cmd.SetOut(out)
	return cmd.RunE(cmd, nil)
}

func TestNewCmdUserRegistersNamespaces(t *testing.T) {
	cmd := NewCmdUser(&cmdutil.Factory{})
	namespaces, _, err := cmd.Find([]string{"namespaces"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"mode", "limit", "json"} {
		if namespaces.Flags().Lookup(flag) == nil {
			t.Fatalf("namespaces flag %q was not registered", flag)
		}
	}
	if got := namespaces.Flags().Lookup("mode").DefValue; got != "intrant" {
		t.Fatalf("default mode = %q, want intrant", got)
	}
	if got := namespaces.Flags().Lookup("limit").DefValue; got != "30" {
		t.Fatalf("default limit = %q, want 30", got)
	}
	if err := namespaces.Args(namespaces, []string{"unexpected"}); err == nil {
		t.Fatal("user namespaces accepted an argument")
	}
}

func TestUserNamespacesModesAndExactQueryNames(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		setMode bool
	}{
		{name: "default intrant", mode: "intrant"},
		{name: "explicit intrant", mode: "intrant", setMode: true},
		{name: "project", mode: "project", setMode: true},
		{name: "all", mode: "all", setMode: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/user/namespaces" {
					t.Errorf("request = %s %s", req.Method, req.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				wantQuery := "mode=" + tt.mode + "&page=1&perPage=100"
				if req.URL.RawQuery != wantQuery {
					t.Errorf("query = %q, want %q", req.URL.RawQuery, wantQuery)
				}
				if _, exists := req.URL.Query()["per_page"]; exists {
					t.Errorf("query unexpectedly used per_page: %q", req.URL.RawQuery)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Errorf("Authorization = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `[]`)
			})
			flags := map[string]string{}
			if tt.setMode {
				flags["mode"] = tt.mode
			}
			var out bytes.Buffer
			if err := runNamespacesCommand(t, factory, flags, &out); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
		})
	}
}

func TestUserNamespacesValidationBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{name: "unknown mode", flags: map[string]string{"mode": "member"}, want: "invalid mode"},
		{name: "uppercase mode", flags: map[string]string{"mode": "INTRANT"}, want: "invalid mode"},
		{name: "zero limit", flags: map[string]string{"limit": "0"}, want: "must be positive"},
		{name: "negative limit", flags: map[string]string{"limit": "-1"}, want: "must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &namespaceCountingConfig{}
			var out bytes.Buffer
			err := runNamespacesCommand(t, &cmdutil.Factory{Config: config}, tt.flags, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if config.tokenCalls != 0 {
				t.Fatalf("GetToken calls = %d, want 0", config.tokenCalls)
			}
		})
	}
}

func TestUserNamespacesRequiresAuthentication(t *testing.T) {
	config := &namespaceCountingConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")}
	var out bytes.Buffer
	err := runNamespacesCommand(t, &cmdutil.Factory{Config: config}, nil, &out)
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
	if config.tokenCalls != 1 {
		t.Fatalf("GetToken calls = %d, want 1", config.tokenCalls)
	}
}

func TestUserNamespacesTextOutputDistinguishesTypes(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id":1,"path":"alice","name":"Alice","html_url":"https://atomgit.com/alice","type":"user"},
			{"id":2,"path":"open-source","name":"Open Source","html_url":"https://atomgit.com/open-source","type":"group"}
		]`)
	})
	var out bytes.Buffer
	if err := runNamespacesCommand(t, factory, nil, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PATH", "NAME", "TYPE", "URL", "alice", "Alice", "user", "open-source", "Open Source", "group"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestUserNamespacesJSONHasStableFields(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":7,"path":"team","name":"Team","html_url":"https://atomgit.com/team","type":"group","extra":"ignored"}]`)
	})
	var out bytes.Buffer
	if err := runNamespacesCommand(t, factory, map[string]string{"json": "true"}, &out); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out.String(), err)
	}
	want := []map[string]any{{
		"id":   float64(7),
		"path": "team",
		"name": "Team",
		"url":  "https://atomgit.com/team",
		"type": "group",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON = %#v, want %#v", got, want)
	}
}

func TestUserNamespacesPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		if got := req.URL.Query().Get("page"); got != strconv.Itoa(requests) {
			t.Errorf("page = %q, want %d", got, requests)
		}
		if got := req.URL.Query().Get("perPage"); got != "100" {
			t.Errorf("perPage = %q, want 100", got)
		}
		count := 100
		start := 1
		if requests == 2 {
			count = 2
			start = 101
		}
		items := make([]api.Namespace, count)
		for index := range items {
			id := start + index
			items[index] = api.Namespace{ID: int64(id), Path: fmt.Sprintf("namespace-%d", id)}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(items); err != nil {
			t.Errorf("encode response: %v", err)
		}
	})
	var out bytes.Buffer
	if err := runNamespacesCommand(t, factory, map[string]string{"limit": "101", "json": "true"}, &out); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	var got []namespaceJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 101 {
		t.Fatalf("namespaces length = %d, want 101", len(got))
	}
	if got[100].Path != "namespace-101" {
		t.Fatalf("last namespace = %#v", got[100])
	}
}

func TestUserNamespacesEmptyResponses(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{name: "text", want: "No namespaces found.\n"},
		{name: "json", flags: map[string]string{"json": "true"}, want: "[]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `[]`)
			})
			var out bytes.Buffer
			if err := runNamespacesCommand(t, factory, tt.flags, &out); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(out.String()); got != strings.TrimSpace(tt.want) {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestUserNamespacesReportsAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "forbidden", status: http.StatusForbidden, body: `{}`, want: "403"},
		{name: "malformed response", status: http.StatusOK, body: `{`, want: "unexpected EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			})
			var out bytes.Buffer
			err := runNamespacesCommand(t, factory, nil, &out)
			if err == nil || !strings.Contains(err.Error(), "failed to list namespaces") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want context and %q", err, tt.want)
			}
		})
	}
}

func TestUserNamespacesSanitizesTextOutput(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":1,"path":"team\tpath","name":"Evil\u001b[31m\nName","html_url":"https://atomgit.com/team","type":"group\rtype"}]`)
	})
	var raw bytes.Buffer
	sanitized := cmdutil.NewSanitizingWriter(&raw)
	cmd := newCmdUserNamespaces(factory)
	cmd.SetOut(sanitized)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := sanitized.Flush(); err != nil {
		t.Fatal(err)
	}
	got := raw.String()
	if strings.ContainsRune(got, '\x1b') || strings.Contains(got, "team\tpath") || strings.Contains(got, "Evil\nName") {
		t.Fatalf("control bytes were not sanitized: %q", got)
	}
	for _, want := range []string{`team\tpath`, `Evil\x1b[31m\nName`, `group\rtype`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, missing %q", got, want)
		}
	}
}
