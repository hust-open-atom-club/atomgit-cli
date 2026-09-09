package auth

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestAuthSetupGitConfiguresHostScopedHelper(t *testing.T) {
	var calls [][]string
	factory := &cmdutil.Factory{
		Config: testConfig{token: "secret", user: "alice"},
		GitConfig: func(args ...string) (string, error) {
			calls = append(calls, append([]string(nil), args...))
			return "", nil
		},
	}
	cmd := newCmdAuthSetupGitWithExecutable(factory, func() (string, error) {
		return "/Applications/AtomGit CLI/ag", nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	wantCalls := [][]string{
		{"config", "--global", "--replace-all", "credential.https://atomgit.com.helper", ""},
		{"config", "--global", "--add", "credential.https://atomgit.com.helper", "!'/Applications/AtomGit CLI/ag' auth git-credential"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("git config calls = %#v, want %#v", calls, wantCalls)
	}
	if got := out.String(); got != "Configured Git to use ag as a credential helper for atomgit.com.\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestAuthSetupGitRequiresAuthentication(t *testing.T) {
	called := false
	factory := &cmdutil.Factory{
		Config: testConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")},
		GitConfig: func(args ...string) (string, error) {
			called = true
			return "", nil
		},
	}
	cmd := newCmdAuthSetupGitWithExecutable(factory, func() (string, error) {
		return "/usr/local/bin/ag", nil
	})

	err := cmd.RunE(cmd, nil)
	if err == nil || err.Error() != "not authenticated: run `ag auth login`" {
		t.Fatalf("error = %v", err)
	}
	if called {
		t.Fatal("git config must not run without an authenticated account")
	}
}

func TestAuthSetupGitStopsAfterConfigurationFailure(t *testing.T) {
	var calls [][]string
	factory := &cmdutil.Factory{
		Config: testConfig{token: "secret", user: "alice"},
		GitConfig: func(args ...string) (string, error) {
			calls = append(calls, append([]string(nil), args...))
			return "", errors.New("permission denied")
		},
	}
	cmd := newCmdAuthSetupGitWithExecutable(factory, func() (string, error) {
		return "/usr/local/bin/ag", nil
	})

	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "reset Git credential helper") {
		t.Fatalf("error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("git config calls = %d, want 1", len(calls))
	}
}

func TestAuthSetupGitWritesExpectedGlobalConfig(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", configFile)
	factory := &cmdutil.Factory{Config: testConfig{token: "secret", user: "alice"}}
	cmd := newCmdAuthSetupGitWithExecutable(factory, func() (string, error) {
		return "/opt/AtomGit CLI/ag", nil
	})
	cmd.SetOut(&bytes.Buffer{})

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("second setup failed: %v", err)
	}
	output, err := exec.Command("git", "config", "--global", "--get-all", "credential.https://atomgit.com.helper").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(output), "\n!'/opt/AtomGit CLI/ag' auth git-credential\n"; got != want {
		t.Fatalf("helper config = %q, want %q", got, want)
	}
}

func TestGitCredentialHelperGet(t *testing.T) {
	tests := []struct {
		name       string
		request    string
		config     testConfig
		wantOutput string
	}{
		{
			name:       "fields",
			request:    "protocol=https\nhost=atomgit.com\n\n",
			config:     testConfig{token: "secret", user: "alice"},
			wantOutput: "protocol=https\nhost=atomgit.com\nusername=alice\npassword=secret\n",
		},
		{
			name:       "URL and matching username",
			request:    "url=https://Alice@atomgit.com/owner/repo.git\n\n",
			config:     testConfig{token: "secret", user: "alice"},
			wantOutput: "protocol=https\nhost=atomgit.com\nusername=alice\npassword=secret\n",
		},
		{
			name:       "URL with explicit HTTPS default port",
			request:    "url=https://Alice@atomgit.com:443/owner/repo.git\n\n",
			config:     testConfig{token: "secret", user: "alice"},
			wantOutput: "protocol=https\nhost=atomgit.com:443\nusername=alice\npassword=secret\n",
		},
		{
			name:       "fields with explicit HTTPS default port",
			request:    "protocol=https\nhost=atomgit.com:443\n\n",
			config:     testConfig{token: "secret", user: "alice"},
			wantOutput: "protocol=https\nhost=atomgit.com:443\nusername=alice\npassword=secret\n",
		},
		{name: "other protocol", request: "protocol=http\nhost=atomgit.com\n\n", config: testConfig{token: "secret", user: "alice"}},
		{name: "other host", request: "protocol=https\nhost=example.com\n\n", config: testConfig{token: "secret", user: "alice"}},
		{name: "other port", request: "protocol=https\nhost=atomgit.com:8443\n\n", config: testConfig{token: "secret", user: "alice"}},
		{name: "lookalike host with default port", request: "protocol=https\nhost=notatomgit.com:443\n\n", config: testConfig{token: "secret", user: "alice"}},
		{name: "other username", request: "protocol=https\nhost=atomgit.com\nusername=bob\n\n", config: testConfig{token: "secret", user: "alice"}},
		{name: "not authenticated", request: "protocol=https\nhost=atomgit.com\n\n", config: testConfig{tokenErr: errors.New("not authenticated")}},
		{name: "unsafe token", request: "protocol=https\nhost=atomgit.com\n\n", config: testConfig{token: "secret\ninjected=value", user: "alice"}},
		{name: "unsafe user", request: "protocol=https\nhost=atomgit.com\n\n", config: testConfig{token: "secret", user: "alice\ninjected=value"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			err := runGitCredentialHelper(&cmdutil.Factory{Config: test.config}, "get", strings.NewReader(test.request), &out)
			if err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != test.wantOutput {
				t.Fatalf("output = %q, want %q", got, test.wantOutput)
			}
		})
	}
}

func TestGitCredentialHelperOperations(t *testing.T) {
	factory := &cmdutil.Factory{Config: testConfig{token: "secret", user: "alice"}}
	for _, operation := range []string{"store", "erase"} {
		t.Run(operation, func(t *testing.T) {
			var out bytes.Buffer
			if err := runGitCredentialHelper(factory, operation, strings.NewReader("protocol=https\nhost=atomgit.com\n"), &out); err != nil {
				t.Fatal(err)
			}
			if out.Len() != 0 {
				t.Fatalf("output = %q", out.String())
			}
		})
	}

	err := runGitCredentialHelper(factory, "unsupported", strings.NewReader(""), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
		t.Fatalf("error = %v", err)
	}
}

func TestGitCredentialHelperDoesNotExposeMalformedURLCredentials(t *testing.T) {
	const suppliedSecret = "credential-from-git"
	request := "url=https://alice:" + suppliedSecret + "@atomgit.com/%zz\n\n"
	var out bytes.Buffer
	err := runGitCredentialHelper(
		&cmdutil.Factory{Config: testConfig{token: "stored-secret", user: "alice"}},
		"get",
		strings.NewReader(request),
		&out,
	)
	if err == nil || !strings.Contains(err.Error(), "invalid URL field") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), suppliedSecret) || out.Len() != 0 {
		t.Fatalf("credential leaked: error=%q output=%q", err, out.String())
	}
}

func TestShellQuote(t *testing.T) {
	tests := map[string]string{
		"/usr/local/bin/ag":         "'/usr/local/bin/ag'",
		"/opt/AtomGit CLI/ag":       "'/opt/AtomGit CLI/ag'",
		"/tmp/AtomGit's CLI/ag":     "'/tmp/AtomGit'\\''s CLI/ag'",
		"/tmp/ag;touch/tmp/pwned":   "'/tmp/ag;touch/tmp/pwned'",
		"/tmp/ag&touch/tmp/pwned":   "'/tmp/ag&touch/tmp/pwned'",
		"/tmp/ag|touch/tmp/pwned":   "'/tmp/ag|touch/tmp/pwned'",
		"/tmp/ag>(touch/tmp/pwned)": "'/tmp/ag>(touch/tmp/pwned)'",
		`C:\\Program Files\\ag.exe`: `'C:\\Program Files\\ag.exe'`,
	}
	for input, want := range tests {
		if got := shellQuote(input); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", input, got, want)
		}
	}
}
