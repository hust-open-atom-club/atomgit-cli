package cmdutil

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAtomGitRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    Repository
		wantErr string
	}{
		{name: "scp SSH", url: "git@atomgit.com:owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "SSH URL", url: "ssh://git@atomgit.com/owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "HTTPS", url: "https://atomgit.com/owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "HTTPS without suffix", url: "https://atomgit.com/owner/repo", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "scp SSH gitcode", url: "git@gitcode.com:owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "SSH URL gitcode", url: "ssh://git@gitcode.com/owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "HTTPS gitcode", url: "https://gitcode.com/owner/repo.git", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "HTTPS gitcode without suffix", url: "https://gitcode.com/owner/repo", want: Repository{Owner: "owner", Name: "repo"}},
		{name: "GitHub", url: "https://github.com/owner/repo.git", wantErr: "not an AtomGit/GitCode remote"},
		{name: "GitLab SSH", url: "git@gitlab.com:owner/repo.git", wantErr: "not an AtomGit/GitCode remote"},
		{name: "wrong path", url: "https://atomgit.com/owner", wantErr: "expected owner/repo"},
		{name: "wrong path gitcode", url: "https://gitcode.com/owner", wantErr: "expected owner/repo"},
		{name: "extra path", url: "https://atomgit.com/owner/repo/extra.git", wantErr: "expected owner/repo"},
		{name: "extra path gitcode", url: "https://gitcode.com/owner/repo/extra.git", wantErr: "expected owner/repo"},
		{name: "unsupported scheme", url: "http://atomgit.com/owner/repo.git", wantErr: "unsupported"},
		{name: "unsupported scheme gitcode", url: "http://gitcode.com/owner/repo.git", wantErr: "unsupported"},
		{name: "query", url: "https://atomgit.com/owner/repo.git?token=redacted", wantErr: "invalid"},
		{name: "query gitcode", url: "https://gitcode.com/owner/repo.git?token=redacted", wantErr: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAtomGitRemoteURL(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("repository = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveRepositoryExplicitDoesNotUseResolver(t *testing.T) {
	called := false
	factory := &Factory{RepositoryResolver: func() (Repository, error) {
		called = true
		return Repository{}, nil
	}}

	got, err := ResolveRepository(factory, "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "owner/repo" {
		t.Fatalf("repository = %s", got.String())
	}
	if called {
		t.Fatal("resolver was called for an explicit repository")
	}
}

func TestResolveRepositoryFromArgs(t *testing.T) {
	factory := &Factory{RepositoryResolver: func() (Repository, error) {
		return Repository{Owner: "inferred", Name: "repo"}, nil
	}}

	tests := []struct {
		name      string
		args      []string
		trailing  int
		wantRepo  string
		wantArgs  []string
		wantError bool
	}{
		{name: "inferred list", wantRepo: "inferred/repo"},
		{name: "explicit list", args: []string{"owner/repo"}, wantRepo: "owner/repo"},
		{name: "explicit empty", args: []string{""}, wantError: true},
		{name: "explicit short name", args: []string{"repo"}, wantError: true},
		{name: "inferred resource", args: []string{"42"}, trailing: 1, wantRepo: "inferred/repo", wantArgs: []string{"42"}},
		{name: "explicit resource", args: []string{"owner/repo", "42"}, trailing: 1, wantRepo: "owner/repo", wantArgs: []string{"42"}},
		{name: "too few", trailing: 1, wantError: true},
		{name: "too many", args: []string{"owner/repo", "42", "extra"}, trailing: 1, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, remaining, err := ResolveRepositoryFromArgs(factory, tt.args, tt.trailing)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repository.String() != tt.wantRepo {
				t.Fatalf("repository = %q, want %q", repository.String(), tt.wantRepo)
			}
			if strings.Join(remaining, ",") != strings.Join(tt.wantArgs, ",") {
				t.Fatalf("remaining = %v, want %v", remaining, tt.wantArgs)
			}
		})
	}
}

func TestGitRepositoryResolver(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*testing.T, string)
		want       string
		wantErr    string
		notGitRepo bool
	}{
		{
			name: "origin SSH",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "git@atomgit.com:alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "origin HTTPS",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "https://atomgit.com/alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "push default wins",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "git@atomgit.com:alice/origin.git")
				gitRun(t, dir, "remote", "add", "work", "git@atomgit.com:alice/work.git")
				gitRun(t, dir, "config", "remote.pushDefault", "work")
			},
			want: "alice/work",
		},
		{
			name: "branch remote wins",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "git@atomgit.com:alice/origin.git")
				gitRun(t, dir, "remote", "add", "upstream", "git@atomgit.com:team/upstream.git")
				branch := strings.TrimSpace(gitOutput(t, dir, "symbolic-ref", "--short", "HEAD"))
				gitRun(t, dir, "config", "branch."+branch+".remote", "upstream")
			},
			want: "team/upstream",
		},
		{
			name: "unique AtomGit ignores foreign remote",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "github", "https://github.com/alice/demo.git")
				gitRun(t, dir, "remote", "add", "atomgit", "git@atomgit.com:alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "duplicate repository is not a conflict",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "one", "git@atomgit.com:alice/demo.git")
				gitRun(t, dir, "remote", "add", "two", "https://atomgit.com/alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "conflicting AtomGit remotes",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "one", "git@atomgit.com:alice/one.git")
				gitRun(t, dir, "remote", "add", "two", "git@atomgit.com:alice/two.git")
			},
			wantErr: "conflict",
		},
		{
			name:      "no remotes",
			configure: func(*testing.T, string) {},
			wantErr:   "no Git remotes",
		},
		{
			name: "origin SSH gitcode",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "git@gitcode.com:alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "origin HTTPS gitcode",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "https://gitcode.com/alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "push default wins gitcode",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "git@gitcode.com:alice/origin.git")
				gitRun(t, dir, "remote", "add", "work", "git@gitcode.com:alice/work.git")
				gitRun(t, dir, "config", "remote.pushDefault", "work")
			},
			want: "alice/work",
		},
		{
			name: "mixed atomgit and gitcode same repo",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "one", "git@atomgit.com:alice/demo.git")
				gitRun(t, dir, "remote", "add", "two", "https://gitcode.com/alice/demo.git")
			},
			want: "alice/demo",
		},
		{
			name: "invalid AtomGit URL",
			configure: func(t *testing.T, dir string) {
				gitRun(t, dir, "remote", "add", "origin", "https://atomgit.com/only-owner.git")
			},
			wantErr: "invalid AtomGit/GitCode remote URL",
		},
		{
			name:       "not a Git repository",
			notGitRepo: true,
			wantErr:    "not a Git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if !tt.notGitRepo {
				gitRun(t, dir, "init")
				tt.configure(t, dir)
			}
			repository, err := NewGitRepositoryResolver(dir)()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repository.String() != tt.want {
				t.Fatalf("repository = %q, want %q", repository.String(), tt.want)
			}
		})
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = gitOutput(t, dir, args...)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", filepath.Clean(dir)}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
