package pr

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/git"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// scpURLRe matches SCP-style git URLs (git@host:owner/repo). It is used by
// findRemoteByURL below, which compares locally configured remotes against
// the AtomGit-reported owner/repo of a PR. This local matcher intentionally
// coexists with cmdutil.parseAtomGitRemoteURL: cmdutil is used to infer the
// current-repo owner/repo for argument resolution, whereas findRemoteByURL
// answers a different question — "does this repo have a remote pointing at
// PR base/head?" — and needs to accept gitcode.com mirrors as well.

var scpURLRe = regexp.MustCompile(`^git@([^:]+):(.+)$`)

var fullNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)

// validateFullName checks that a repository identifier from the AtomGit API
// is a plain "owner/repo" pair. This is a defensive check against an
// unexpected upstream response smuggling path segments (e.g. "../evil/x")
// or a scheme prefix into fields we splice into URLs and refspecs.
func validateFullName(field, s string) error {
	if s == "" {
		return fmt.Errorf("%s is empty", field)
	}
	if !fullNameRe.MatchString(s) {
		return fmt.Errorf("%s %q is not a valid owner/repo identifier", field, s)
	}
	if strings.Contains(s, "..") {
		return fmt.Errorf("%s %q contains forbidden segment", field, s)
	}
	return nil
}

// isAtomGitHost reports whether host is an AtomGit server.
// Comparison is case-insensitive and ignores port suffixes (e.g. ":443").
func isAtomGitHost(host string) bool {
	if idx := strings.LastIndex(host, ":"); idx != -1 && !strings.Contains(host[idx+1:], ":") {
		host = host[:idx] // strip port for IPv4-style host:port
	}
	host = strings.ToLower(host)
	return host == "atomgit.com" || host == "gitcode.com"
}

type checkoutOptions struct {
	Branch            string
	Force             bool
	Detach            bool
	RecurseSubmodules bool
}

func newCmdPRCheckout(f *cmdutil.Factory) *cobra.Command {
	var opts checkoutOptions
	cmd := &cobra.Command{
		Use:   "checkout [<owner>/<repo>] <number>",
		Short: "Check out a pull request locally",
		Long: `Check out the source branch of a pull request into a local branch.

When run inside a git repository, the owner and repository are inferred from
the current git remote. You can also specify them explicitly.

For same-repository PRs, the source branch is fetched from the matching remote.
For fork PRs, a temporary remote is added to fetch the source branch.

By default, the command will not overwrite uncommitted changes or existing local
branches. Use --force to override these safety checks.`,
		Example: `  # Check out PR #42, inferring the repository from git remote
  ag pr checkout 42

  # Check out PR #42 from a specific repository
  ag pr checkout owner/repo 42

  # Check out to a custom branch name
  ag pr checkout 42 --branch review-fix

  # Force checkout, discarding safety checks
  ag pr checkout 42 --force

  # Check out in detached HEAD mode and update submodules
  ag pr checkout 42 --detach --recurse-submodules`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			gitClient := git.NewClient()
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to determine current working directory: %w", err)
			}
			gitClient.Dir = wd
			gitClient.Stderr = cmd.ErrOrStderr()

			// Resolve owner/repo via the shared cmdutil helper. It parses
			// an explicit "owner/repo" arg when present, otherwise falls
			// back to the factory-installed RepositoryResolver (which
			// inspects git remotes and honours remote.pushDefault, the
			// current branch upstream, then origin).
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				// Narrow bridge for a single host-support gap in
				// cmdutil.parseAtomGitRemoteURL (upstream PR #68), which
				// only accepts atomgit.com. When the shared resolver
				// specifically reports "no AtomGit remote found" for a
				// single-argument invocation, retry against gitcode.com
				// mirrors using the same selection precedence
				// (remote.pushDefault -> branch upstream -> origin ->
				// unique mirror; conflicting mirrors error out). Every
				// other resolver error (ambiguity, invalid URL, not a
				// Git repository, argument-parse) propagates unchanged.
				if len(args) == 1 && isNoAtomGitRemoteError(err) {
					o, r, fbErr := inferRepoFromMirrorRemote(cmd.Context(), gitClient)
					if fbErr == nil {
						repository = cmdutil.Repository{Owner: o, Name: r}
						remaining = args
					} else {
						return fbErr
					}
				} else {
					return err
				}
			}
			owner, repo := repository.Owner, repository.Name
			number, err := parsePRNumber(remaining[0])
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			apiClient, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			return runCheckout(cmd.Context(), gitClient, apiClient, owner, repo, number, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Local branch name (default: PR head branch name)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force checkout, bypassing safety checks for dirty tree and branch conflicts")
	cmd.Flags().BoolVar(&opts.Detach, "detach", false, "Check out PR in detached HEAD mode")
	cmd.Flags().BoolVar(&opts.RecurseSubmodules, "recurse-submodules", false, "Update submodules after checkout")
	return cmd
}

// runCheckout is the core logic, separated for testability.
func runCheckout(ctx context.Context, gitClient *git.Client, apiClient *api.Client,
	owner, repo, number string, opts checkoutOptions, stdout io.Writer) (err error) {

	var cleanupRemote string

	defer func() {
		if cleanupRemote != "" && err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if rmErr := gitClient.RemoteRemove(cleanupCtx, cleanupRemote); rmErr != nil {
				err = fmt.Errorf("%w (additionally, failed to clean up temporary remote %s: %v)", err, cleanupRemote, rmErr)
			}
		}
	}()

	// 1. Fetch PR metadata
	var pr api.PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
	if err := apiClient.Get(path, &pr); err != nil {
		return fmt.Errorf("failed to get PR #%s: %w", number, err)
	}

	// Defensive validation of upstream-provided identifiers before splicing
	// them into URLs, refspecs, or shell arguments. head.repo.full_name is
	// allowed to be empty (upstream signal that the head repo was deleted);
	// that case is handled where we decide the remote.
	if err := validateFullName("PR base repository", pr.Base.Repo.FullName); err != nil {
		return err
	}
	if pr.Head.Repo.FullName != "" {
		if err := validateFullName("PR head repository", pr.Head.Repo.FullName); err != nil {
			return err
		}
	}
	if pr.Head.Ref == "" {
		return fmt.Errorf("PR #%s has no head branch reference", number)
	}
	if err := gitClient.CheckBranchName(ctx, pr.Head.Ref); err != nil {
		return fmt.Errorf("invalid PR head ref %q: %w", pr.Head.Ref, err)
	}

	// 2. Determine local branch name and handle default branch collision.
	localBranch := pr.Head.Ref
	if opts.Branch != "" {
		localBranch = opts.Branch
	}
	// When the PR head branch name matches the base branch name AND the user
	// did not explicitly specify a branch name, prefix with the PR head
	// repository owner to avoid shadowing the base branch.
	if !opts.Detach && opts.Branch == "" && localBranch == pr.Base.Ref {
		headOwner := pr.Head.User.Login
		if headOwner == "" {
			headOwner = pr.Head.Repo.Owner.Login
		}
		if headOwner == "" {
			headOwner = pr.User.Login
		}
		if headOwner != "" {
			localBranch = headOwner + "/" + localBranch
		}
	}

	// Validate branch name using git's own ref-format rules.
	if err := gitClient.CheckBranchName(ctx, localBranch); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", localBranch, err)
	}

	// 3. Safety: dirty working tree
	if !opts.Force {
		status, err := gitClient.StatusPorcelain(ctx)
		if err != nil {
			return fmt.Errorf("failed to check working tree: %w", err)
		}
		if status != "" {
			return fmt.Errorf("working tree is not clean; use --force to override")
		}
	}

	// 4. Safety: existing branch (skip when detaching — detached HEAD ignores branches).
	skipFetchCheckout := false
	if !opts.Detach {
		if gitClient.HasLocalBranch(ctx, localBranch) {
			sha, err := gitClient.RevParse(ctx, "refs/heads/"+localBranch)
			if err != nil {
				return fmt.Errorf("failed to check local branch %s: %w", localBranch, err)
			}
			if strings.EqualFold(sha, pr.Head.SHA) {
				// Branch exists at correct commit — just switch to it.
				coArgs := []string{"checkout"}
				if opts.Force {
					coArgs = append(coArgs, "--force")
				}
				coArgs = append(coArgs, localBranch)
				if err := gitClient.Run(ctx, coArgs...); err != nil {
					return fmt.Errorf("failed to switch to branch %s: %w", localBranch, err)
				}
				fmt.Fprintf(stdout, "branch %s already exists and is at %s\n", localBranch, shortSHA(pr.Head.SHA))
				skipFetchCheckout = true
			} else if !opts.Force {
				return fmt.Errorf("branch %s already exists; use --force to override", localBranch)
			}
		}
	}

	if !skipFetchCheckout {
		// 5. Determine remote and refspec
		headFullName := pr.Head.Repo.FullName
		baseFullName := pr.Base.Repo.FullName
		isFork := headFullName != baseFullName

		remotes, rErr := gitClient.Remotes(ctx)
		if rErr != nil {
			return fmt.Errorf("failed to list git remotes: %w", rErr)
		}

		var remote string
		if isFork {
			// Fork PR: fail fast if the upstream reports the head repo as
			// deleted; there is nothing meaningful to check out.
			if headFullName == "" {
				return fmt.Errorf("the head repository for PR #%s has been deleted", number)
			}
			// Require the current repository to have an AtomGit remote matching
			// either the base or head repo. This prevents accidentally checking
			// a PR into an unrelated project's clone.
			baseRemote := findRemoteByURL(remotes, baseFullName)
			headRemote := findRemoteByURL(remotes, headFullName)
			if baseRemote == "" && headRemote == "" {
				return fmt.Errorf(
					"current repository has no AtomGit remote matching %s or %s; run this command from a clone of one of them, or pass owner/repo explicitly",
					baseFullName, headFullName,
				)
			}
			if headRemote != "" {
				// Reuse an existing remote pointing to the fork repo.
				remote = headRemote
			} else {
				remote = uniqueRemoteName(remotes, number)
				forkURL := repoGitURL(headFullName)
				if err := gitClient.RemoteAdd(ctx, remote, forkURL); err != nil {
					return fmt.Errorf("failed to add remote %s: %w", remote, err)
				}
				cleanupRemote = remote
			}
		} else {
			// Same-repo PR: must find a matching AtomGit remote.
			remote = findRemoteByURL(remotes, baseFullName)
			if remote == "" {
				return fmt.Errorf("no local AtomGit remote points to %s; add a remote or specify the repository explicitly", baseFullName)
			}
		}

		// Build fetch refspec. For detached mode, omit the local tracking ref
		// so git writes to FETCH_HEAD instead of a remote-tracking branch.
		refspec := fmt.Sprintf("+refs/heads/%s", pr.Head.Ref)
		if !opts.Detach {
			refspec += fmt.Sprintf(":refs/remotes/%s/%s", remote, pr.Head.Ref)
		}

		// 6. Fetch
		fetchArgs := []string{"fetch", "--no-tags"}
		if opts.Force {
			fetchArgs = append(fetchArgs, "--force")
		}
		fetchArgs = append(fetchArgs, remote, refspec)
		if err := gitClient.Run(ctx, fetchArgs...); err != nil {
			return fmt.Errorf("failed to fetch from %s: %w", remote, err)
		}
		// Once fetch succeeds, the temporary remote has produced tracking refs
		// the user can rely on for manual recovery if a later step fails.
		// Keep it so we do not leave dangling refs/remotes/pr-N/* references.
		cleanupRemote = ""

		// 7. Checkout
		if opts.Detach {
			coArgs := []string{"checkout", "--detach"}
			if opts.Force {
				coArgs = append(coArgs, "--force")
			}
			coArgs = append(coArgs, "FETCH_HEAD")
			if err := gitClient.Run(ctx, coArgs...); err != nil {
				return fmt.Errorf("failed to checkout PR #%s in detached HEAD: %w", number, err)
			}
		} else {
			track := fmt.Sprintf("%s/%s", remote, pr.Head.Ref)
			if err := gitClient.Checkout(ctx, localBranch, track, opts.Force); err != nil {
				return fmt.Errorf("failed to checkout branch %s: %w", localBranch, err)
			}
		}
	}

	// 8. Submodules
	if opts.RecurseSubmodules {
		if err := gitClient.Run(ctx, "submodule", "sync", "--recursive"); err != nil {
			return fmt.Errorf("failed to sync submodules: %w", err)
		}
		if err := gitClient.Run(ctx, "submodule", "update", "--init", "--recursive"); err != nil {
			return fmt.Errorf("failed to update submodules: %w", err)
		}
	}

	// 9. Output
	if !opts.Detach && !skipFetchCheckout {
		fmt.Fprintf(stdout, "Switched to branch '%s'\n", localBranch)
	}
	fmt.Fprintf(stdout, "View PR at: %s\n", pr.HTMLURL)
	return nil
}

// repoGitURL constructs a git clone URL from a full repository name.
func repoGitURL(fullName string) string {
	return "https://atomgit.com/" + fullName + ".git"
}

// isNoAtomGitRemoteError reports whether err is the specific
// "no AtomGit remote found" sentinel string returned by
// cmdutil.NewGitRepositoryResolver when every configured remote has an
// unsupported host. It intentionally matches by substring because
// cmdutil does not export the error value. Every other resolver error
// (conflict, invalid URL, not a Git repository, argument-parse) must NOT
// trigger the mirror fallback, so this predicate is narrow by design.
func isNoAtomGitRemoteError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no AtomGit remote found")
}

// inferRepoFromMirrorRemote picks a gitcode.com mirror remote using the
// exact selection precedence cmdutil.NewGitRepositoryResolver applies to
// atomgit.com remotes:
//
//  1. remote.pushDefault (if the referenced remote is a supported mirror);
//  2. the current branch's upstream (branch.<X>.remote);
//  3. a remote literally named "origin";
//  4. the unique mirror across all remotes.
//
// If step 4 finds multiple mirrors pointing at different owner/repo
// pairs, an ambiguity error is returned instead of silently taking the
// first one; this mirrors cmdutil's "AtomGit remotes conflict" behaviour
// and prevents accidentally fetching a same-numbered PR from the wrong
// repository. Callers should have already established that cmdutil's
// atomgit.com resolver reported "no AtomGit remote found" for the
// current directory (see isNoAtomGitRemoteError); otherwise this
// function's own selection rules would double-run on top of cmdutil's.
func inferRepoFromMirrorRemote(ctx context.Context, gitClient *git.Client) (string, string, error) {
	remotes, err := gitClient.Remotes(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to list git remotes: %w", err)
	}

	valid := make(map[string]string) // remote name -> "owner/repo"
	for _, r := range remotes {
		if owner, repo, ok := parseMirrorRemoteURL(r.URL); ok {
			valid[r.Name] = owner + "/" + repo
		}
	}
	if len(valid) == 0 {
		return "", "", fmt.Errorf("unable to determine repository: no AtomGit remote found; pass owner/repo explicitly")
	}

	pick := func(name string) (string, string, bool) {
		fullName, ok := valid[strings.TrimSpace(name)]
		if !ok {
			return "", "", false
		}
		parts := strings.SplitN(fullName, "/", 2)
		return parts[0], parts[1], true
	}

	if pd, err := gitClient.Config(ctx, "remote.pushDefault"); err == nil {
		if o, r, ok := pick(pd); ok {
			return o, r, nil
		}
	}

	// Read current branch name via symbolic-ref, then its upstream remote.
	if branch, err := gitClient.Output(ctx, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		branch = strings.TrimSpace(branch)
		if branch != "" {
			if up, err := gitClient.Config(ctx, "branch."+branch+".remote"); err == nil {
				if o, r, ok := pick(up); ok {
					return o, r, nil
				}
			}
		}
	}

	if o, r, ok := pick("origin"); ok {
		return o, r, nil
	}

	unique := make(map[string][2]string)
	for _, fullName := range valid {
		parts := strings.SplitN(fullName, "/", 2)
		unique[fullName] = [2]string{parts[0], parts[1]}
	}
	if len(unique) == 1 {
		for _, v := range unique {
			return v[0], v[1], nil
		}
	}

	names := make([]string, 0, len(valid))
	for n := range valid {
		names = append(names, n)
	}
	sort.Strings(names)
	return "", "", fmt.Errorf(
		"unable to determine repository: AtomGit mirror remotes conflict (%s); pass owner/repo explicitly or configure remote.pushDefault",
		strings.Join(names, ", "),
	)
}

// parseMirrorRemoteURL extracts owner/repo from an HTTPS or SCP-style git URL
// whose host passes isAtomGitHost. Returns ok=false when the URL is not on a
// supported host or is otherwise malformed.
func parseMirrorRemoteURL(rawURL string) (owner, repo string, ok bool) {
	var path string
	if m := scpURLRe.FindStringSubmatch(rawURL); m != nil {
		if !isAtomGitHost(m[1]) {
			return "", "", false
		}
		path = m[2]
	} else {
		u, err := url.Parse(rawURL)
		if err != nil || !isAtomGitHost(u.Host) {
			return "", "", false
		}
		path = strings.TrimPrefix(u.Path, "/")
	}
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// findRemoteByURL checks if any remote's URL targets the given repository
// fullName (e.g., "forker/repo"). It parses each remote URL and compares
// the path component (stripped of .git suffix) case-insensitively.
func findRemoteByURL(remotes []git.Remote, fullName string) string {
	for _, r := range remotes {
		// Handle SCP-style: git@host:owner/repo.git
		if m := scpURLRe.FindStringSubmatch(r.URL); m != nil {
			if !isAtomGitHost(m[1]) {
				continue
			}
			path := strings.TrimSuffix(m[2], ".git")
			if strings.EqualFold(path, fullName) {
				return r.Name
			}
			continue
		}
		// Handle HTTPS: https://host/owner/repo.git
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		if !isAtomGitHost(u.Host) {
			continue
		}
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		if strings.EqualFold(path, fullName) {
			return r.Name
		}
	}
	return ""
}

// shortSHA returns the first 7 characters of a SHA.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// uniqueRemoteName returns a remote name like "pr-<number>", appending "-1",
// "-2", etc. if a remote with that name already exists.
func uniqueRemoteName(remotes []git.Remote, number string) string {
	base := fmt.Sprintf("pr-%s", number)
	existing := make(map[string]bool, len(remotes))
	for _, r := range remotes {
		existing[r.Name] = true
	}
	if !existing[base] {
		return base
	}
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%d", base, i)
		if !existing[name] {
			return name
		}
	}
}
