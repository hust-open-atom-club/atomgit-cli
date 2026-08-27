package repo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepoContentCommandRegistration(t *testing.T) {
	content := newCmdRepoContent(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))

	for _, name := range []string{"list", "view"} {
		child, _, err := content.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("content %s subcommand: %v", name, err)
		}
		for _, flag := range []string{"json", "ref"} {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("content %s --%s flag was not registered", name, flag)
			}
		}
	}

	list, _, _ := content.Find([]string{"list"})
	if err := list.Args(list, []string{"owner/repo", "path", "extra"}); err == nil {
		t.Fatal("content list accepted too many arguments")
	}
	view, _, _ := content.Find([]string{"view"})
	if err := view.Args(view, nil); err == nil {
		t.Fatal("content view accepted no arguments")
	}
	if err := view.Args(view, []string{"owner/repo", "path", "extra"}); err == nil {
		t.Fatal("content view accepted too many arguments")
	}
}

func TestRepoContentListInferredRootAndRef(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/api/v5/repos/team/inferred/contents" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.URL.Query().Get("ref") != "feature/x" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return forkResponse(http.StatusOK, `[{"name":"docs","path":"docs","sha":"abc123","type":"dir"}]`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoContentList(factory)
	if err := cmd.Flags().Set("ref", "feature/x"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "dir\tdocs\tabc123\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRepoContentListSingleArgumentIsInferredRepositoryPath(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/v5/repos/team/inferred/contents/docs/guides%20and%20examples" {
			t.Fatalf("escaped path = %q", req.URL.EscapedPath())
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoContentList(factory)
	if err := cmd.RunE(cmd, []string{"docs/guides and examples"}); err != nil {
		t.Fatal(err)
	}
}

func TestRepoContentListExplicitRepositoryTakesPrecedence(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/other/project/contents" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		t.Fatal("repository resolver was called for an explicit repository")
		return cmdutil.Repository{}, nil
	}
	cmd := newCmdRepoContentList(factory)
	if err := cmd.RunE(cmd, []string{"other/project", "."}); err != nil {
		t.Fatal(err)
	}
}

func TestRepoContentListJSONPreservesAPIMetadata(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[{"name":"docs","path":"docs","sha":"abc","type":"dir","url":"https://example.test/docs","_links":{"self":"https://example.test/self"},"future_field":"kept"}]`), nil
	})

	cmd := newCmdRepoContentList(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}

	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %#v", entries)
	}
	for _, key := range []string{"url", "_links", "future_field", "sha"} {
		if _, ok := entries[0][key]; !ok {
			t.Errorf("JSON output lost %q", key)
		}
	}
}

func TestRepoContentListRejectsFile(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"README.md","path":"README.md","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`), nil
	})

	cmd := newCmdRepoContentList(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.Flags().Set("ref", "feature/x"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo", "README.md"})
	if err == nil {
		t.Fatal("expected file type mismatch")
	}
	for _, want := range []string{"is a file", "content view", "alice/demo", "README.md", "feature/x"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want %q", err, want)
		}
	}
	if !strings.Contains(err.Error(), `--ref "feature/x"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoContentListEmptyDirectory(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[]`), nil
	})

	for _, jsonOutput := range []bool{false, true} {
		cmd := newCmdRepoContentList(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
		if jsonOutput {
			if err := cmd.Flags().Set("json", "true"); err != nil {
				t.Fatal(err)
			}
		}
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
			t.Fatal(err)
		}
		want := ""
		if jsonOutput {
			want = "[]\n"
		}
		if out.String() != want {
			t.Fatalf("json=%v output = %q, want %q", jsonOutput, out.String(), want)
		}
	}
}

func TestRepoContentViewDecodedBytesAndExplicitRepository(t *testing.T) {
	decoded := []byte{'a', 0, 'b', '\n'}
	encoded := base64.StdEncoding.EncodeToString(decoded)
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/other/project/contents/bin/data" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{"name":"data","path":"bin/data","sha":"abc","size":4,"type":"file","encoding":"base64","content":"`+encoded+`"}`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		t.Fatal("repository resolver was called for an explicit repository")
		return cmdutil.Repository{}, nil
	}
	cmd := newCmdRepoContentView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"other/project", "bin/data"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), decoded) {
		t.Fatalf("output = %v, want %v", out.Bytes(), decoded)
	}
}

func TestRepoContentViewJSONPreservesEncodedAPIObject(t *testing.T) {
	body := `{"name":"README.md","path":"README.md","sha":"abc","size":4,"type":"file","encoding":"base64","content":"dGVzdA==","download_url":"https://example.test/raw","_links":{"html":"https://example.test/html"},"future_field":42}`
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, body), nil
	})

	cmd := newCmdRepoContentView(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "README.md"}); err != nil {
		t.Fatal(err)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &object); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"content", "download_url", "_links", "future_field"} {
		if _, ok := object[key]; !ok {
			t.Errorf("JSON output lost %q", key)
		}
	}
	if string(object["content"]) != `"dGVzdA=="` {
		t.Fatalf("content = %s", object["content"])
	}
}

func TestRepoContentViewRejectsDirectory(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[{"name":"main.go","path":"src/main.go","sha":"abc","type":"file"}]`), nil
	})

	cmd := newCmdRepoContentView(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.Flags().Set("ref", "release/v1"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo", "src"})
	if err == nil {
		t.Fatal("expected directory type mismatch")
	}
	for _, want := range []string{"is a directory", "content list", "alice/demo", "src", "release/v1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want %q", err, want)
		}
	}
	if !strings.Contains(err.Error(), `--ref "release/v1"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoContentViewRejectsMalformedBase64(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"bad","path":"bad","sha":"abc","size":3,"type":"file","encoding":"base64","content":"!!!"}`), nil
	})

	cmd := newCmdRepoContentView(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "bad"})
	if err == nil || !strings.Contains(err.Error(), "decode Base64 content") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoContentViewReportsWriterError(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"f","path":"f","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`), nil
	})
	wantErr := errors.New("write failed")
	cmd := newCmdRepoContentView(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	cmd.SetOut(failingContentWriter{err: wantErr})
	err := cmd.RunE(cmd, []string{"alice/demo", "f"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestRepoContentValidationBeforeAuthentication(t *testing.T) {
	factory := repoFactory(repoCommandConfig{tokenErr: errors.New("missing token"), user: "alice"}, nil)

	view := newCmdRepoContentView(factory)
	if err := view.RunE(view, []string{"alice/demo", "../secret"}); err == nil || !strings.Contains(err.Error(), "must not contain") {
		t.Fatalf("view error = %v", err)
	}

	list := newCmdRepoContentList(factory)
	if err := list.Flags().Set("ref", " "); err != nil {
		t.Fatal(err)
	}
	if err := list.RunE(list, []string{"alice/demo", "."}); err == nil || !strings.Contains(err.Error(), "--ref cannot be empty") {
		t.Fatalf("list error = %v", err)
	}
}
