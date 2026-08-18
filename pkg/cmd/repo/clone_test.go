package repo

import (
	"bytes"
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
	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", opts, command); err != nil {
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
	if err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "git@example.invalid:owner/repo.git", opts, command); err != nil {
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

	err := runCloneWithCommand(strings.NewReader(""), io.Discard, io.Discard, "https://atomgit.com/owner/repo.git", &CloneOptions{}, command)
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
	err := runCloneWithCommand(strings.NewReader(""), safeOut, safeErr, "https://example.invalid/repo.git", &CloneOptions{Directory: "repo"}, command)
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
