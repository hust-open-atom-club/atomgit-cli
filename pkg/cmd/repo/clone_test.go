package repo

import (
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
		wantURL  string
		wantName string
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
			wantURL:  "https://atomgit.com/project",
			wantName: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotName := parseRepoArg(tt.arg)
			if gotURL != tt.wantURL || gotName != tt.wantName {
				t.Fatalf("parseRepoArg(%q) = (%q, %q), want (%q, %q)", tt.arg, gotURL, gotName, tt.wantURL, tt.wantName)
			}
		})
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
