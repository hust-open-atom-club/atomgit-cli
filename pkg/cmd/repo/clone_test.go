package repo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		owner    string
		wantURL  string
		wantName string
		wantErr  bool
	}{
		{
			name:     "HTTPS URL",
			arg:      "https://atomgit.com/owner/project.git",
			wantURL:  "https://atomgit.com/owner/project.git",
			wantName: "project",
		},
		{
			name:     "HTTP URL",
			arg:      "http://atomgit.com/owner/project",
			wantURL:  "http://atomgit.com/owner/project",
			wantName: "project",
		},
		{
			name:     "SSH URL",
			arg:      "git@atomgit.com:owner/project.git",
			wantURL:  "git@atomgit.com:owner/project.git",
			wantName: "project",
		},
		{
			name:     "owner and repository",
			arg:      "owner/project",
			wantURL:  "https://atomgit.com/owner/project.git",
			wantName: "project",
		},
		{
			name:     "repository only",
			arg:      "project",
			owner:    "alice",
			wantURL:  "https://atomgit.com/alice/project.git",
			wantName: "project",
		},
		{name: "repository without owner", arg: "project", wantErr: true},
		{name: "too many path components", arg: "owner/group/project", wantErr: true},
		{name: "empty owner", arg: "/project", wantErr: true},
		{name: "empty repository", arg: "owner/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotName, err := parseRepoArg(tt.arg, tt.owner)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseRepoArg() expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotURL != tt.wantURL || gotName != tt.wantName {
				t.Fatalf("parseRepoArg(%q) = (%q, %q), want (%q, %q)", tt.arg, gotURL, gotName, tt.wantURL, tt.wantName)
			}
		})
	}
}

func TestResolveCloneRepoArgUsesCurrentUserForShortName(t *testing.T) {
	factory := repoFactory(repoCommandConfig{user: "alice"}, nil)
	cloneURL, repoName, err := resolveCloneRepoArg(factory, "project")
	if err != nil {
		t.Fatal(err)
	}
	if cloneURL != "https://atomgit.com/alice/project.git" || repoName != "project" {
		t.Fatalf("resolveCloneRepoArg() = (%q, %q)", cloneURL, repoName)
	}
}

func TestResolveCloneRepoArgReportsUserError(t *testing.T) {
	factory := repoFactory(repoCommandConfig{userErr: errors.New("missing user")}, nil)
	_, _, err := resolveCloneRepoArg(factory, "project")
	if err == nil || !strings.Contains(err.Error(), "not authenticated: missing user") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunCloneWithCommand(t *testing.T) {
	var gotName string
	var gotArgs []string
	command := func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=success")
		return cmd
	}

	opts := &CloneOptions{Branch: "dev", Directory: "target"}
	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", opts, nil, command); err != nil {
		t.Fatal(err)
	}
	if gotName != "git" {
		t.Fatalf("command = %q", gotName)
	}
	wantArgs := []string{"clone", "--branch", "dev", "--", "https://atomgit.com/owner/repo.git", "target"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestRunCloneWithCommandRejectsGitOptionInjection(t *testing.T) {
	var gotArgs []string
	command := func(_ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=success")
		return cmd
	}

	opts := &CloneOptions{Directory: "--config=core.sshCommand=attacker-command"}
	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "git@example.invalid:owner/repo.git", opts, nil, command); err != nil {
		t.Fatal(err)
	}
	want := []string{"clone", "--", "git@example.invalid:owner/repo.git", opts.Directory}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want option separator before user input: %#v", gotArgs, want)
	}
}

func TestRunCloneWithCommandReportsFailure(t *testing.T) {
	command := func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=failure")
		return cmd
	}

	err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", &CloneOptions{}, nil, command)
	if err == nil || !strings.Contains(err.Error(), "failed to clone repository") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunCloneWithCommandSanitizesGitOutput(t *testing.T) {
	command := func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=malicious")
		return cmd
	}

	var stdout, stderr bytes.Buffer
	safeOut := cmdutil.NewSanitizingWriter(&stdout)
	safeErr := cmdutil.NewSanitizingWriter(&stderr)
	err := runCloneWithCommand(strings.NewReader(""), safeOut, safeErr, "https://example.invalid/repo.git", &CloneOptions{Directory: "repo"}, nil, command)
	if err != nil {
		t.Fatal(err)
	}
	if err := safeOut.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := safeErr.Flush(); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]string{"stdout": stdout.String(), "stderr": stderr.String()} {
		if strings.ContainsRune(got, '\x1b') {
			t.Fatalf("%s contains a raw escape sequence: %q", name, got)
		}
		if !strings.Contains(got, `\x1b]52;c;attack\x07`) {
			t.Fatalf("%s did not contain visible sanitized output: %q", name, got)
		}
	}
}

func TestResolveCloneAuth(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		tokenErr  error
		user      string
		userErr   error
		cloneURL  string
		wantCreds bool
		wantHost  string
	}{
		{
			name:      "HTTPS AtomGit clone authenticates",
			token:     "token",
			user:      "alice",
			cloneURL:  "https://atomgit.com/owner/repo.git",
			wantCreds: true,
			wantHost:  "atomgit.com",
		},
		{
			name:      "full HTTPS URL authenticates",
			token:     "token",
			user:      "alice",
			cloneURL:  "https://atomgit.com/owner/repo",
			wantCreds: true,
			wantHost:  "atomgit.com",
		},
		{
			name:      "HTTPS GitCode mirror clone authenticates",
			token:     "token",
			user:      "alice",
			cloneURL:  "https://gitcode.com/owner/repo.git",
			wantCreds: true,
			wantHost:  "gitcode.com",
		},
		{
			name:      "full HTTPS GitCode mirror URL authenticates",
			token:     "token",
			user:      "alice",
			cloneURL:  "https://gitcode.com/owner/repo",
			wantCreds: true,
			wantHost:  "gitcode.com",
		},
		{name: "SSH clone stays anonymous", token: "token", user: "alice", cloneURL: "git@atomgit.com:owner/repo.git"},
		{name: "foreign host stays anonymous", token: "token", user: "alice", cloneURL: "https://github.com/owner/repo.git"},
		{name: "insecure HTTP stays anonymous", token: "token", user: "alice", cloneURL: "http://atomgit.com/owner/repo.git"},
		{name: "unauthenticated clone stays anonymous", cloneURL: "https://atomgit.com/owner/repo.git"},
		{
			name:     "missing token stays anonymous",
			tokenErr: errors.New("not authenticated: run `ag auth login`"),
			user:     "alice",
			cloneURL: "https://atomgit.com/owner/repo.git",
		},
		{name: "missing user stays anonymous", token: "token", userErr: errors.New("missing user"), cloneURL: "https://atomgit.com/owner/repo.git"},
		{name: "empty token stays anonymous", user: "alice", cloneURL: "https://atomgit.com/owner/repo.git"},
		{name: "empty user stays anonymous", token: "token", cloneURL: "https://atomgit.com/owner/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := repoCommandConfig{token: tt.token, tokenErr: tt.tokenErr, user: tt.user, userErr: tt.userErr}
			creds := resolveCloneAuth(cfg, tt.cloneURL)
			if !tt.wantCreds {
				if creds != nil {
					t.Fatalf("resolveCloneAuth() = %#v, want nil", creds)
				}
				return
			}
			if creds == nil {
				t.Fatal("resolveCloneAuth() = nil, want credentials")
			}
			if creds.host != tt.wantHost || creds.username != tt.user || creds.token != tt.token {
				t.Fatalf("resolveCloneAuth() = %#v", creds)
			}
		})
	}
}

func TestRunCloneWithCommandAttachesAuthToGit(t *testing.T) {
	var captured *exec.Cmd
	command := func(_ string, args ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(),
			"AG_CLONE_HELPER=success",
			"GIT_CONFIG_COUNT=99",
			"GIT_CONFIG_KEY_0=user.overridden",
			"GIT_CONFIG_VALUE_0=stale",
		)
		captured = cmd
		return cmd
	}

	creds := &cloneCredentials{host: "atomgit.com", username: "alice", token: "secret-token"}
	opts := &CloneOptions{Directory: "repo"}
	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", opts, creds, command); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{}
	for _, kv := range captured.Env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	if env["GIT_CONFIG_COUNT"] != "100" {
		t.Fatalf("GIT_CONFIG_COUNT = %q, want inherited count 99 plus the auth entry", env["GIT_CONFIG_COUNT"])
	}
	if env["GIT_CONFIG_KEY_0"] != "user.overridden" || env["GIT_CONFIG_VALUE_0"] != "stale" {
		t.Fatalf("inherited GIT_CONFIG_0 = %q/%q, want preserved", env["GIT_CONFIG_KEY_0"], env["GIT_CONFIG_VALUE_0"])
	}
	if env["GIT_CONFIG_KEY_99"] != "http.https://atomgit.com/.extraheader" {
		t.Fatalf("GIT_CONFIG_KEY_99 = %q", env["GIT_CONFIG_KEY_99"])
	}
	wantValue := "AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret-token"))
	if env["GIT_CONFIG_VALUE_99"] != wantValue {
		t.Fatalf("GIT_CONFIG_VALUE_99 = %q, want %q", env["GIT_CONFIG_VALUE_99"], wantValue)
	}
	if env["GIT_TERMINAL_PROMPT"] != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", env["GIT_TERMINAL_PROMPT"])
	}
	if env["AG_CLONE_HELPER"] != "success" {
		t.Fatalf("AG_CLONE_HELPER = %q, want preserved", env["AG_CLONE_HELPER"])
	}
	// The token must not appear in cleartext anywhere in the child environment.
	for _, kv := range captured.Env {
		if strings.Contains(kv, "secret-token") {
			t.Fatalf("token leaked into the git environment: %q", kv)
		}
	}
}

func TestWithCloneAuthEnvPreservesInheritedGitConfig(t *testing.T) {
	authKey := "http.https://atomgit.com/.extraheader"
	wantValue := "AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte("alice:token"))

	tests := []struct {
		name    string
		env     []string
		wantEnv map[string]string
	}{
		{
			name: "appends auth after unrelated inherited config",
			env: []string{
				"PATH=/usr/bin",
				"GIT_CONFIG_COUNT=2",
				"GIT_CONFIG_KEY_0=http.proxy",
				"GIT_CONFIG_VALUE_0=http://127.0.0.1:8080",
				"GIT_CONFIG_KEY_1=http.sslCAInfo",
				"GIT_CONFIG_VALUE_1=/etc/ssl/certs/ca.pem",
			},
			wantEnv: map[string]string{
				"PATH":                "/usr/bin",
				"GIT_CONFIG_COUNT":    "3",
				"GIT_CONFIG_KEY_0":    "http.proxy",
				"GIT_CONFIG_VALUE_0":  "http://127.0.0.1:8080",
				"GIT_CONFIG_KEY_1":    "http.sslCAInfo",
				"GIT_CONFIG_VALUE_1":  "/etc/ssl/certs/ca.pem",
				"GIT_CONFIG_KEY_2":    authKey,
				"GIT_CONFIG_VALUE_2":  wantValue,
				"GIT_TERMINAL_PROMPT": "0",
			},
		},
		{
			name: "replaces an extraheader for the same URL",
			env: []string{
				"GIT_CONFIG_COUNT=1",
				"GIT_CONFIG_KEY_0=" + authKey,
				"GIT_CONFIG_VALUE_0=AUTHORIZATION: basic c3RhbGU=",
			},
			wantEnv: map[string]string{
				"GIT_CONFIG_COUNT":    "1",
				"GIT_CONFIG_KEY_0":    authKey,
				"GIT_CONFIG_VALUE_0":  wantValue,
				"GIT_TERMINAL_PROMPT": "0",
			},
		},
		{
			name: "starts at index zero without inherited config",
			env:  []string{"HOME=/home/alice"},
			wantEnv: map[string]string{
				"HOME":                "/home/alice",
				"GIT_CONFIG_COUNT":    "1",
				"GIT_CONFIG_KEY_0":    authKey,
				"GIT_CONFIG_VALUE_0":  wantValue,
				"GIT_TERMINAL_PROMPT": "0",
			},
		},
		{
			name: "drops inert keys without GIT_CONFIG_COUNT",
			env: []string{
				"GIT_CONFIG_KEY_0=http.proxy",
				"GIT_CONFIG_VALUE_0=http://127.0.0.1:8080",
			},
			wantEnv: map[string]string{
				"GIT_CONFIG_COUNT":    "1",
				"GIT_CONFIG_KEY_0":    authKey,
				"GIT_CONFIG_VALUE_0":  wantValue,
				"GIT_TERMINAL_PROMPT": "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &cloneCredentials{host: "atomgit.com", username: "alice", token: "token"}
			got := withCloneAuthEnv(tt.env, creds)
			envMap := map[string]string{}
			for _, kv := range got {
				if i := strings.IndexByte(kv, '='); i >= 0 {
					envMap[kv[:i]] = kv[i+1:]
				}
			}
			for key, want := range tt.wantEnv {
				if envMap[key] != want {
					t.Fatalf("%s = %q, want %q (full env: %q)", key, envMap[key], want, got)
				}
			}
			if len(envMap) != len(tt.wantEnv) {
				t.Fatalf("unexpected extra environment entries: %q (full env: %q)", envMap, got)
			}
		})
	}
}

func TestRunCloneWithoutAuthKeepsGitEnvironment(t *testing.T) {
	var captured *exec.Cmd
	var baseEnv []string
	command := func(_ string, _ ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=success")
		captured = cmd
		baseEnv = append([]string(nil), cmd.Env...)
		return cmd
	}

	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", &CloneOptions{}, nil, command); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(captured.Env, baseEnv) {
		t.Fatalf("anonymous clone modified the git environment:\n got: %q\nwant: %q", captured.Env, baseEnv)
	}
}

func TestCloneCommandHelper(t *testing.T) {
	switch os.Getenv("AG_CLONE_HELPER") {
	case "success":
		os.Exit(0)
	case "failure":
		os.Exit(1)
	case "malicious":
		fmt.Fprint(os.Stdout, "remote: \x1b]52;c;attack\x07\n")
		fmt.Fprint(os.Stderr, "warning: \x1b]52;c;attack\x07\n")
		os.Exit(0)
	}
}
