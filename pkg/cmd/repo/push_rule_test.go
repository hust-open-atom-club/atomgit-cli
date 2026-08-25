package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func pushRuleTestFactory(t *testing.T, handler http.HandlerFunc) *cmdutil.Factory {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport := server.Client().Transport
	return repoFactory(repoCommandConfig{token: "token", user: "alice"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
		cloned.Host = target.Host
		return serverTransport.RoundTrip(cloned)
	}))
}

func runPushRuleView(t *testing.T, factory *cmdutil.Factory, args []string, jsonOutput bool) (string, error) {
	t.Helper()
	cmd := newCmdRepoPushRuleView(factory)
	if jsonOutput {
		if err := cmd.Flags().Set("json", "true"); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := cmd.RunE(cmd, args)
	return out.String(), err
}

func runPushRuleEdit(t *testing.T, factory *cmdutil.Factory, args []string, input io.Reader, flags map[string]string) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoPushRuleEdit(factory)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	if input != nil {
		cmd.SetIn(input)
	}
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.RunE(cmd, args)
	return out.String(), errOut.String(), err
}

func TestRepoPushRuleViewTextAndJSON(t *testing.T) {
	factory := pushRuleTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/push_config" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"reject_not_signed_by_gpg":1,"commit_message_regex":"^(feat|fix): ","max_file_size":25,"skip_rule_for_owner":"false","deny_force_push":true}`)
	})

	textOutput, err := runPushRuleView(t, factory, []string{"alice/demo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Repository: alice/demo",
		"Reject unsigned commits: true",
		`Commit message regex: "^(feat|fix): "`,
		"Maximum file size (MB): 25",
		"Skip rules for repository owner: false",
		"Deny force pushes: true",
	} {
		if !strings.Contains(textOutput, want) {
			t.Errorf("text output missing %q:\n%s", want, textOutput)
		}
	}

	jsonOutput, err := runPushRuleView(t, factory, []string{"alice/demo"}, true)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonOutput, err)
	}
	want := map[string]interface{}{
		"repository":               "alice/demo",
		"reject_not_signed_by_gpg": true,
		"commit_message_regex":     "^(feat|fix): ",
		"max_file_size":            float64(25),
		"skip_rule_for_owner":      false,
		"deny_force_push":          true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON = %#v, want %#v", got, want)
	}
}

func TestRepoPushRuleEditSendsExactPartialRequests(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]string
		wantBody map[string]interface{}
	}{
		{
			name:     "explicit false",
			flags:    map[string]string{"reject-not-signed-by-gpg": "false", "yes": "true"},
			wantBody: map[string]interface{}{"reject_not_signed_by_gpg": false},
		},
		{
			name:     "explicit empty regex",
			flags:    map[string]string{"commit-message-regex": "", "yes": "true"},
			wantBody: map[string]interface{}{"commit_message_regex": ""},
		},
		{
			name:     "explicit zero",
			flags:    map[string]string{"max-file-size": "0", "yes": "true"},
			wantBody: map[string]interface{}{"max_file_size": float64(0)},
		},
		{
			name: "all documented fields",
			flags: map[string]string{
				"reject-not-signed-by-gpg": "true",
				"commit-message-regex":     "^PROJ-[0-9]+ ",
				"max-file-size":            "50",
				"skip-rule-for-owner":      "false",
				"deny-force-push":          "true",
				"yes":                      "true",
				"json":                     "true",
			},
			wantBody: map[string]interface{}{
				"reject_not_signed_by_gpg": true,
				"commit_message_regex":     "^PROJ-[0-9]+ ",
				"max_file_size":            float64(50),
				"skip_rule_for_owner":      false,
				"deny_force_push":          true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := pushRuleTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				requests++
				if req.Method != http.MethodPut || req.URL.Path != "/api/v5/repos/alice/demo/push_config" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
					t.Fatalf("Content-Type = %q", contentType)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, tt.wantBody) {
					t.Fatalf("body = %#v, want %#v", body, tt.wantBody)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			})

			out, errOut, err := runPushRuleEdit(t, factory, []string{"alice/demo"}, panicReader{}, tt.flags)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 || errOut != "" {
				t.Fatalf("requests = %d, stderr = %q", requests, errOut)
			}
			if !strings.Contains(out, "alice/demo") {
				t.Fatalf("output = %q", out)
			}
			for field := range tt.wantBody {
				if !strings.Contains(out, field) {
					t.Errorf("output missing changed field %q: %q", field, out)
				}
			}
		})
	}
}

func TestRepoPushRuleEditConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		input      io.Reader
		flags      map[string]string
		wantWrites int
		wantPrompt bool
		wantCancel bool
	}{
		{name: "confirmed", input: strings.NewReader("yes\n"), flags: map[string]string{"deny-force-push": "true"}, wantWrites: 1, wantPrompt: true},
		{name: "cancelled", input: strings.NewReader("no\n"), flags: map[string]string{"deny-force-push": "true"}, wantPrompt: true, wantCancel: true},
		{name: "empty response", input: strings.NewReader("\n"), flags: map[string]string{"commit-message-regex": ""}, wantPrompt: true, wantCancel: true},
		{name: "yes flag", input: panicReader{}, flags: map[string]string{"skip-rule-for-owner": "false", "yes": "true"}, wantWrites: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			factory := pushRuleTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				writes++
				fmt.Fprint(w, `{}`)
			})
			out, errOut, err := runPushRuleEdit(t, factory, []string{"alice/demo"}, tt.input, tt.flags)
			if err != nil {
				t.Fatal(err)
			}
			if writes != tt.wantWrites {
				t.Fatalf("writes = %d, want %d", writes, tt.wantWrites)
			}
			if tt.wantPrompt && (!strings.Contains(errOut, "Repository: alice/demo") || !strings.Contains(errOut, "Changed fields:") || !strings.Contains(errOut, "Apply these push-rule changes?")) {
				t.Fatalf("confirmation prompt = %q", errOut)
			}
			if tt.wantCancel {
				if out != "" || !strings.Contains(errOut, "cancelled") {
					t.Fatalf("stdout = %q, stderr = %q", out, errOut)
				}
			} else {
				if !strings.Contains(out, "alice/demo") {
					t.Fatalf("stdout = %q", out)
				}
				if !tt.wantPrompt && errOut != "" {
					t.Fatalf("stderr = %q, want empty", errOut)
				}
			}
		})
	}
}

func TestRepoPushRuleEditValidatesBeforeRequest(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		wantError string
	}{
		{name: "no changes", flags: map[string]string{}, wantError: "at least one"},
		{name: "negative maximum", flags: map[string]string{"max-file-size": "-1"}, wantError: "cannot be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := pushRuleTestFactory(t, func(http.ResponseWriter, *http.Request) { requests++ })
			_, _, err := runPushRuleEdit(t, factory, []string{"alice/demo"}, strings.NewReader("yes\n"), tt.flags)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			if requests != 0 {
				t.Fatalf("requests = %d", requests)
			}
		})
	}
}

func TestRepoPushRuleReportsMissingAndPermissionErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			factory := pushRuleTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"message":"unavailable"}`, status)
			})
			_, err := runPushRuleView(t, factory, []string{"alice/demo"}, false)
			if err == nil || !strings.Contains(err.Error(), http.StatusText(status)) || !strings.Contains(err.Error(), "alice/demo") {
				t.Fatalf("view error = %v", err)
			}

			_, _, err = runPushRuleEdit(t, factory, []string{"alice/demo"}, panicReader{}, map[string]string{"deny-force-push": "true", "yes": "true"})
			if err == nil || !strings.Contains(err.Error(), http.StatusText(status)) || !strings.Contains(err.Error(), "alice/demo") {
				t.Fatalf("edit error = %v", err)
			}
		})
	}
}

func TestRepoPushRuleInfersRepository(t *testing.T) {
	factory := pushRuleTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v5/repos/team/inferred/push_config" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		fmt.Fprint(w, `{}`)
	})
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	if _, err := runPushRuleView(t, factory, nil, false); err != nil {
		t.Fatal(err)
	}
}

func TestRepoPushRuleOutputEscapesTerminalControls(t *testing.T) {
	factory := pushRuleTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"commit_message_regex\":\"unsafe \\u001b]52;c;attack\\u0007\"}")
	})
	out, err := runPushRuleView(t, factory, []string{"alice/demo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out, "\x1b\x07") || !strings.Contains(out, `\x1b]52;c;attack\a`) {
		t.Fatalf("unsafe output = %q", out)
	}
}
