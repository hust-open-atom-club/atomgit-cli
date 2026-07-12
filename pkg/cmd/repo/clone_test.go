package repo

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
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
	if err == nil || !strings.Contains(err.Error(), "failed to get current user: missing user") {
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
	if err := runCloneWithCommand("https://atomgit.com/owner/repo.git", opts, command); err != nil {
		t.Fatal(err)
	}
	if gotName != "git" {
		t.Fatalf("command = %q", gotName)
	}
	wantArgs := []string{"clone", "--branch", "dev", "https://atomgit.com/owner/repo.git", "target"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestRunCloneWithCommandReportsFailure(t *testing.T) {
	command := func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestCloneCommandHelper")
		cmd.Env = append(os.Environ(), "AG_CLONE_HELPER=failure")
		return cmd
	}

	err := runCloneWithCommand("https://atomgit.com/owner/repo.git", &CloneOptions{}, command)
	if err == nil || !strings.Contains(err.Error(), "failed to clone repository") {
		t.Fatalf("error = %v", err)
	}
}

func TestCloneCommandHelper(t *testing.T) {
	switch os.Getenv("AG_CLONE_HELPER") {
	case "success":
		os.Exit(0)
	case "failure":
		os.Exit(1)
	}
}
