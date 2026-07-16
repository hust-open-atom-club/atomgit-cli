package browser

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"slices"
	"strings"
)

const host = "atomgit.com"

var validHosts = []string{
	"atomgit.com",
	"gitcode.com",
}

var (
	ErrUnknownHost    = errors.New("unknown host")
	ErrParseRemoteURL = errors.New("failed to parse url")
)

type Opener func(rawURL string) error

func NewOpener() Opener {
	return func(url string) error {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			return fmt.Errorf("unsupported GOOS: %s", runtime.GOOS)
		}
		return cmd.Run()
	}
}

func BuildRepoURL(owner, repo string) string {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s", owner, repo),
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
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/%s/%s/releases", owner, repo),
	}
	return u.String()
}

func ParseRemoteURL(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "ssh://")
	s = strings.TrimPrefix(s, "git@")
	s = strings.Replace(s, ":", "/", 1)

	host, rest, ok := strings.Cut(s, "/")
	if !ok || !slices.Contains(validHosts, host) {
		return "", "", fmt.Errorf("cannot resolve repo %s: %w", raw, ErrUnknownHost)
	}

	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot resolve repo %s: %w", raw, ErrParseRemoteURL)
	}
	return parts[0], parts[1], nil
}
