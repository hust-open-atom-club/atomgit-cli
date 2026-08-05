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

type contentTestConfig struct{}

func (contentTestConfig) GetToken() (string, error) { return "test-token", nil }
func (contentTestConfig) GetUser() (string, error)  { return "alice", nil }
func (contentTestConfig) GetHost() string           { return "atomgit.com" }

func contentFactory(config contentTestConfig, transport forkRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func TestReadFilePathValidation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "empty", path: "", wantErr: "path cannot be empty"},
		{name: "leading slash", path: "/etc/passwd", wantErr: "must not start with '/'"},
		{name: "trailing slash", path: "src/", wantErr: "must not end with '/'"},
		{name: "repeated separator", path: "src//file", wantErr: "repeated separators"},
		{name: "dot segment", path: "./file", wantErr: "'.' or '..' segments"},
		{name: "dotdot segment", path: "../file", wantErr: "'.' or '..' segments"},
		{name: "mid dotdot segment", path: "src/../file", wantErr: "'.' or '..' segments"},
		{name: "dot only", path: ".", wantErr: "only valid for directory listing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContentPath(tt.path, false)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateContentPath(%q) = %v, want contains %q", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestReadDirPathValidation(t *testing.T) {
	t.Run("dot is valid", func(t *testing.T) {
		if err := validateContentPath(".", true); err != nil {
			t.Fatalf("expected '.' to be valid, got: %v", err)
		}
	})

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "leading slash", path: "/dir", wantErr: "must not start with '/'"},
		{name: "mid dotdot", path: "dir/../etc", wantErr: "'.' or '..' segments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContentPath(tt.path, true)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateContentPath(%q) = %v, want contains %q", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestReadFileRefValidation(t *testing.T) {
	factory := contentFactory(contentTestConfig{}, nil)
	cmd := newCmdRepoReadFile(factory)
	cmd.SetArgs([]string{"--ref", "", "alice/demo", "README.md"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "ref cannot be empty") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadDirRefValidation(t *testing.T) {
	factory := contentFactory(contentTestConfig{}, nil)
	cmd := newCmdRepoReadDir(factory)
	cmd.SetArgs([]string{"--ref", "", "alice/demo", "."})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "ref cannot be empty") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadFileValidationBeforeAuth(t *testing.T) {
	badCfg := repoCommandConfig{tokenErr: errors.New("missing token"), user: "alice"}
	factory := repoFactory(badCfg, nil)

	cmd := newCmdRepoReadFile(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "/bad/path"})
	if err == nil || !strings.Contains(err.Error(), "must not start with '/'") {
		t.Fatalf("expected path validation before auth, got: %v", err)
	}
}

func TestReadDirValidationBeforeAuth(t *testing.T) {
	badCfg := repoCommandConfig{tokenErr: errors.New("missing token"), user: "alice"}
	factory := repoFactory(badCfg, nil)

	// Path validation must happen before auth
	cmd := newCmdRepoReadDir(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "dir/../evil"})
	if err == nil || !strings.Contains(err.Error(), "'.' or '..'") {
		t.Fatalf("expected path validation before auth, got: %v", err)
	}
}

func TestReadFileOutputText(t *testing.T) {
	decoded := "Hello, World!\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(decoded))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s", req.Method)
		}
		if req.URL.Path != "/api/v5/repos/alice/demo/contents/README.md" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		body := `{"name":"README.md","path":"README.md","sha":"abc","size":14,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "README.md"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != decoded {
		t.Fatalf("output = %q, want %q", out.String(), decoded)
	}
}

func TestReadFileOutputJSON(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("test"))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"name":"f.txt","path":"f.txt","sha":"abc","size":4,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoReadFile(factory)
	_ = cmd.Flags().Set("json", "true")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"}); err != nil {
		t.Fatal(err)
	}

	want := `{
  "name": "f.txt",
  "path": "f.txt",
  "sha": "abc",
  "size": 4,
  "encoding": "base64",
  "content": "` + encoded + `",
  "ref": ""
}`
	assertJSONEqual(t, out.Bytes(), []byte(want))
}

func TestReadFileJSONWithRef(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"name":"data.txt","path":"data.txt","sha":"123","size":4,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoReadFile(factory)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("ref", "feature/x")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "data.txt"}); err != nil {
		t.Fatal(err)
	}

	want := `{
  "name": "data.txt",
  "path": "data.txt",
  "sha": "123",
  "size": 4,
  "encoding": "base64",
  "content": "` + encoded + `",
  "ref": "feature/x"
}`
	assertJSONEqual(t, out.Bytes(), []byte(want))
}

func TestReadFileRejectsDirectory(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"src","path":"src","sha":"abc","size":0,"type":"dir"}`), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "src"})
	if err == nil || !strings.Contains(err.Error(), "is a dir") || !strings.Contains(err.Error(), "not a file") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadFileRejectsUnsupportedEncoding(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"f.txt","path":"f.txt","sha":"abc","size":4,"type":"file","encoding":"utf-8","content":"test"}`), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"})
	if err == nil || !strings.Contains(err.Error(), "unsupported encoding") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadFileRejectsMissingContent(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"f.txt","path":"f.txt","sha":"abc","size":0,"type":"file","encoding":"base64"}`), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"})
	if err == nil || !strings.Contains(err.Error(), "empty file content") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadFileAcceptsZeroByteFile(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"empty.txt","path":"empty.txt","sha":"abc","size":0,"type":"file","encoding":"base64","content":""}`), nil
	})

	t.Run("text", func(t *testing.T) {
		cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "empty.txt"}); err != nil {
			t.Fatal(err)
		}
		if out.Len() != 0 {
			t.Fatalf("output = %q, want empty", out.String())
		}
	})

	t.Run("json", func(t *testing.T) {
		cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
		if err := cmd.Flags().Set("json", "true"); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "empty.txt"}); err != nil {
			t.Fatal(err)
		}
		var result contentFileJSON
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Content != "" || result.Size != 0 {
			t.Fatalf("result = %#v", result)
		}
	})
}

func TestReadFileRejectsMalformedBase64(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `{"name":"f.txt","path":"f.txt","sha":"abc","size":4,"type":"file","encoding":"base64","content":"!!invalid!!"}`), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"})
	if err == nil || !strings.Contains(err.Error(), "malformed base64") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadDirOutputText(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"name":"src","path":"src","sha":"aaa","size":0,"type":"dir"},{"name":"README.md","path":"README.md","sha":"bbb","size":42,"type":"file"}]`
		return forkResponse(http.StatusOK, body), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}

	want := "dir\t0\tsrc\nfile\t42\tREADME.md\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestReadDirOutputJSON(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"name":"cmd","path":"cmd","sha":"def","size":0,"type":"dir"},{"name":"main.go","path":"main.go","sha":"abc","size":100,"type":"file"}]`
		return forkResponse(http.StatusOK, body), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoReadDir(factory)
	_ = cmd.Flags().Set("json", "true")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}

	want := `[
  {
    "name": "cmd",
    "path": "cmd",
    "type": "dir",
    "sha": "def",
    "size": 0
  },
  {
    "name": "main.go",
    "path": "main.go",
    "type": "file",
    "sha": "abc",
    "size": 100
  }
]`
	assertJSONEqual(t, out.Bytes(), []byte(want))
}

func TestReadDirEmptyDirectory(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[]`), nil
	})

	t.Run("text", func(t *testing.T) {
		cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
			t.Fatal(err)
		}
		if out.String() != "" {
			t.Fatalf("expected empty output, got %q", out.String())
		}
	})

	t.Run("json", func(t *testing.T) {
		factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
		cmd := newCmdRepoReadDir(factory)
		_ = cmd.Flags().Set("json", "true")
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, out.Bytes(), []byte(`[]`))
	})
}

func TestReadDirPreservesServerOrder(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[{"name":"z","path":"z","sha":"1","size":0,"type":"file"},{"name":"a","path":"a","sha":"2","size":0,"type":"file"}]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "z") || !strings.HasSuffix(lines[1], "a") {
		t.Fatalf("output = %q, expected z first, then a", out.String())
	}
}

func TestReadFileGetOnlyRequest(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	var capturedMethod string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		body := `{"name":"f.txt","path":"f.txt","sha":"abc","size":4,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"}); err != nil {
		t.Fatal(err)
	}
	if capturedMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", capturedMethod)
	}
}

func TestReadDirGetOnlyRequest(t *testing.T) {
	var capturedMethod string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		return forkResponse(http.StatusOK, `[]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}
	if capturedMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", capturedMethod)
	}
}

func TestReadDirSubdirectoryPath(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/alice/demo/contents/src/pkg" {
			t.Fatalf("path = %q, want /api/v5/repos/alice/demo/contents/src/pkg", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "src/pkg"}); err != nil {
		t.Fatal(err)
	}
}

func TestReadFileInferredRepository(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("inferred"))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/team/inferred/contents/README.md" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		body := `{"name":"README.md","path":"README.md","sha":"abc","size":8,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoReadFile(factory)
	if err := cmd.RunE(cmd, []string{"README.md"}); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirInferredRepository(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/team/inferred/contents" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoReadDir(factory)
	if err := cmd.RunE(cmd, []string{"."}); err != nil {
		t.Fatal(err)
	}
}

func TestReadFileCommandRegistration(t *testing.T) {
	cmd := NewCmdRepo(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))

	child, _, err := cmd.Find([]string{"read-file"})
	if err != nil || child.Name() != "read-file" {
		t.Fatalf("read-file subcommand: %v", err)
	}

	if child.Short != "Read a file from a repository" {
		t.Errorf("read-file Short = %q", child.Short)
	}

	for _, flag := range []string{"ref", "json"} {
		if child.Flags().Lookup(flag) == nil {
			t.Errorf("read-file --%s flag was not registered", flag)
		}
	}

	if err := child.Args(child, nil); err == nil {
		t.Fatal("read-file accepted no arguments")
	}
	if err := child.Args(child, []string{"owner/repo", "path", "extra"}); err == nil {
		t.Fatal("read-file accepted too many arguments")
	}
}

func TestReadDirCommandRegistration(t *testing.T) {
	cmd := NewCmdRepo(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))

	child, _, err := cmd.Find([]string{"read-dir"})
	if err != nil || child.Name() != "read-dir" {
		t.Fatalf("read-dir subcommand: %v", err)
	}

	if child.Short != "List contents of a repository directory" {
		t.Errorf("read-dir Short = %q", child.Short)
	}

	for _, flag := range []string{"ref", "json"} {
		if child.Flags().Lookup(flag) == nil {
			t.Errorf("read-dir --%s flag was not registered", flag)
		}
	}

	if err := child.Args(child, nil); err == nil {
		t.Fatal("read-dir accepted no arguments")
	}
	if err := child.Args(child, []string{"owner/repo", "path", "extra"}); err == nil {
		t.Fatal("read-dir accepted too many arguments")
	}
}

func TestReadDirJSONStableFields(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return forkResponse(http.StatusOK, `[{"name":"f.txt","path":"dir/f.txt","sha":"sha1","size":256,"type":"file"}]`), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoReadDir(factory)
	_ = cmd.Flags().Set("json", "true")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "dir"}); err != nil {
		t.Fatal(err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	entry := result[0]
	for _, key := range []string{"name", "path", "type", "sha", "size"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("JSON entry missing key %q", key)
		}
	}
	if _, ok := entry["encoding"]; ok {
		t.Errorf("JSON entry should not have encoding key for directory entry")
	}
	if _, ok := entry["content"]; ok {
		t.Errorf("JSON entry should not have content key for directory entry")
	}
}

func TestReadFileCommandHelp(t *testing.T) {
	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, want := range []string{"read-file", "path", "--ref", "--json"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not contain %q:\n%s", want, help)
		}
	}
}

func TestReadDirCommandHelp(t *testing.T) {
	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, want := range []string{"read-dir", "path", "--ref", "--json"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not contain %q:\n%s", want, help)
		}
	}
}

func TestReadFileRequestCount(t *testing.T) {
	var count int
	encoded := base64.StdEncoding.EncodeToString([]byte("x"))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		count++
		body := `{"name":"f.txt","path":"f.txt","sha":"abc","size":1,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	cmd := newCmdRepoReadFile(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestReadDirRequestCount(t *testing.T) {
	var count int

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		count++
		return forkResponse(http.StatusOK, `[]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestReadFileRefQueryParameter(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("ref-test"))

	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("ref") != "v2.0" {
			t.Fatalf("ref query = %q, want v2.0", req.URL.RawQuery)
		}
		body := `{"name":"f.txt","path":"f.txt","sha":"abc","size":8,"type":"file","encoding":"base64","content":"` + encoded + `"}`
		return forkResponse(http.StatusOK, body), nil
	})

	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoReadFile(factory)
	_ = cmd.Flags().Set("ref", "v2.0")
	if err := cmd.RunE(cmd, []string{"alice/demo", "f.txt"}); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirRootPathAPI(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/alice/demo/contents" {
			t.Fatalf("path = %q, want /api/v5/repos/alice/demo/contents", req.URL.Path)
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "."}); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirEscapedPath(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/contents/my%20docs" {
			t.Fatalf("escaped path = %q, want /api/v5/repos/alice/demo/contents/my%%20docs", req.URL.EscapedPath())
		}
		return forkResponse(http.StatusOK, `[]`), nil
	})

	cmd := newCmdRepoReadDir(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	if err := cmd.RunE(cmd, []string{"alice/demo", "my docs"}); err != nil {
		t.Fatal(err)
	}
}
