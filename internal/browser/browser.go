package browser

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

const host = "atomgit.com"

type Opener func(rawURL string) error

func newURLOpenerCmd(rawURL string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL), nil
	case "linux":
		return exec.Command("xdg-open", rawURL), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL), nil
	default:
		return nil, fmt.Errorf("unsupported GOOS: %s", runtime.GOOS)
	}
}

// NewSyncOpener returns a function that asks the operating system to open a URL.
//
// The returned opener is synchronous. It blocks until the operating
// system's URL opener process exits and returns an error if that process
// exits with a non-zero status.
func NewSyncOpener() Opener {
	return func(rawURL string) error {
		cmd, err := newURLOpenerCmd(rawURL)
		if err != nil {
			return err
		}
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return fmt.Errorf("browser exited with error: %w", exitErr)
			}
			return fmt.Errorf("browser opener: %w", err)
		}
		return nil
	}
}

// NewAsyncOpener returns a function that asks the operating system to open a URL.
//
// The returned opener is asynchronous. It only reports failures that prevent
// the operating system's URL opener process from being started and does not
// wait for that process to exit.
func NewAsyncOpener() Opener {
	return func(rawURL string) error {
		cmd, err := newURLOpenerCmd(rawURL)
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("browser opener: %w", err)
		}

		// Reap the child process without blocking the caller.
		go func() {
			_ = cmd.Wait()
		}()
		return nil
	}
}

// Deprecated: NewOpener creates an asynchronous opener.
// Use NewAsyncOpener or NewSyncOpener explicitly.
func NewOpener() Opener {
	return NewAsyncOpener()
}

func BuildRepoURL(owner, repo string) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s", owner, repo),
	}
	return u.String()
}

// buildPageURL is a shared helper for page-level URLs that follow the
// pattern https://atomgit.com/{owner}/{repo}/{suffix}.
func buildPageURL(owner, repo, suffix string) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/%s", owner, repo, suffix),
	}
	return u.String()
}

func BuildFileURL(owner, repo, branch, filePath string, lineStart, lineEnd int) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/blob/%s/%s", owner, repo, branch, filePath),
	}
	if lineStart > 0 {
		if lineEnd > 0 && lineEnd > lineStart {
			u.Fragment = fmt.Sprintf("L%d-L%d", lineStart, lineEnd)
		} else {
			u.Fragment = fmt.Sprintf("L%d", lineStart)
		}
	}
	return u.String()
}

func BuildIssueURL(owner, repo string, number int) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/issues/%d", owner, repo, number),
	}
	return u.String()
}

func BuildPRURL(owner, repo string, number int) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/pull/%d", owner, repo, number),
	}
	return u.String()
}

func BuildCommitURL(owner, repo, sha string) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/commit/%s", owner, repo, sha),
	}
	return u.String()
}

func BuildReleasesURL(owner, repo string) string {
	return buildPageURL(owner, repo, "releases")
}

func BuildActionsURL(owner, repo string) string {
	return buildPageURL(owner, repo, "actions")
}

func BuildWikiURL(owner, repo string) string {
	return buildPageURL(owner, repo, "wiki")
}

func BuildSettingsURL(owner, repo string) string {
	return buildPageURL(owner, repo, "setting")
}
