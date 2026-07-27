package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

// TestGitHelper is a sentinel test function used by the mock command.
// When AG_GIT_HELPER=1, it writes configured output and exits with a configured code.
// Otherwise it is a no-op (so normal test runs are unaffected).
func TestGitHelper(t *testing.T) {
	if os.Getenv("AG_GIT_HELPER") != "1" {
		return
	}
	exitCode, _ := strconv.Atoi(os.Getenv("AG_GIT_EXIT"))
	stdout := os.Getenv("AG_GIT_STDOUT")
	stderr := os.Getenv("AG_GIT_STDERR")

	if stdout != "" {
		os.Stdout.WriteString(stdout)
	}
	if stderr != "" {
		os.Stderr.WriteString(stderr)
	}
	os.Exit(exitCode)
}

type call struct {
	Name string
	Args []string
}

// recorder records invocations and controls mock command behavior.
type recorder struct {
	mu              sync.Mutex
	calls           []call
	exitCodes       map[string]int    // key: git subcommand name, e.g. "status", "fetch"
	outputs         map[string]string // key: git subcommand name, stdout content
	stderrs         map[string]string // key: git subcommand name, stderr content
	returnExecError bool              // when true, mock returns a command that always fails with exec error
}

func (r *recorder) setupCommand(cmdName string, exitCode int, stdout, stderr string) {
	if r.exitCodes == nil {
		r.exitCodes = make(map[string]int)
	}
	if r.outputs == nil {
		r.outputs = make(map[string]string)
	}
	if r.stderrs == nil {
		r.stderrs = make(map[string]string)
	}
	r.exitCodes[cmdName] = exitCode
	r.outputs[cmdName] = stdout
	r.stderrs[cmdName] = stderr
}

func (r *recorder) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	r.mu.Lock()
	r.calls = append(r.calls, call{Name: name, Args: args})

	var exitCode int
	var stdout, stderr string

	if r.returnExecError {
		r.mu.Unlock()
		return exec.CommandContext(ctx, "nonexistent-binary-that-does-not-exist")
	}

	if len(args) > 0 {
		exitCode = r.exitCodes[args[0]]
		stdout = r.outputs[args[0]]
		stderr = r.stderrs[args[0]]
	}
	r.mu.Unlock()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestGitHelper")
	cmd.Env = append(os.Environ(),
		"AG_GIT_HELPER=1",
		fmt.Sprintf("AG_GIT_EXIT=%d", exitCode),
		fmt.Sprintf("AG_GIT_STDOUT=%s", stdout),
		fmt.Sprintf("AG_GIT_STDERR=%s", stderr),
	)
	return cmd
}

func newTestClient() (*Client, *recorder) {
	r := &recorder{}
	c := &Client{commandContext: r.command}
	return c, r
}

func TestClientRunSuccess(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("status", 0, "", "")

	ctx := context.Background()
	err := c.Run(ctx, "status")
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
}

func TestClientRunNonZeroExit(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("fetch", 128, "", "fatal: not a git repository")

	ctx := context.Background()
	err := c.Run(ctx, "fetch")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	var gitErr *Err
	if !errors.As(err, &gitErr) {
		t.Fatalf("expected *Err, got %T", err)
	}
	if gitErr.ExitCode != 128 {
		t.Fatalf("ExitCode = %d, want 128", gitErr.ExitCode)
	}
	if gitErr.Stderr != "fatal: not a git repository" {
		t.Fatalf("Stderr = %q, want %q", gitErr.Stderr, "fatal: not a git repository")
	}
}

func TestClientRunExecError(t *testing.T) {
	c, r := newTestClient()
	r.returnExecError = true

	ctx := context.Background()
	err := c.Run(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	var gitErr *Err
	if errors.As(err, &gitErr) {
		t.Fatal("expected non-*Err error, got *Err")
	}
}

func TestClientOutput(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("rev-parse", 0, "abc123\n", "")

	ctx := context.Background()
	out, err := c.Output(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		t.Fatalf("Output() returned error: %v", err)
	}
	if out != "abc123" {
		t.Fatalf("Output() = %q, want %q", out, "abc123")
	}
}

func TestClientOutputNonZeroExit(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("status", 1, "", "fatal: error")

	ctx := context.Background()
	out, err := c.Output(ctx, "status")
	if out != "" {
		t.Fatalf("Output() = %q, want empty string", out)
	}
	if err == nil {
		t.Fatal("Output() expected error, got nil")
	}

	var gitErr *Err
	if !errors.As(err, &gitErr) {
		t.Fatalf("expected *Err, got %T", err)
	}
	if gitErr.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", gitErr.ExitCode)
	}
}

func TestClientDirInjection(t *testing.T) {
	ctx := context.Background()
	c := &Client{
		Dir:            "/tmp/repo",
		commandContext: exec.CommandContext,
	}
	cmd := c.Command(ctx, "status")
	if cmd.Dir != "/tmp/repo" {
		t.Fatalf("cmd.Dir = %q, want %q", cmd.Dir, "/tmp/repo")
	}
}

func TestFetch(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("fetch", 0, "", "")

	ctx := context.Background()
	err := c.Fetch(ctx, "origin", "+refs/heads/x:refs/remotes/origin/x")
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}

	want := []call{{Name: "git", Args: []string{"fetch", "origin", "+refs/heads/x:refs/remotes/origin/x"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestCheckout(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("checkout", 0, "", "")

	ctx := context.Background()
	err := c.Checkout(ctx, "feature", "origin/feature", false)
	if err != nil {
		t.Fatalf("Checkout() returned error: %v", err)
	}

	want := []call{{Name: "git", Args: []string{"checkout", "-b", "feature", "--track", "origin/feature"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestCheckoutForce(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("checkout", 0, "", "")

	ctx := context.Background()
	err := c.Checkout(ctx, "feature", "", true)
	if err != nil {
		t.Fatalf("Checkout() returned error: %v", err)
	}

	want := []call{{Name: "git", Args: []string{"checkout", "--force", "-B", "feature"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestRemoteAdd(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("remote", 0, "", "")

	ctx := context.Background()
	err := c.RemoteAdd(ctx, "upstream", "https://atomgit.com/owner/repo.git")
	if err != nil {
		t.Fatalf("RemoteAdd() returned error: %v", err)
	}

	want := []call{{Name: "git", Args: []string{"remote", "add", "upstream", "https://atomgit.com/owner/repo.git"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestRemoteRemove(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("remote", 0, "", "")

	ctx := context.Background()
	err := c.RemoteRemove(ctx, "upstream")
	if err != nil {
		t.Fatalf("RemoteRemove() returned error: %v", err)
	}

	want := []call{{Name: "git", Args: []string{"remote", "remove", "upstream"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestRemoteGetURL(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("remote", 0, "https://atomgit.com/owner/repo.git\n", "")

	ctx := context.Background()
	url, err := c.RemoteGetURL(ctx, "origin")
	if err != nil {
		t.Fatalf("RemoteGetURL() returned error: %v", err)
	}
	if url != "https://atomgit.com/owner/repo.git" {
		t.Fatalf("RemoteGetURL() = %q, want %q", url, "https://atomgit.com/owner/repo.git")
	}

	want := []call{{Name: "git", Args: []string{"remote", "get-url", "origin"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestRevParse(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("rev-parse", 0, "abc123def\n", "")

	ctx := context.Background()
	sha, err := c.RevParse(ctx, "HEAD")
	if err != nil {
		t.Fatalf("RevParse() returned error: %v", err)
	}
	if sha != "abc123def" {
		t.Fatalf("RevParse() = %q, want %q", sha, "abc123def")
	}

	want := []call{{Name: "git", Args: []string{"rev-parse", "--verify", "HEAD"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestStatusPorcelain(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("status", 0, " M file.go\n?? new.txt\n", "")

	ctx := context.Background()
	out, err := c.StatusPorcelain(ctx)
	if err != nil {
		t.Fatalf("StatusPorcelain() returned error: %v", err)
	}
	if out != " M file.go\n?? new.txt" {
		t.Fatalf("StatusPorcelain() = %q, want %q", out, " M file.go\n?? new.txt")
	}

	want := []call{{Name: "git", Args: []string{"status", "--porcelain"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestHasLocalBranch(t *testing.T) {
	t.Run("branch exists", func(t *testing.T) {
		c, r := newTestClient()
		r.setupCommand("rev-parse", 0, "abc123\n", "")

		ctx := context.Background()
		exists := c.HasLocalBranch(ctx, "main")
		if !exists {
			t.Fatal("HasLocalBranch() = false, want true")
		}
	})

	t.Run("branch does not exist", func(t *testing.T) {
		c, r := newTestClient()
		r.setupCommand("rev-parse", 128, "", "fatal: Not a valid object name")

		ctx := context.Background()
		exists := c.HasLocalBranch(ctx, "nonexistent")
		if exists {
			t.Fatal("HasLocalBranch() = true, want false")
		}
	})
}

func TestConfig(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("config", 0, "some-user\n", "")

	ctx := context.Background()
	val, err := c.Config(ctx, "user.name")
	if err != nil {
		t.Fatalf("Config() returned error: %v", err)
	}
	if val != "some-user" {
		t.Fatalf("Config() = %q, want %q", val, "some-user")
	}

	want := []call{{Name: "git", Args: []string{"config", "user.name"}}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls = %#v, want %#v", r.calls, want)
	}
}

func TestParseRemotes(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantURLs map[string]string
	}{
		{
			name:     "empty",
			output:   "",
			wantURLs: map[string]string{},
		},
		{
			name: "single remote",
			output: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n" +
				"origin\thttps://atomgit.com/owner/repo.git (push)",
			wantURLs: map[string]string{"origin": "https://atomgit.com/owner/repo.git"},
		},
		{
			name: "multiple remotes",
			output: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n" +
				"origin\thttps://atomgit.com/owner/repo.git (push)\n" +
				"upstream\thttps://atomgit.com/upstream/repo.git (fetch)\n" +
				"upstream\thttps://atomgit.com/upstream/repo.git (push)",
			wantURLs: map[string]string{
				"origin":   "https://atomgit.com/owner/repo.git",
				"upstream": "https://atomgit.com/upstream/repo.git",
			},
		},
		{
			name:     "push only",
			output:   "origin\thttps://atomgit.com/owner/repo.git (push)",
			wantURLs: map[string]string{"origin": "https://atomgit.com/owner/repo.git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remotes := parseRemotes(tt.output)
			if len(remotes) != len(tt.wantURLs) {
				t.Fatalf("len(remotes) = %d, want %d", len(remotes), len(tt.wantURLs))
			}
			for _, r := range remotes {
				wantURL, ok := tt.wantURLs[r.Name]
				if !ok {
					t.Fatalf("unexpected remote %s", r.Name)
				}
				if r.URL != wantURL {
					t.Fatalf("remote[%s].URL = %q, want %q", r.Name, r.URL, wantURL)
				}
			}
		})
	}
}

func TestRemotes(t *testing.T) {
	c, r := newTestClient()
	r.setupCommand("remote", 0, "origin\thttps://atomgit.com/a/b.git (fetch)\norigin\thttps://atomgit.com/a/b.git (push)", "")

	remotes, err := c.Remotes(context.Background())
	if err != nil {
		t.Fatalf("Remotes() error = %v", err)
	}
	if len(remotes) != 1 {
		t.Fatalf("len(remotes) = %d, want 1", len(remotes))
	}
	if remotes[0].Name != "origin" || remotes[0].URL != "https://atomgit.com/a/b.git" {
		t.Fatalf("remotes[0] = %+v", remotes[0])
	}
}
