package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) {
	panic("confirmation input was unexpectedly read")
}

func populatedEditResponse() *http.Response {
	return forkResponse(http.StatusOK, `{"name":"updated","full_name":"alice/updated","web_url":"https://atomgit.com/alice/updated"}`)
}

func runRepoEditCommand(t *testing.T, transport forkRoundTripFunc, args []string, input io.Reader, flags map[string]string) (string, error) {
	t.Helper()
	cmd := newCmdRepoEdit(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if input != nil {
		cmd.SetIn(input)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	err := cmd.RunE(cmd, args)
	return out.String(), err
}

func TestRepoEditBuildsExactPartialRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flags    map[string]string
		input    string
		wantBody map[string]interface{}
	}{
		{
			name:     "description only",
			args:     []string{"alice/demo"},
			flags:    map[string]string{"description": "updated"},
			wantBody: map[string]interface{}{"description": "updated"},
		},
		{
			name:     "clear description",
			args:     []string{"alice/demo"},
			flags:    map[string]string{"description": ""},
			wantBody: map[string]interface{}{"description": ""},
		},
		{
			name:  "all fields",
			args:  []string{"alice/demo"},
			input: "yes\n",
			flags: map[string]string{
				"name":           "updated",
				"description":    "new description",
				"default-branch": "trunk",
				"visibility":     "private",
			},
			wantBody: map[string]interface{}{
				"name":           "updated",
				"description":    "new description",
				"default_branch": "trunk",
				"private":        true,
			},
		},
		{
			name:     "public alias",
			args:     []string{"alice/demo"},
			flags:    map[string]string{"public": "true", "yes": "true"},
			wantBody: map[string]interface{}{"private": false},
		},
		{
			name:     "private alias",
			args:     []string{"alice/demo"},
			flags:    map[string]string{"private": "true", "yes": "true"},
			wantBody: map[string]interface{}{"private": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/alice/demo" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q", got)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, tt.wantBody) {
					t.Fatalf("body = %#v, want %#v", body, tt.wantBody)
				}
				return populatedEditResponse(), nil
			})

			output, err := runRepoEditCommand(t, transport, tt.args, strings.NewReader(tt.input), tt.flags)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d", requests)
			}
			if !strings.Contains(output, "alice/updated") || !strings.Contains(output, "https://atomgit.com/alice/updated") {
				t.Fatalf("output = %q", output)
			}
		})
	}
}

func TestRepoEditValidatesBeforeRequest(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		flags     map[string]string
		wantError string
	}{
		{name: "no updates", args: []string{"alice/demo"}, wantError: "at least one"},
		{name: "empty repository", args: []string{""}, flags: map[string]string{"description": "x"}, wantError: "invalid repository format"},
		{name: "short repository", args: []string{"demo"}, flags: map[string]string{"description": "x"}, wantError: "invalid repository format"},
		{name: "missing owner", args: []string{"/demo"}, flags: map[string]string{"description": "x"}, wantError: "invalid repository format"},
		{name: "missing repo", args: []string{"alice/"}, flags: map[string]string{"description": "x"}, wantError: "invalid repository format"},
		{name: "too many components", args: []string{"alice/demo/extra"}, flags: map[string]string{"description": "x"}, wantError: "invalid repository format"},
		{name: "blank name", args: []string{"alice/demo"}, flags: map[string]string{"name": " "}, wantError: "name cannot be empty"},
		{name: "dot name", args: []string{"alice/demo"}, flags: map[string]string{"name": "."}, wantError: "invalid repository name"},
		{name: "dot dot name", args: []string{"alice/demo"}, flags: map[string]string{"name": ".."}, wantError: "invalid repository name"},
		{name: "slash name", args: []string{"alice/demo"}, flags: map[string]string{"name": "bad/name"}, wantError: "invalid repository name"},
		{name: "backslash name", args: []string{"alice/demo"}, flags: map[string]string{"name": `bad\name`}, wantError: "invalid repository name"},
		{name: "control name", args: []string{"alice/demo"}, flags: map[string]string{"name": "bad\nname"}, wantError: "invalid repository name"},
		{name: "blank default branch", args: []string{"alice/demo"}, flags: map[string]string{"default-branch": " "}, wantError: "default branch cannot be empty"},
		{name: "invalid visibility", args: []string{"alice/demo"}, flags: map[string]string{"visibility": "PUBLIC"}, wantError: "invalid visibility"},
		{name: "internal visibility", args: []string{"alice/demo"}, flags: map[string]string{"visibility": "internal"}, wantError: "supports only public and private"},
		{name: "visibility and public", args: []string{"alice/demo"}, flags: map[string]string{"visibility": "public", "public": "true"}, wantError: "mutually exclusive"},
		{name: "visibility and public false", args: []string{"alice/demo"}, flags: map[string]string{"visibility": "private", "public": "false"}, wantError: "mutually exclusive"},
		{name: "public and private", args: []string{"alice/demo"}, flags: map[string]string{"public": "true", "private": "true"}, wantError: "mutually exclusive"},
		{name: "public false", args: []string{"alice/demo"}, flags: map[string]string{"public": "false"}, wantError: "--public=false is not supported"},
		{name: "private false", args: []string{"alice/demo"}, flags: map[string]string{"private": "false"}, wantError: "--private=false is not supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return populatedEditResponse(), nil
			})
			_, err := runRepoEditCommand(t, transport, tt.args, strings.NewReader("no\n"), tt.flags)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			if requests != 0 {
				t.Fatalf("requests = %d", requests)
			}
		})
	}
}

func TestRepoEditInfersRepository(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch || req.URL.Path != "/api/v5/repos/team/inferred" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{"name":"inferred","full_name":"team/inferred","web_url":"https://atomgit.com/team/inferred"}`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoEdit(factory)
	if err := cmd.Flags().Set("description", "updated"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRepoEditRejectsUnknownFlags(t *testing.T) {
	cmd := NewCmdRepo(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))
	cmd.SetArgs([]string{"edit", "alice/demo", "--enable-wiki"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoEditConfirmation(t *testing.T) {
	tests := []struct {
		name         string
		input        io.Reader
		flags        map[string]string
		wantRequests int
		wantCancel   bool
	}{
		{name: "confirmed y", input: strings.NewReader("y\n"), flags: map[string]string{"name": "updated"}, wantRequests: 1},
		{name: "confirmed yes uppercase", input: strings.NewReader("YES\n"), flags: map[string]string{"visibility": "private"}, wantRequests: 1},
		{name: "declined", input: strings.NewReader("no\n"), flags: map[string]string{"name": "updated"}, wantCancel: true},
		{name: "empty", input: strings.NewReader("\n"), flags: map[string]string{"private": "true"}, wantCancel: true},
		{name: "eof", input: strings.NewReader(""), flags: map[string]string{"public": "true"}, wantCancel: true},
		{name: "yes flag", input: panicReader{}, flags: map[string]string{"name": "updated", "yes": "true"}, wantRequests: 1},
		{name: "low risk", input: panicReader{}, flags: map[string]string{"description": "updated"}, wantRequests: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return populatedEditResponse(), nil
			})
			output, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, tt.input, tt.flags)
			if err != nil {
				t.Fatal(err)
			}
			if requests != tt.wantRequests {
				t.Fatalf("requests = %d, want %d", requests, tt.wantRequests)
			}
			if tt.wantCancel && !strings.Contains(output, "cancelled") {
				t.Fatalf("output = %q", output)
			}
		})
	}
}

func TestRepoEditRecoversEmptySuccessResponses(t *testing.T) {
	tests := []struct {
		name        string
		patchStatus int
		patchBody   string
	}{
		{name: "empty ok", patchStatus: http.StatusOK},
		{name: "no content", patchStatus: http.StatusNoContent},
		{name: "unusable representation", patchStatus: http.StatusOK, patchBody: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests = append(requests, req.Method+" "+req.URL.Path)
				switch req.Method {
				case http.MethodPatch:
					return forkResponse(tt.patchStatus, tt.patchBody), nil
				case http.MethodGet:
					return forkResponse(http.StatusOK, `{"name":"recovered","full_name":"alice/recovered","web_url":"https://atomgit.com/alice/recovered"}`), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.Path)
				}
				return nil, nil
			})
			output, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, nil, map[string]string{"description": "updated"})
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"PATCH /api/v5/repos/alice/demo", "GET /api/v5/repos/alice/demo"}
			if !reflect.DeepEqual(requests, want) {
				t.Fatalf("requests = %#v, want %#v", requests, want)
			}
			if !strings.Contains(output, "alice/recovered") || !strings.Contains(output, "https://atomgit.com/alice/recovered") {
				t.Fatalf("output = %q", output)
			}
		})
	}
}

func TestRepoEditRecoveryFailureReportsSplitOutcome(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			return forkResponse(http.StatusNoContent, ""), nil
		}
		return forkResponse(http.StatusForbidden, `{"message":"denied"}`), nil
	})
	_, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, nil, map[string]string{"description": "updated"})
	if err == nil || !strings.Contains(err.Error(), "was updated, but failed to read") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoEditUnusableRecoveryReportsSplitOutcome(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			return forkResponse(http.StatusNoContent, ""), nil
		}
		return forkResponse(http.StatusOK, `{}`), nil
	})
	_, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, nil, map[string]string{"description": "updated"})
	if err == nil || !strings.Contains(err.Error(), "was updated") || !strings.Contains(err.Error(), "did not return usable") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoEditReportsAPIErrors(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusUnprocessableEntity} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return forkResponse(status, `{"message":"invalid setting"}`), nil
			})
			_, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, nil, map[string]string{"description": "updated"})
			if err == nil ||
				!strings.Contains(err.Error(), "failed to edit repository alice/demo") ||
				!strings.Contains(err.Error(), fmt.Sprint(status)) ||
				!strings.Contains(err.Error(), "invalid setting") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRepoEditReportsClientConstructionError(t *testing.T) {
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil)
	factory.HttpClient = func() (*http.Client, error) {
		return nil, fmt.Errorf("client unavailable")
	}
	cmd := newCmdRepoEdit(factory)
	if err := cmd.Flags().Set("description", "updated"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil ||
		!strings.Contains(err.Error(), "failed to edit repository alice/demo") ||
		!strings.Contains(err.Error(), "client unavailable") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoEditDoesNotUseAPIURLAsBrowserURL(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantURL  string
	}{
		{
			name:     "derive from path",
			response: `{"name":"Updated name","path":"stable-path","url":"https://api.atomgit.com/api/v5/repos/alice/stable-path"}`,
			wantURL:  "https://atomgit.com/alice/stable-path",
		},
		{
			name:     "alternate html url",
			response: `{"name":"Updated name","html_url":"https://atomgit.com/alice/browser-path","url":"https://api.atomgit.com/api/v5/repos/alice/browser-path"}`,
			wantURL:  "https://atomgit.com/alice/browser-path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return forkResponse(http.StatusOK, tt.response), nil
			})
			output, err := runRepoEditCommand(t, transport, []string{"alice/demo"}, nil, map[string]string{"description": "updated"})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output, "api.atomgit.com") || !strings.Contains(output, tt.wantURL) {
				t.Fatalf("output = %q", output)
			}
		})
	}
}
