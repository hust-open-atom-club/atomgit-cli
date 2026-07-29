package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type webhookTestConfig struct{}

func (webhookTestConfig) GetToken() (string, error) { return "token", nil }
func (webhookTestConfig) GetUser() (string, error)  { return "alice", nil }
func (webhookTestConfig) GetHost() string           { return "atomgit.com" }

type webhookRoundTripFunc func(*http.Request) (*http.Response, error)

func (f webhookRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRepoWebhookRegistersCommandsAndFlags(t *testing.T) {
	repoCmd := NewCmdRepo(&cmdutil.Factory{})
	registered, _, err := repoCmd.Find([]string{"webhook"})
	if err != nil || registered.Name() != "webhook" {
		t.Fatalf("repo webhook was not registered: command = %v, error = %v", registered, err)
	}

	cmd := newCmdRepoWebhook(&cmdutil.Factory{})
	want := map[string][]string{
		"list":   {"limit", "json"},
		"view":   {"json"},
		"create": {"url", "events", "encryption", "secret-env", "secret-file", "secret-stdin"},
		"edit":   {"url", "events", "encryption", "secret-env", "secret-file", "secret-stdin"},
		"delete": {"yes"},
		"test":   {"yes"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s help does not explain repository inference", name)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("%s --%s flag was not registered", name, flag)
			}
		}
	}
}

func TestRepoWebhookListPaginatesAndRedactsSecrets(t *testing.T) {
	requests := 0
	factory := webhookFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/hooks" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %v", req.URL.Query())
		}
		if req.URL.Query().Get("page") == "1" {
			items := make([]map[string]interface{}, 100)
			for index := range items {
				items[index] = map[string]interface{}{"id": index + 1, "url": "https://example.com/hook", "password": "top-secret", "push_events": true, "active": true}
			}
			body, err := json.Marshal(items)
			if err != nil {
				t.Fatal(err)
			}
			return webhookResponse(http.StatusOK, string(body)), nil
		}
		return webhookResponse(http.StatusOK, `[{"id":101,"url":"https://example.com/issues","password":"never-print","issues_events":1,"active":false}]`), nil
	})
	cmd := newCmdRepoWebhookList(factory)
	_ = cmd.Flags().Set("limit", "101")
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if strings.Contains(output.String(), "top-secret") || strings.Contains(output.String(), "never-print") || strings.Contains(output.String(), "password") {
		t.Fatalf("output leaked a secret field: %s", output.String())
	}
	var values []map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &values); err != nil || len(values) != 101 {
		t.Fatalf("JSON count = %d, error = %v", len(values), err)
	}
}

func TestRepoWebhookViewRedactsPassword(t *testing.T) {
	factory := webhookFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/hooks/42" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return webhookResponse(http.StatusOK, `{"id":42,"url":"https://example.com/hook","password":"hidden","active":true,"push_events":true,"merge_requests_events":1,"result":"ok","result_code":200}`), nil
	})
	for _, jsonOutput := range []bool{false, true} {
		cmd := newCmdRepoWebhookView(factory)
		if jsonOutput {
			_ = cmd.Flags().Set("json", "true")
		}
		var output bytes.Buffer
		cmd.SetOut(&output)
		if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), "hidden") || strings.Contains(output.String(), "password") {
			t.Fatalf("output leaked password: %s", output.String())
		}
		for _, want := range []string{"https://example.com/hook", "push", "merge-requests"} {
			if !strings.Contains(output.String(), want) {
				t.Errorf("output does not contain %q: %s", want, output.String())
			}
		}
	}
}

func TestRepoWebhookCreateMapsEventsAndEnvironmentSecret(t *testing.T) {
	const secret = "environment-secret"
	t.Setenv("WEBHOOK_SECRET", secret)
	requests := 0
	factory := webhookFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/hooks" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		var body map[string]interface{}
		decodeWebhookBody(t, req, &body)
		if body["url"] != "https://example.com/hook" || body["password"] != secret || body["encryption_type"] != float64(1) {
			t.Fatalf("body = %#v", body)
		}
		if body["push_events"] != true || body["issues_events"] != true || body["note_events"] != false || body["tag_push_events"] != false || body["merge_requests_events"] != false {
			t.Fatalf("event body = %#v", body)
		}
		return webhookResponse(http.StatusOK, `{"id":42,"url":"https://example.com/hook","password":"environment-secret"}`), nil
	})
	cmd := newCmdRepoWebhookCreate(factory)
	setWebhookFlags(t, cmd, map[string]string{
		"url": "https://example.com/hook", "events": "push,issues", "secret-env": "WEBHOOK_SECRET", "encryption": "signature",
	})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 || output.String() != "Created webhook #42 for https://example.com/hook\n" || strings.Contains(output.String(), secret) {
		t.Fatalf("requests = %d, output = %q", requests, output.String())
	}
}

func TestRepoWebhookCreateErrorOmitsResponseBody(t *testing.T) {
	const secret = "server-echoed-secret"
	t.Setenv("WEBHOOK_SECRET", secret)
	factory := webhookFactory(func(*http.Request) (*http.Response, error) {
		return webhookResponse(http.StatusBadRequest, `{"message":"invalid server-echoed-secret"}`), nil
	})
	cmd := newCmdRepoWebhookCreate(factory)
	setWebhookFlags(t, cmd, map[string]string{"url": "https://example.com/hook", "events": "push", "secret-env": "WEBHOOK_SECRET"})
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") || strings.Contains(err.Error(), secret) {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoWebhookReadErrorOmitsResponseBody(t *testing.T) {
	const secret = "stored-secret"
	factory := webhookFactory(func(*http.Request) (*http.Response, error) {
		return webhookResponse(http.StatusForbidden, `{"message":"denied","password":"stored-secret"}`), nil
	})
	cmd := newCmdRepoWebhookView(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "42"})
	if err == nil || !strings.Contains(err.Error(), "403 Forbidden") || strings.Contains(err.Error(), secret) {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoWebhookEditPreservesRequiredURLAndReplacesEvents(t *testing.T) {
	requests := 0
	factory := webhookFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/hooks/42" {
				t.Fatalf("request = %s %s", req.Method, req.URL.Path)
			}
			return webhookResponse(http.StatusOK, `{"id":42,"url":"https://example.com/current","push_events":true}`), nil
		}
		if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/alice/demo/hooks/42" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		var body map[string]interface{}
		decodeWebhookBody(t, req, &body)
		if body["url"] != "https://example.com/current" || body["push_events"] != false || body["issues_events"] != false || body["merge_requests_events"] != false {
			t.Fatalf("body = %#v", body)
		}
		return webhookResponse(http.StatusOK, `{"id":42,"url":"https://example.com/current"}`), nil
	})
	cmd := newCmdRepoWebhookEdit(factory)
	_ = cmd.Flags().Set("events", "none")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || output.String() != "Updated webhook #42\n" {
		t.Fatalf("requests = %d, output = %q", requests, output.String())
	}
}

func TestReadWebhookSecretSources(t *testing.T) {
	file := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(file, []byte("file-secret\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOOK_SECRET", "env-secret")
	tests := []struct {
		name    string
		input   string
		opts    webhookSecretOptions
		want    string
		wantErr string
	}{
		{name: "environment", opts: webhookSecretOptions{EnvName: "HOOK_SECRET"}, want: "env-secret"},
		{name: "file", opts: webhookSecretOptions{File: file}, want: "file-secret"},
		{name: "stdin", input: "stdin-secret\n", opts: webhookSecretOptions{Stdin: true}, want: "stdin-secret"},
		{name: "missing environment", opts: webhookSecretOptions{EnvName: "MISSING_HOOK_SECRET"}, wantErr: "is not set"},
		{name: "multiple", opts: webhookSecretOptions{EnvName: "HOOK_SECRET", File: file}, wantErr: "mutually exclusive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, provided, err := readWebhookSecret(strings.NewReader(test.input), test.opts)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || !provided || value != test.want {
				t.Fatalf("value = %q, provided = %t, error = %v", value, provided, err)
			}
		})
	}
}

func TestRepoWebhookDeleteAndTestConfirmBeforeRequests(t *testing.T) {
	for _, test := range []struct {
		name       string
		command    func(*cmdutil.Factory) *cobra.Command
		input      string
		yes        bool
		method     string
		pathSuffix string
		wantCalls  int
		wantOutput string
	}{
		{name: "delete cancelled", command: newCmdRepoWebhookDelete, input: "n\n", wantOutput: "Deletion cancelled.\n"},
		{name: "delete confirmed", command: newCmdRepoWebhookDelete, input: "yes\n", method: http.MethodDelete, pathSuffix: "/42", wantCalls: 1, wantOutput: "Deleted webhook #42\n"},
		{name: "test cancelled", command: newCmdRepoWebhookTest, input: "n\n", wantOutput: "Webhook test cancelled.\n"},
		{name: "test yes flag", command: newCmdRepoWebhookTest, yes: true, method: http.MethodPost, pathSuffix: "/42/tests", wantCalls: 1, wantOutput: "Sent test payload through webhook #42\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			factory := webhookFactory(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != test.method || !strings.HasSuffix(req.URL.Path, test.pathSuffix) {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return webhookResponse(http.StatusOK, `{}`), nil
			})
			cmd := test.command(factory)
			if test.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(test.input))
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
			if calls != test.wantCalls || !strings.HasSuffix(output.String(), test.wantOutput) {
				t.Fatalf("calls = %d, output = %q", calls, output.String())
			}
		})
	}
}

func TestRepoWebhookValidationHappensBeforeRequests(t *testing.T) {
	requests := 0
	factory := webhookFactory(func(*http.Request) (*http.Response, error) {
		requests++
		return webhookResponse(http.StatusOK, `{}`), nil
	})
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
		want string
	}{
		{name: "invalid id", cmd: func() *cobra.Command { return newCmdRepoWebhookView(factory) }, args: []string{"alice/demo", "zero"}, want: "invalid webhook id"},
		{name: "invalid url", cmd: func() *cobra.Command {
			cmd := newCmdRepoWebhookCreate(factory)
			setWebhookFlags(t, cmd, map[string]string{"url": "ftp://example.com/hook", "events": "push"})
			return cmd
		}, args: []string{"alice/demo"}, want: "invalid webhook URL"},
		{name: "invalid event", cmd: func() *cobra.Command {
			cmd := newCmdRepoWebhookCreate(factory)
			setWebhookFlags(t, cmd, map[string]string{"url": "https://example.com/hook", "events": "release"})
			return cmd
		}, args: []string{"alice/demo"}, want: "unsupported webhook event"},
		{name: "edit no settings", cmd: func() *cobra.Command { return newCmdRepoWebhookEdit(factory) }, args: []string{"alice/demo", "42"}, want: "at least one webhook setting"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.cmd()
			err := cmd.RunE(cmd, test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func webhookFactory(roundTrip webhookRoundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: webhookTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTrip}, nil
		},
	}
}

func webhookResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func decodeWebhookBody(t *testing.T, req *http.Request, target interface{}) {
	t.Helper()
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func setWebhookFlags(t *testing.T, cmd *cobra.Command, values map[string]string) {
	t.Helper()
	for name, value := range values {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
}
