// Package git provides a lightweight, testable Client for executing git commands.
//
// The Client wraps os/exec with support for context cancellation, working directory
// injection, stdout/stderr capture, and structured error handling.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Err represents a git command failure with the captured stderr output.
type Err struct {
	ExitCode int
	Stderr   string
	Err      error
}

func (e *Err) Error() string {
	return fmt.Sprintf("git command exited with code %d\nstderr:\n%s", e.ExitCode, e.Stderr)
}

// Unwrap returns the underlying *exec.ExitError.
func (e *Err) Unwrap() error {
	return e.Err
}

// commandCtx is the injection point for tests.
type commandCtx func(ctx context.Context, name string, args ...string) *exec.Cmd

// Client wraps git command execution with testability injection.
type Client struct {
	// Dir is the working directory for the git command. When non-empty,
	// git commands are executed in this directory (via cmd.Dir).
	Dir string

	// Stderr controls where stderr is written. When nil, stderr is captured
	// into a buffer and returned via Err.Stderr on failure.
	Stderr io.Writer

	// Stdout controls where stdout is written. When nil, stdout is captured
	// into a buffer and returned by Output.
	Stdout io.Writer

	commandContext commandCtx
}

// NewTestClient creates a Client using the given command factory function.
// This is primarily used in tests to mock git command execution.
func NewTestClient(fn commandCtx) *Client {
	return &Client{commandContext: fn}
}

// NewClient creates a Client using exec.CommandContext as the command factory.
func NewClient() *Client {
	return &Client{
		commandContext: exec.CommandContext,
	}
}

// Command builds an *exec.Cmd for the given git args.
// If Dir is set, it injects cmd.Dir (rather than a -C flag) so that the git
// process runs in the specified directory.
func (c *Client) Command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := c.commandContext(ctx, "git", args...)
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	return cmd
}

// Run executes a git command and returns a *Err on failure (non-zero exit or
// exec error). Non-git-command errors (context cancelled, binary not found) are
// wrapped but NOT returned as *Err.
func (c *Client) Run(ctx context.Context, args ...string) error {
	cmd := c.Command(ctx, args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if c.Stderr != nil {
		cmd.Stderr = io.MultiWriter(c.Stderr, stderr)
	}
	if c.Stdout != nil {
		cmd.Stdout = c.Stdout
	} else {
		cmd.Stdout = io.Discard
	}

	err := cmd.Run()
	if err != nil {
		return c.wrapErr(err, stderr.String())
	}
	return nil
}

// Output executes a git command and returns stdout as a string.
// On failure it returns an empty string and an error (same semantics as Run).
func (c *Client) Output(ctx context.Context, args ...string) (string, error) {
	cmd := c.Command(ctx, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if c.Stdout != nil {
		cmd.Stdout = io.MultiWriter(c.Stdout, stdout)
	}
	if c.Stderr != nil {
		cmd.Stderr = io.MultiWriter(c.Stderr, stderr)
	}

	err := cmd.Run()
	if err != nil {
		return "", c.wrapErr(err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n\r"), nil
}

// wrapErr converts an exec error into a *Err if it is an *exec.ExitError.
// Other errors (context cancelled, binary not found) are returned as-is.
func (c *Client) wrapErr(err error, stderr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &Err{
			ExitCode: exitErr.ExitCode(),
			Stderr:   stderr,
			Err:      err,
		}
	}
	return err
}

// ---------------------------------------------------------------------------
// Convenience methods
// ---------------------------------------------------------------------------

// Fetch runs: git fetch <remote> <refspec>.
func (c *Client) Fetch(ctx context.Context, remote, refspec string) error {
	return c.Run(ctx, "fetch", remote, refspec)
}

// Checkout runs: git checkout [opts] <branch> [--track <track>].
// When force is true it uses -B instead of -b.
// When track is non-empty it appends --track <track>.
func (c *Client) Checkout(ctx context.Context, branch, track string, force bool) error {
	args := []string{"checkout"}
	if force {
		args = append(args, "--force")
		args = append(args, "-B", branch)
	} else {
		args = append(args, "-b", branch)
	}
	if track != "" {
		args = append(args, "--track", track)
	}
	return c.Run(ctx, args...)
}

// RemoteAdd runs: git remote add <name> <url>.
func (c *Client) RemoteAdd(ctx context.Context, name, url string) error {
	return c.Run(ctx, "remote", "add", name, url)
}

// RemoteRemove runs: git remote remove <name>.
func (c *Client) RemoteRemove(ctx context.Context, name string) error {
	return c.Run(ctx, "remote", "remove", name)
}

// Remote holds parsed information about a git remote.
type Remote struct {
	Name string // remote name (e.g., "origin", "upstream")
	URL  string // fetch URL
}

// Remotes returns all configured remotes by parsing `git remote -v`.
// Only fetch URLs are returned; duplicates (push entries) are skipped.
func (c *Client) Remotes(ctx context.Context) ([]Remote, error) {
	out, err := c.Output(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	return parseRemotes(out), nil
}

// parseRemotes parses the output of `git remote -v`.
// Format: <name>\t<url> (fetch)
//
//	<name>\t<url> (push)
func parseRemotes(output string) []Remote {
	var remotes []Remote
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		rest := parts[1]
		idx := strings.LastIndex(rest, " (")
		if idx < 0 {
			continue
		}
		url := rest[:idx]
		if !seen[name] {
			seen[name] = true
			remotes = append(remotes, Remote{Name: name, URL: url})
		}
	}
	return remotes
}

// RemoteGetURL runs: git remote get-url <name>. Returns trimmed stdout.
func (c *Client) RemoteGetURL(ctx context.Context, name string) (string, error) {
	return c.Output(ctx, "remote", "get-url", name)
}

// RevParse runs: git rev-parse --verify <ref>. Returns trimmed stdout (the SHA or ref).
func (c *Client) RevParse(ctx context.Context, ref string) (string, error) {
	return c.Output(ctx, "rev-parse", "--verify", ref)
}

// StatusPorcelain runs: git status --porcelain. Returns trimmed stdout.
func (c *Client) StatusPorcelain(ctx context.Context) (string, error) {
	return c.Output(ctx, "status", "--porcelain")
}

// HasLocalBranch checks whether a local branch exists by running
// git rev-parse --verify --quiet refs/heads/<name>.
// The --quiet flag suppresses the "fatal:" message when the branch does not exist.
func (c *Client) HasLocalBranch(ctx context.Context, name string) bool {
	err := c.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// CheckBranchName validates a branch name using git check-ref-format --branch.
func (c *Client) CheckBranchName(ctx context.Context, name string) error {
	return c.Run(ctx, "check-ref-format", "--branch", name)
}

// Config runs: git config <key>. Returns trimmed stdout.
func (c *Client) Config(ctx context.Context, key string) (string, error) {
	return c.Output(ctx, "config", key)
}
