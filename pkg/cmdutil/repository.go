package cmdutil

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

var atomgitHosts = []string{"atomgit.com", "gitcode.com"}

// isAtomGitHost reports whether host matches a known AtomGit/GitCode host.
func isAtomGitHost(host string) bool {
	for _, h := range atomgitHosts {
		if strings.EqualFold(host, h) {
			return true
		}
	}
	return false
}

// containsKnownHost reports whether value contains any known host substring.
func containsKnownHost(value string) bool {
	lower := strings.ToLower(value)
	for _, h := range atomgitHosts {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// RepositoryContextHelp describes repository selection for command help.
const RepositoryContextHelp = `When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.`

var errNotAtomGitRemote = errors.New("not an AtomGit/GitCode remote")

// Repository identifies an AtomGit repository.
type Repository struct {
	Owner string
	Name  string
}

func (r Repository) String() string {
	return r.Owner + "/" + r.Name
}

// RepositoryResolver resolves a repository from the current working context.
type RepositoryResolver func() (Repository, error)

// AddRepositoryContextHelp documents repository selection on a command and
// its existing descendants.
func AddRepositoryContextHelp(cmd *cobra.Command) {
	if cmd.Long == "" {
		cmd.Long = cmd.Short
	}
	if !strings.Contains(cmd.Long, RepositoryContextHelp) {
		cmd.Long += "\n\n" + RepositoryContextHelp
	}
	for _, child := range cmd.Commands() {
		AddRepositoryContextHelp(child)
	}
}

// ParseRepository parses an explicit owner/repository value.
func ParseRepository(value string) (Repository, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return Repository{}, fmt.Errorf("invalid repository format: %s (expected owner/repo)", value)
	}

	repository := Repository{
		Owner: strings.TrimSpace(parts[0]),
		Name:  strings.TrimSpace(parts[1]),
	}
	if repository.Owner == "" || repository.Name == "" {
		return Repository{}, fmt.Errorf("invalid repository format: %s (expected owner/repo)", value)
	}
	if InvalidRepositoryPart(repository.Owner) || InvalidRepositoryPart(repository.Name) {
		return Repository{}, fmt.Errorf("invalid repository format: %s (repository names contain an unsafe path character)", value)
	}
	return repository, nil
}

// InvalidRepositoryPart reports whether a single owner/repo path segment
// contains characters that could alter the API request path (? # % backslash
// or slash separators, control characters, or dot segments). Rejecting them
// before the value is interpolated into a URL keeps callers safe from
// request-path injection; a single segment must not contain a "/" separator.
func InvalidRepositoryPart(value string) bool {
	if value == "." || value == ".." || strings.ContainsAny(value, `\\/?#%`) {
		return true
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// ResolveRepository uses an explicit repository when provided and otherwise
// delegates to the repository resolver configured on the factory.
func ResolveRepository(f *Factory, explicit string) (Repository, error) {
	if strings.TrimSpace(explicit) != "" {
		return ParseRepository(explicit)
	}
	if f == nil || f.RepositoryResolver == nil {
		return Repository{}, errors.New("unable to determine repository; pass owner/repo explicitly")
	}

	repository, err := f.RepositoryResolver()
	if err != nil {
		return Repository{}, err
	}
	if repository.Owner == "" || repository.Name == "" {
		return Repository{}, errors.New("repository resolver returned an empty repository")
	}
	return repository, nil
}

// ResolveRepositoryFromArgs treats the first argument as an explicit
// repository only when one more argument than trailingArgs is present.
func ResolveRepositoryFromArgs(f *Factory, args []string, trailingArgs int) (Repository, []string, error) {
	if len(args) < trailingArgs || len(args) > trailingArgs+1 {
		return Repository{}, nil, fmt.Errorf("expected %d or %d arguments, got %d", trailingArgs, trailingArgs+1, len(args))
	}

	remaining := args
	if len(args) == trailingArgs+1 {
		remaining = args[1:]
		repository, err := ParseRepository(args[0])
		if err != nil {
			return Repository{}, nil, err
		}
		return repository, remaining, nil
	}

	repository, err := ResolveRepository(f, "")
	if err != nil {
		return Repository{}, nil, err
	}
	return repository, remaining, nil
}

// NewGitRepositoryResolver returns a resolver that inspects Git configuration
// in dir. An empty dir uses the process working directory.
func NewGitRepositoryResolver(dir string) RepositoryResolver {
	return func() (Repository, error) {
		git := gitRunner{dir: dir}
		inside, err := git.output("rev-parse", "--is-inside-work-tree")
		if errors.Is(err, exec.ErrNotFound) {
			return Repository{}, errors.New("unable to determine repository: Git executable not found; install Git or pass owner/repo explicitly")
		}
		if err != nil || strings.TrimSpace(inside) != "true" {
			return Repository{}, errors.New("unable to determine repository: current directory is not a Git repository; pass owner/repo explicitly")
		}

		remoteOutput, err := git.output("remote")
		if err != nil {
			return Repository{}, fmt.Errorf("list Git remotes: %w", err)
		}
		remoteNames := strings.Fields(remoteOutput)
		if len(remoteNames) == 0 {
			return Repository{}, errors.New("unable to determine repository: no Git remotes configured; pass owner/repo explicitly")
		}

		valid := make(map[string]Repository)
		invalid := make([]string, 0)
		for _, name := range remoteNames {
			remoteURL, getErr := git.output("remote", "get-url", name)
			if getErr != nil {
				invalid = append(invalid, name)
				continue
			}
			repository, parseErr := parseAtomGitRemoteURL(strings.TrimSpace(remoteURL))
			switch {
			case parseErr == nil:
				valid[name] = repository
			case !errors.Is(parseErr, errNotAtomGitRemote):
				invalid = append(invalid, name)
			}
		}

		if name, ok, configErr := git.optionalOutput("config", "--get", "remote.pushDefault"); configErr != nil {
			return Repository{}, fmt.Errorf("read remote.pushDefault: %w", configErr)
		} else if ok {
			if repository, found := valid[strings.TrimSpace(name)]; found {
				return repository, nil
			}
		}

		if branch, ok, branchErr := git.optionalOutput("symbolic-ref", "--quiet", "--short", "HEAD"); branchErr != nil {
			return Repository{}, fmt.Errorf("read current Git branch: %w", branchErr)
		} else if ok {
			key := "branch." + strings.TrimSpace(branch) + ".remote"
			if name, configured, configErr := git.optionalOutput("config", "--get", key); configErr != nil {
				return Repository{}, fmt.Errorf("read current branch remote: %w", configErr)
			} else if configured {
				if repository, found := valid[strings.TrimSpace(name)]; found {
					return repository, nil
				}
			}
		}

		if repository, ok := valid["origin"]; ok {
			return repository, nil
		}

		unique := make(map[string]Repository)
		for _, repository := range valid {
			unique[repository.String()] = repository
		}
		if len(unique) == 1 {
			for _, repository := range unique {
				return repository, nil
			}
		}
		if len(unique) > 1 {
			names := make([]string, 0, len(valid))
			for name := range valid {
				names = append(names, name)
			}
			sort.Strings(names)
			return Repository{}, fmt.Errorf("unable to determine repository: AtomGit/GitCode remotes conflict (%s); pass owner/repo explicitly or configure remote.pushDefault", strings.Join(names, ", "))
		}
		if len(invalid) > 0 {
			sort.Strings(invalid)
			return Repository{}, fmt.Errorf("unable to determine repository: invalid AtomGit/GitCode remote URL for %s; pass owner/repo explicitly", strings.Join(invalid, ", "))
		}
		return Repository{}, errors.New("unable to determine repository: no AtomGit/GitCode remote found; pass owner/repo explicitly")
	}
}

func parseAtomGitRemoteURL(value string) (Repository, error) {
	if value == "" {
		return Repository{}, errors.New("empty remote URL")
	}

	if !strings.Contains(value, "://") {
		at := strings.LastIndex(value, "@")
		colon := strings.Index(value, ":")
		if colon > at {
			host := value[at+1 : colon]
			if !isAtomGitHost(host) {
				return Repository{}, errNotAtomGitRemote
			}
			return parseRemotePath(value[colon+1:])
		}
		if containsKnownHost(value) {
			return Repository{}, errors.New("invalid AtomGit/GitCode remote URL")
		}
		return Repository{}, errNotAtomGitRemote
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return Repository{}, errors.New("invalid remote URL")
	}
	if !isAtomGitHost(parsed.Hostname()) {
		return Repository{}, errNotAtomGitRemote
	}
	if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
		return Repository{}, errors.New("unsupported AtomGit remote URL scheme")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return Repository{}, errors.New("invalid AtomGit remote URL")
	}
	return parseRemotePath(parsed.Path)
}

func parseRemotePath(value string) (Repository, error) {
	path := strings.Trim(strings.TrimSpace(value), "/")
	path = strings.TrimSuffix(path, ".git")
	return ParseRepository(path)
}

type gitRunner struct {
	dir string
}

func (g gitRunner) output(args ...string) (string, error) {
	if g.dir != "" {
		args = append([]string{"-C", g.dir}, args...)
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (g gitRunner) optionalOutput(args ...string) (string, bool, error) {
	output, err := g.output(args...)
	if err == nil {
		return output, true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, err
}
