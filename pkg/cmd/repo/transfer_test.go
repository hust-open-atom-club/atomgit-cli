package repo

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runTransferCommand(t *testing.T, factory *cmdutil.Factory, args []string, input io.Reader) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoTransfer(factory)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if input != nil {
		cmd.SetIn(input)
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestRepoTransferUserOwnedRepository(t *testing.T) {
	var requests []string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		switch len(requests) {
		case 1:
			return forkResponse(http.StatusOK, `{"name":"demo","full_name":"alice/demo","web_url":"https://atomgit.com/alice/demo","owner":{"login":"alice","type":"User"}}`), nil
		case 2:
			if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/transfer" {
				t.Fatalf("transfer request = %s %s", req.Method, req.URL.Path)
			}
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, map[string]string{"new_owner": "bob"}) {
				t.Fatalf("body = %#v", body)
			}
			return forkResponse(http.StatusOK, `{"new_owner":"bob","new_name":"renamed"}`), nil
		case 3:
			if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/bob/renamed" {
				t.Fatalf("read-back request = %s %s", req.Method, req.URL.Path)
			}
			return forkResponse(http.StatusOK, `{"name":"renamed","full_name":"bob/renamed","web_url":"https://atomgit.com/bob/renamed","owner":{"login":"bob","type":"User"}}`), nil
		default:
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	out, errOut, err := runTransferCommand(t, repoFactory(repoCommandConfig{token: "token"}, transport), []string{"alice/demo", "--to", "bob"}, strings.NewReader("yes\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"GET /api/v5/repos/alice/demo", "POST /api/v5/repos/alice/demo/transfer", "GET /api/v5/repos/bob/renamed"}; !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	for _, value := range []string{"Source: alice/demo", "Destination: bob", "bob/renamed", "https://atomgit.com/bob/renamed", "Local Git remotes were not changed"} {
		if !strings.Contains(out, value) {
			t.Fatalf("output %q missing %q", out, value)
		}
	}
	if !strings.Contains(errOut, "Repository URLs and access may change") {
		t.Fatalf("confirmation output = %q", errOut)
	}
}

func TestRepoTransferOrganizationOwnedRepository(t *testing.T) {
	const password = "organization-secret"
	var requests []string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		switch len(requests) {
		case 1:
			return forkResponse(http.StatusOK, `{"name":"demo","namespace":{"path":"source-org"},"web_url":"https://atomgit.com/source-org/demo"}`), nil
		case 2:
			if req.Method != http.MethodPost || req.URL.Path != "/api/v5/org/source-org/projects/demo/transfer" {
				t.Fatalf("transfer request = %s %s", req.Method, req.URL.Path)
			}
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, map[string]string{"transfer_to": "target-org", "password": password}) {
				t.Fatalf("body = %#v", body)
			}
			return forkResponse(http.StatusOK, `{"code":1,"msg":"success"}`), nil
		case 3:
			return forkResponse(http.StatusOK, `{"name":"demo","namespace":{"path":"target-org"},"web_url":"https://atomgit.com/target-org/demo"}`), nil
		default:
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	out, errOut, err := runTransferCommand(t, repoFactory(repoCommandConfig{token: "token"}, transport), []string{"source-org/demo", "--to", "target-org", "--yes", "--password-stdin"}, strings.NewReader(password+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 || !strings.Contains(out, "target-org/demo") {
		t.Fatalf("requests = %#v, output = %q", requests, out)
	}
	if strings.Contains(out, password) || strings.Contains(errOut, password) {
		t.Fatalf("password leaked: stdout=%q stderr=%q", out, errOut)
	}
}

func TestRepoTransferCancellationSendsNoPost(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet {
			t.Fatalf("cancellation sent %s %s", req.Method, req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{"name":"demo","owner":{"login":"alice","type":"User"}}`), nil
	})

	out, _, err := runTransferCommand(t, repoFactory(repoCommandConfig{token: "token"}, transport), []string{"alice/demo", "--to", "bob"}, strings.NewReader("no\n"))
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || !strings.Contains(out, "Transfer cancelled") {
		t.Fatalf("requests = %d, output = %q", requests, out)
	}
}

func TestRepoTransferValidatesBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing destination", args: []string{"alice/demo"}, want: "required flag"},
		{name: "empty destination", args: []string{"alice/demo", "--to", " "}, want: "must not be empty"},
		{name: "unsafe destination", args: []string{"alice/demo", "--to", "bad/name"}, want: "invalid destination"},
		{name: "invalid repository", args: []string{"demo", "--to", "bob"}, want: "invalid repository format"},
		{name: "stdin without yes", args: []string{"alice/demo", "--to", "bob", "--password-stdin"}, want: "requires --yes"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("validation unexpectedly sent an HTTP request")
				return nil, nil
			}))
			_, _, err := runTransferCommand(t, factory, test.args, strings.NewReader(""))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
			if strings.Contains(err.Error(), "not authenticated") {
				t.Fatalf("authentication preceded validation: %v", err)
			}
		})
	}
}

func TestRepoTransferInfersRepositoryAndYesSkipsPrompt(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			return forkResponse(http.StatusOK, `{"name":"demo","owner":{"login":"alice","type":"User"}}`), nil
		case 2:
			return forkResponse(http.StatusOK, `{"new_owner":"bob","new_name":"demo"}`), nil
		case 3:
			return forkResponse(http.StatusOK, `{"name":"demo","owner":{"login":"bob","type":"User"},"web_url":"https://atomgit.com/bob/demo"}`), nil
		default:
			t.Fatal("unexpected request")
			return nil, nil
		}
	})
	factory := repoFactory(repoCommandConfig{token: "token"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
	}
	if _, _, err := runTransferCommand(t, factory, []string{"--to", "bob", "--yes"}, panicReader{}); err != nil {
		t.Fatal(err)
	}
}

func TestRepoTransferReportsContextualAPIErrorsWithoutSecrets(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		failSource bool
	}{
		{name: "source not found", status: http.StatusNotFound, failSource: true},
		{name: "destination permission", status: http.StatusForbidden},
		{name: "name conflict", status: http.StatusConflict},
		{name: "unprocessable", status: http.StatusUnprocessableEntity},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if test.failSource || requests == 2 {
					return forkResponse(test.status, `{"message":"denied","password":"do-not-print"}`), nil
				}
				return forkResponse(http.StatusOK, `{"name":"demo","owner":{"login":"alice","type":"User"}}`), nil
			})
			_, _, err := runTransferCommand(t, repoFactory(repoCommandConfig{token: "token"}, transport), []string{"alice/demo", "--to", "bob", "--yes"}, panicReader{})
			if err == nil {
				t.Fatal("expected transfer error")
			}
			if test.failSource && !strings.Contains(err.Error(), "inspect source repository") {
				t.Fatalf("error = %v", err)
			}
			if !test.failSource && !strings.Contains(err.Error(), "failed to transfer repository") {
				t.Fatalf("error = %v", err)
			}
			if strings.Contains(err.Error(), "do-not-print") {
				t.Fatalf("error leaked password: %v", err)
			}
		})
	}
}

func TestRepoTransferRejectsAmbiguousSuccess(t *testing.T) {
	tests := []struct {
		name         string
		responses    []string
		wantError    string
		organization bool
	}{
		{name: "mismatched response owner", responses: []string{`{"name":"demo","owner":{"login":"alice","type":"User"}}`, `{"new_owner":"mallory","new_name":"demo"}`}, wantError: "reported destination owner"},
		{name: "mismatched read back", responses: []string{`{"name":"demo","owner":{"login":"alice","type":"User"}}`, `{"new_owner":"bob","new_name":"demo"}`, `{"name":"other","owner":{"login":"bob","type":"User"},"web_url":"https://atomgit.com/bob/other"}`}, wantError: "final state is ambiguous"},
		{name: "missing read-back URL", responses: []string{`{"name":"demo","owner":{"login":"alice","type":"User"}}`, `{"new_owner":"bob","new_name":"demo"}`, `{"name":"demo","owner":{"login":"bob","type":"User"}}`}, wantError: "omitted the repository URL"},
		{name: "organization code", responses: []string{`{"name":"demo","namespace":{"path":"source-org"}}`, `{"code":0,"msg":"unknown"}`}, wantError: "returned code 0", organization: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := 0
			transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
				response := test.responses[request]
				request++
				return forkResponse(http.StatusOK, response), nil
			})
			args := []string{"alice/demo", "--to", "bob", "--yes"}
			input := io.Reader(panicReader{})
			if test.organization {
				args = []string{"source-org/demo", "--to", "bob", "--yes", "--password-stdin"}
				input = strings.NewReader("secret\n")
			}
			_, _, err := runTransferCommand(t, repoFactory(repoCommandConfig{token: "token"}, transport), args, input)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestClassifyRepositoryOwner(t *testing.T) {
	organization := api.Repository{}
	organization.Namespace.Path = "team"
	organization.Owner.Login = "alice"
	organization.Owner.Type = "User"
	user := api.Repository{}
	user.Owner.Login = "alice"
	user.Owner.Type = "User"
	user.Namespace.Path = "alice"

	if kind, err := classifyRepositoryOwner(organization, "team"); err != nil || kind != repositoryOwnerOrganization {
		t.Fatalf("organization kind = %v, error = %v", kind, err)
	}
	if kind, err := classifyRepositoryOwner(user, "alice"); err != nil || kind != repositoryOwnerUser {
		t.Fatalf("user kind = %v, error = %v", kind, err)
	}
	if _, err := classifyRepositoryOwner(api.Repository{}, "unknown"); err == nil {
		t.Fatal("ambiguous owner was accepted")
	}
}
