package repo

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepoForkListTextAndPagination(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/team/demo/forks" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("per_page") != "100" || req.URL.Query().Get("page") != "1" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return forkResponse(http.StatusOK, `[{"id":1,"full_name":"alice/demo-copy","url":"https://api.atomgit.com/api/v5/repos/alice/demo-copy","namespace":{"path":"alice","html_url":"https://atomgit.com/alice"},"owner":{"login":"alice"},"parent":{"full_name":"team/demo"},"private":false,"public":true},{"id":2,"full_name":"team/demo-copy-2","url":"https://api.atomgit.com/api/v5/repos/team/demo-copy-2","namespace":{"path":"team","html_url":"https://atomgit.com/team"},"owner":{"login":"team"},"parent":{"full_name":"team/demo"},"private":true,"public":false}]`), nil
	})

	cmd := newCmdRepoForkList(repoFactory(repoCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
	for _, value := range []string{
		"alice/demo-copy [public] owner=alice https://atomgit.com/alice/demo-copy",
		"team/demo-copy-2 [private] owner=team https://atomgit.com/team/demo-copy-2",
	} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("output missing %q:\n%s", value, out.String())
		}
	}
}

func TestRepoForkListJSONAndEmptyResults(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[{"id":1,"full_name":"alice/copy","url":"https://api.atomgit.com/api/v5/repos/alice/copy","namespace":{"path":"alice","html_url":"https://atomgit.com/alice"},"owner":{"login":"alice"},"parent":{"full_name":"team/demo"},"private":false,"public":true}]`), nil
	})
	cmd := newCmdRepoForkList(repoFactory(repoCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"name": "copy"`, `"fullName": "alice/copy"`, `"url": "https://atomgit.com/alice/copy"`, `"fork": true`, `"parent": "team/demo"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("json missing %q: %q", want, out.String())
		}
	}

	empty := newCmdRepoForkList(repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[]`), nil
	})))
	out.Reset()
	empty.SetOut(&out)
	empty.SetArgs([]string{"team/demo", "--json"})
	if err := empty.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Fatalf("empty JSON = %q", out.String())
	}
}

func TestRepoForkListInfersRepository(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/inferred/repo/forks" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[{"full_name":"alice/copy","namespace":{"path":"alice"},"parent":{"full_name":"inferred/repo"}}]`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRepoForkList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoForkListRejectsInvalidLimitBeforeAuthentication(t *testing.T) {
	cfg := &repoRecordingConfig{}
	cmd := newCmdRepoForkList(&cmdutil.Factory{Config: cfg})
	cmd.SetOut(io.Discard)
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"team/demo"})
	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("error = %v", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken calls = %d", cfg.getTokenCalls)
	}
}

func TestRepoForkListReportsAPIError(t *testing.T) {
	cmd := newCmdRepoForkList(repoFactory(repoCommandConfig{token: "token"}, forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusForbidden, `{"message":"private forks denied"}`), nil
	})))
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to list forks for team/demo") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoForkListResolverErrorPrecedesAuthentication(t *testing.T) {
	resolverErr := errors.New("repository context unavailable")
	cfg := &repoRecordingConfig{}
	cfg.repoCommandConfig.tokenErr = errors.New("authentication reached")
	factory := &cmdutil.Factory{Config: cfg, RepositoryResolver: func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, resolverErr
	}}
	cmd := newCmdRepoForkList(factory)
	cmd.SetOut(io.Discard)
	err := cmd.RunE(cmd, nil)
	if !errors.Is(err, resolverErr) {
		t.Fatalf("error = %v", err)
	}
}
