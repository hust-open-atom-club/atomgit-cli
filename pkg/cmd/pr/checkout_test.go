package pr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/git"
)

// ---------------------------------------------------------------------------
// Git mock helpers (re-execution pattern)
// ---------------------------------------------------------------------------

type gitCall struct {
	Args []string // includes "git" as first element
}

type mockResult struct {
	exitCode int
	stdout   string
}

func newTestGitClient(t *testing.T, results map[string]mockResult) (*git.Client, *[]gitCall) {
	t.Helper()
	var calls []gitCall
	var mu sync.Mutex

	fn := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		mu.Lock()
		defer mu.Unlock()
		allArgs := append([]string{name}, args...)
		calls = append(calls, gitCall{Args: allArgs})

		// Determine response key
		key := ""
		if len(args) > 0 {
			key = args[0]
			if key == "rev-parse" && len(args) > 2 {
				// args may be ["--verify", "--quiet", "refs/heads/..."]
				// or       ["--verify", "refs/heads/..."]
				for i := 1; i < len(args); i++ {
					if !strings.HasPrefix(args[i], "-") {
						key = "rev-parse:" + args[i]
						break
					}
				}
			}
			if key == "remote" && len(args) > 1 {
				key = key + ":" + args[1] // remote:add, remote:get-url
			}
			if key == "config" && len(args) > 1 {
				// Distinguish per-config-key so tests can seed different
				// stdout/exit codes for e.g. remote.pushDefault vs.
				// branch.<name>.remote. `git config <key>` uses arg[1].
				key = key + ":" + args[1]
			}
			if key == "check-ref-format" && len(args) > 0 {
				// Distinguish per-branch-name so tests can target a specific
				// argument (e.g. only "--evil" fails while a legitimate PR
				// head ref passes).
				key = key + ":" + args[len(args)-1]
			}
		}

		r, ok := results[key]
		if !ok {
			r = mockResult{exitCode: 0, stdout: ""}
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestCheckoutGitHelper")
		cmd.Env = append(os.Environ(),
			"AG_CHECKOUT_GIT_HELPER=1",
			fmt.Sprintf("AG_EXIT_CODE=%d", r.exitCode),
			"AG_STDOUT="+r.stdout,
		)
		return cmd
	}
	return git.NewTestClient(fn), &calls
}

// TestCheckoutGitHelper is the sentinel test for the re-execution mock pattern.
func TestCheckoutGitHelper(t *testing.T) {
	if os.Getenv("AG_CHECKOUT_GIT_HELPER") != "1" {
		return
	}
	exitCode, _ := strconv.Atoi(os.Getenv("AG_EXIT_CODE"))
	stdout := os.Getenv("AG_STDOUT")

	if exitCode != 0 {
		fmt.Fprint(os.Stderr, "git: simulated error\n")
	}
	fmt.Print(stdout)
	os.Exit(exitCode)
}

// ---------------------------------------------------------------------------
// HTTP mock helpers — reused from pr_test.go (prRoundTripFunc, prResponse)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

// sameRepoPR returns JSON for a PR where head and base are in the same repo.
func sameRepoPR() string {
	return `{
		"head": {"ref": "feature", "sha": "abc123def456abc123def456abc123def456abc1", "repo": {"full_name": "owner/repo"}},
		"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "owner/repo"}},
		"html_url": "https://atomgit.com/owner/repo/pull/42"
	}`
}

func TestRunCheckout_SameRepoPR(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify fetch called with "origin" remote and --no-tags
	foundFetch := false
	for _, call := range *calls {
		if len(call.Args) == 5 && call.Args[1] == "fetch" && call.Args[2] == "--no-tags" && call.Args[3] == "origin" && call.Args[4] == "+refs/heads/feature:refs/remotes/origin/feature" {
			foundFetch = true
			break
		}
	}
	if !foundFetch {
		t.Fatal("expected fetch --no-tags origin, but not found")
	}

	// Verify checkout called with -b and --track origin/feature
	foundCheckout := false
	for _, call := range *calls {
		if len(call.Args) == 6 && call.Args[1] == "checkout" && call.Args[2] == "-b" && call.Args[3] == "feature" && call.Args[4] == "--track" && call.Args[5] == "origin/feature" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout -b feature --track origin/feature, but not found")
	}
}

func TestRunCheckout_SameRepoPR_NoMatchingRemote_Error(t *testing.T) {
	// Same-repo PR where no local remote points to the base repository.
	// Must return an error instead of falling back to a wrong remote.
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://other.com/different/repo.git (fetch)\n"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error when no AtomGit remote matches, got nil")
	}
	if !strings.Contains(err.Error(), "no local AtomGit remote") {
		t.Fatalf("error = %q, want containing 'no local AtomGit remote'", err.Error())
	}

	// Verify no fetch or checkout was called.
	for _, call := range *calls {
		if len(call.Args) > 1 && (call.Args[1] == "fetch" || call.Args[1] == "checkout") {
			t.Fatalf("unexpected fetch/checkout: %v", call.Args)
		}
	}
}

// forkPR returns JSON for a PR where head is in a different repo (fork).
func forkPR() string {
	return `{
		"head": {"ref": "patch-fix", "sha": "def456abc123def456abc123def456abc123def4", "repo": {"full_name": "forker/repo"}},
		"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "owner/repo"}},
		"html_url": "https://atomgit.com/owner/repo/pull/42"
	}`
}

func TestRunCheckout_ForkPR(t *testing.T) {
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		// The current clone belongs to the base repo (owner/repo); no remote for the fork yet.
		"remote:-v":  {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add": {exitCode: 0, stdout: ""},
		"fetch":      {exitCode: 0, stdout: ""},
		"checkout":   {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify remote pr-42 added with correct URL
	foundRemoteAdd := false
	for _, call := range *calls {
		if len(call.Args) == 5 && call.Args[1] == "remote" && call.Args[2] == "add" && call.Args[3] == "pr-42" && call.Args[4] == "https://atomgit.com/forker/repo.git" {
			foundRemoteAdd = true
			break
		}
	}
	if !foundRemoteAdd {
		t.Fatal("expected remote add pr-42 with fork URL, but not found")
	}

	// Verify fetch uses fork remote with --no-tags
	foundFetch := false
	for _, call := range *calls {
		if len(call.Args) == 5 && call.Args[1] == "fetch" && call.Args[2] == "--no-tags" && call.Args[3] == "pr-42" {
			foundFetch = true
			break
		}
	}
	if !foundFetch {
		t.Fatal("expected fetch --no-tags from pr-42, but not found")
	}

	// Verify checkout uses fork remote
	foundCheckout := false
	for _, call := range *calls {
		if len(call.Args) == 6 && call.Args[1] == "checkout" && call.Args[2] == "-b" && call.Args[5] == "pr-42/patch-fix" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout -b patch-fix --track pr-42/patch-fix, but not found")
	}
}

func TestRunCheckout_ForkPR_ExistingRemote(t *testing.T) {
	// When a remote already exists that points to the fork repository,
	// it should be reused instead of adding pr-N.
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		// SCP-style URL for the existing remote
		"remote:-v": {exitCode: 0, stdout: "forker\tgit@atomgit.com:forker/repo.git (fetch)\n"},
		"fetch":     {exitCode: 0, stdout: ""},
		"checkout":  {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify NO remote add was called (reused existing remote)
	for _, call := range *calls {
		if len(call.Args) > 1 && call.Args[1] == "remote" && call.Args[2] == "add" {
			t.Fatal("expected no remote add when existing remote matches")
		}
	}

	// Verify fetch uses the existing remote name "forker" with --no-tags
	foundFetch := false
	for _, call := range *calls {
		if len(call.Args) == 5 && call.Args[1] == "fetch" && call.Args[2] == "--no-tags" && call.Args[3] == "forker" {
			foundFetch = true
			break
		}
	}
	if !foundFetch {
		t.Fatal("expected fetch --no-tags from existing remote 'forker', but not found")
	}
}

func TestRunCheckout_DirtyTree(t *testing.T) {
	results := map[string]mockResult{
		"status": {exitCode: 0, stdout: " M file.go"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for dirty tree, got nil")
	}
	if !strings.Contains(err.Error(), "not clean") {
		t.Fatalf("error = %q, want containing 'not clean'", err.Error())
	}

	// Verify no fetch or checkout was called
	for _, call := range *calls {
		if len(call.Args) > 1 && (call.Args[1] == "fetch" || call.Args[1] == "checkout") {
			t.Fatalf("unexpected git call after dirty tree: %v", call.Args)
		}
	}
}

func TestRunCheckout_DirtyTreeForce(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: " M file.go"},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{Force: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() with --force error = %v", err)
	}

	// Verify fetch and checkout were called despite dirty tree
	foundFetch := false
	foundCheckout := false
	for _, call := range *calls {
		if len(call.Args) > 1 && call.Args[1] == "fetch" {
			foundFetch = true
		}
		if len(call.Args) > 1 && call.Args[1] == "checkout" {
			foundCheckout = true
		}
	}
	if !foundFetch {
		t.Fatal("expected fetch to be called with --force")
	}
	if !foundCheckout {
		t.Fatal("expected checkout to be called with --force")
	}
}

func TestRunCheckout_BranchExistsSameSHA(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "abc123def456abc123def456abc123def456abc1"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "already exists") {
		t.Fatalf("stdout = %q, want containing 'already exists'", stdout.String())
	}
	if !strings.Contains(stdout.String(), "https://atomgit.com/owner/repo/pull/42") {
		t.Fatalf("stdout = %q, want containing PR URL", stdout.String())
	}

	// Verify checkout was called (just git checkout <branch>, no -b, no --track)
	foundCheckout := false
	for _, call := range *calls {
		if len(call.Args) == 3 && call.Args[1] == "checkout" && call.Args[2] == "feature" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout feature (without -b or --track), but not found")
	}

	// Verify no fetch was called
	for _, call := range *calls {
		if len(call.Args) > 1 && call.Args[1] == "fetch" {
			t.Fatalf("unexpected fetch for existing branch at correct SHA: %v", call.Args)
		}
	}
}

func TestRunCheckout_BranchExistsDifferentSHA(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "differentSHA1234567890123456789012345678901234"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for existing branch at different SHA, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want containing 'already exists'", err.Error())
	}

	// Verify no fetch or checkout was called
	for _, call := range *calls {
		if len(call.Args) > 1 && (call.Args[1] == "fetch" || call.Args[1] == "checkout") {
			t.Fatalf("unexpected fetch/checkout for existing branch: %v", call.Args)
		}
	}
}

func TestRunCheckout_BranchExistsDifferentSHA_Force(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "differentSHA1234567890123456789012345678901234"},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{Force: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() with --force error = %v", err)
	}

	// Verify checkout was called with --force and -B
	foundCheckoutForce := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" {
			hasForce := false
			hasB := false
			for _, arg := range call.Args[2:] {
				if arg == "--force" {
					hasForce = true
				}
				if arg == "-B" {
					hasB = true
				}
			}
			if hasForce && hasB {
				foundCheckoutForce = true
				break
			}
		}
	}
	if !foundCheckoutForce {
		t.Fatal("expected checkout with -B (force), but not found")
	}
}

func TestRunCheckout_CustomBranch(t *testing.T) {
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/my-review": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                          {exitCode: 0, stdout: ""},
		"checkout":                       {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{Branch: "my-review"}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify checkout called with custom branch name
	foundCheckout := false
	for _, call := range *calls {
		if len(call.Args) == 6 && call.Args[1] == "checkout" && call.Args[2] == "-b" && call.Args[3] == "my-review" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout -b my-review, but not found")
	}
}

func TestRunCheckout_PRFetchError(t *testing.T) {
	gitClient, calls := newTestGitClient(t, nil)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(404, `{"message":"not found"}`), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get PR") {
		t.Fatalf("error = %q, want containing 'failed to get PR'", err.Error())
	}

	// Verify no git operations were called
	if len(*calls) > 0 {
		t.Fatalf("expected no git calls, got %d", len(*calls))
	}
}

func TestRunCheckout_FetchError(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 128, stdout: "fatal: couldn't find remote ref"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for fetch failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("error = %q, want containing 'failed to fetch'", err.Error())
	}

	// Verify checkout was NOT called
	for _, call := range *calls {
		if len(call.Args) > 1 && call.Args[1] == "checkout" {
			t.Fatal("checkout should not be called after fetch error")
		}
	}
}

func TestRunCheckout_Output(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, _ := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Switched to branch") {
		t.Fatalf("stdout = %q, want containing 'Switched to branch'", output)
	}
	if !strings.Contains(output, "https://atomgit.com/owner/repo/pull/42") {
		t.Fatalf("stdout = %q, want containing PR URL", output)
	}
}

func TestRunCheckout_InvalidBranchName(t *testing.T) {
	results := map[string]mockResult{
		"status":                  {exitCode: 0, stdout: ""},
		"check-ref-format:--evil": {exitCode: 128, stdout: ""},
	}
	gitClient, _ := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Branch: "--evil"}, &stdout)
	if err == nil {
		t.Fatal("expected error for invalid branch name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid branch name") {
		t.Fatalf("error = %q, want containing 'invalid branch name'", err.Error())
	}
}

// deletedForkPR returns JSON for a PR where the head repo has been deleted.
func deletedForkPR() string {
	return `{
		"head": {"ref": "patch-fix", "sha": "def456abc123def456abc123def456abc123def4", "repo": {"full_name": ""}},
		"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "owner/repo"}},
		"html_url": "https://atomgit.com/owner/repo/pull/42"
	}`
}

func TestRunCheckout_DeletedFork(t *testing.T) {
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: ""},
	}
	gitClient, _ := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, deletedForkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for deleted fork, got nil")
	}
	if !strings.Contains(err.Error(), "has been deleted") {
		t.Fatalf("error = %q, want containing 'has been deleted'", err.Error())
	}
}

func TestRunCheckout_Detach(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Detach: true}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() detach error = %v", err)
	}

	// Verify refspec has no local target (FETCH_HEAD mode).
	foundFetch := false
	for _, call := range *calls {
		if call.Args[1] == "fetch" && !strings.Contains(call.Args[len(call.Args)-1], ":refs/remotes") {
			foundFetch = true
			break
		}
	}
	if !foundFetch {
		t.Fatal("expected fetch refspec without :refs/remotes/... for detach mode")
	}

	// Verify checkout used --detach instead of -b.
	foundCheckout := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" && call.Args[2] == "--detach" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout --detach, but not found")
	}

	output := stdout.String()
	if strings.Contains(output, "Switched to branch") {
		t.Fatal("detach mode should not print 'Switched to branch'")
	}
	if !strings.Contains(output, "https://atomgit.com/owner/repo/pull/42") {
		t.Fatal("detach mode should print PR URL")
	}
}

func TestRunCheckout_RecurseSubmodules(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
		"submodule":                    {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{RecurseSubmodules: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() recurse error = %v", err)
	}

	// Verify submodule commands were called.
	syncFound := false
	updateFound := false
	for _, call := range *calls {
		if call.Args[1] == "submodule" {
			if call.Args[2] == "sync" {
				syncFound = true
			}
			if call.Args[2] == "update" {
				updateFound = true
			}
		}
	}
	if !syncFound {
		t.Fatal("expected submodule sync --recursive, not found")
	}
	if !updateFound {
		t.Fatal("expected submodule update --init --recursive, not found")
	}
}

func TestRunCheckout_ForceFetch(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 128, stdout: ""},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Force: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() force error = %v", err)
	}

	// Verify fetch includes --force flag.
	foundForce := false
	for _, call := range *calls {
		for _, arg := range call.Args {
			if arg == "--force" && call.Args[1] == "fetch" {
				foundForce = true
				break
			}
		}
	}
	if !foundForce {
		t.Fatal("expected fetch --force when Force is true")
	}
}

func TestRunCheckout_DefaultBranchCollision(t *testing.T) {
	// PR where head branch name matches the base branch name.
	// Uses pr.Head.User.Login (inside head object).
	body := `{
		"head": {"ref": "main", "sha": "abc123def456abc123def456abc123def456abc1",
			"repo": {"full_name": "forker/repo"},
			"user": {"login": "forker"}},
		"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000",
			"repo": {"full_name": "owner/repo"}},
		"html_url": "https://atomgit.com/owner/repo/pull/42"
	}`

	results := map[string]mockResult{
		"status":                           {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/forker/main": {exitCode: 128, stdout: ""},
		"rev-parse:refs/heads/main":        {exitCode: 128, stdout: ""},
		"remote:-v":                        {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add":                       {exitCode: 0, stdout: ""},
		"fetch":                            {exitCode: 0, stdout: ""},
		"checkout":                         {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, body), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() collision error = %v", err)
	}

	// Verify checkout used the prefixed branch name "forker/main".
	foundCheckout := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" && call.Args[3] == "forker/main" {
			foundCheckout = true
			break
		}
	}
	if !foundCheckout {
		t.Fatal("expected checkout with prefixed branch 'forker/main' after collision")
	}
}

// ---------------------------------------------------------------------------
// F2: Detach mode when branch exists — should still fetch and use --detach.
// ---------------------------------------------------------------------------

func TestRunCheckout_Detach_WhenBranchExists(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "abc123def456abc123def456abc123def456abc1\n"},
		"remote:-v":                    {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                        {exitCode: 0, stdout: ""},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Detach: true}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() detach error = %v", err)
	}

	// Should NOT print "already exists" since detach skips branch check.
	if strings.Contains(stdout.String(), "already exists") {
		t.Fatal("detach mode should not print 'already exists'")
	}

	// Verify checkout --detach was called (not a regular checkout of the existing branch).
	foundDetach := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" && call.Args[2] == "--detach" {
			foundDetach = true
			break
		}
	}
	if !foundDetach {
		t.Fatal("expected checkout --detach, but not found")
	}
}

// ---------------------------------------------------------------------------
// F3+F4: Force when branch already exists and is at the correct SHA.
// ---------------------------------------------------------------------------

func TestRunCheckout_Force_ReuseBranch(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "abc123def456abc123def456abc123def456abc1\n"},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Force: true}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() force error = %v", err)
	}

	// Verify --force flag appears in the checkout args.
	foundForce := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" {
			for _, arg := range call.Args[2:] {
				if arg == "--force" {
					foundForce = true
					break
				}
			}
		}
	}
	if !foundForce {
		t.Fatal("expected --force flag in checkout args when reusing branch with Force option")
	}

	if !strings.Contains(stdout.String(), "already exists") {
		t.Fatal("expected 'already exists' message in stdout")
	}
}

// ---------------------------------------------------------------------------
// F5: RecurseSubmodules when branch already exists.
// ---------------------------------------------------------------------------

func TestRunCheckout_RecurseSubmodules_WhenBranchExists(t *testing.T) {
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "abc123def456abc123def456abc123def456abc1\n"},
		"checkout":                     {exitCode: 0, stdout: ""},
		"submodule":                    {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{RecurseSubmodules: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() recurse error = %v", err)
	}

	// Verify submodule commands were called even though branch already existed.
	syncFound := false
	updateFound := false
	for _, call := range *calls {
		if call.Args[1] == "submodule" {
			if call.Args[2] == "sync" {
				syncFound = true
			}
			if call.Args[2] == "update" {
				updateFound = true
			}
		}
	}
	if !syncFound {
		t.Fatal("expected submodule sync --recursive, not found")
	}
	if !updateFound {
		t.Fatal("expected submodule update --init --recursive, not found")
	}
}

// ---------------------------------------------------------------------------
// F7: findRemoteByURL skips non-AtomGit hosts.
// ---------------------------------------------------------------------------

func TestFindRemoteByURL_SkipsNonAtomGitHost(t *testing.T) {
	remotes := []git.Remote{
		{Name: "github", URL: "https://github.com/forker/repo.git"},
		{Name: "atomgit", URL: "https://atomgit.com/forker/repo.git"},
	}

	name := findRemoteByURL(remotes, "forker/repo")
	if name != "atomgit" {
		t.Fatalf("findRemoteByURL() = %q, want %q", name, "atomgit")
	}
}

// ---------------------------------------------------------------------------
// B4: Remote cleanup on fetch failure
// ---------------------------------------------------------------------------

func TestRunCheckout_RemoteCleanup_FetchFailure(t *testing.T) {
	// Fork PR where fetch fails after adding temp remote.
	// The temp remote should be cleaned up.
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add":                     {exitCode: 0, stdout: ""},
		"fetch":                          {exitCode: 128, stdout: ""},
		"remote:remove":                  {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for fetch failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("error = %q, want containing 'failed to fetch'", err.Error())
	}

	// Verify remote remove was called for cleanup.
	foundRemove := false
	for _, call := range *calls {
		if len(call.Args) >= 4 && call.Args[1] == "remote" && call.Args[2] == "remove" && call.Args[3] == "pr-42" {
			foundRemove = true
			break
		}
	}
	if !foundRemove {
		t.Fatal("expected remote remove for pr-42 after fetch failure, not found")
	}
}

// ---------------------------------------------------------------------------
// B5: Remote cleanup on success — should keep the remote
// ---------------------------------------------------------------------------

func TestRunCheckout_RemoteCleanup_Success(t *testing.T) {
	// Fork PR checkout succeeds; temp remote should be KEPT (not removed).
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add":                     {exitCode: 0, stdout: ""},
		"fetch":                          {exitCode: 0, stdout: ""},
		"checkout":                       {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify remote remove was NOT called (success = keep remote).
	for _, call := range *calls {
		if len(call.Args) >= 3 && call.Args[1] == "remote" && call.Args[2] == "remove" {
			t.Fatalf("unexpected remote remove on success: %v", call.Args)
		}
	}
}

// ---------------------------------------------------------------------------
// B6: Remote name conflict
// ---------------------------------------------------------------------------

func TestRunCheckout_RemoteNameConflict(t *testing.T) {
	// The current clone is a base-repo clone (origin -> owner/repo). It also
	// happens to already have a remote literally named "pr-42" that points
	// somewhere else. The command should use "pr-42-1" for the new temp remote
	// while still satisfying the base-or-head ownership check via "origin".
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v": {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n" +
			"pr-42\thttps://atomgit.com/unrelated/repo.git (fetch)\n"},
		"remote:add": {exitCode: 0, stdout: ""},
		"fetch":      {exitCode: 0, stdout: ""},
		"checkout":   {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify remote add used "pr-42-1" (not "pr-42").
	foundUniqueAdd := false
	for _, call := range *calls {
		if len(call.Args) >= 5 && call.Args[1] == "remote" && call.Args[2] == "add" && call.Args[3] == "pr-42-1" {
			foundUniqueAdd = true
			break
		}
	}
	if !foundUniqueAdd {
		t.Fatal("expected remote add with 'pr-42-1' to avoid name conflict, not found")
	}
}

// ---------------------------------------------------------------------------
// B6b: Fork PR must be refused in an unrelated repository
// ---------------------------------------------------------------------------

func TestRunCheckout_ForkPR_UnrelatedRepo_Rejected(t *testing.T) {
	// The current clone belongs to a completely unrelated project. Neither
	// the base nor the head repository of the PR is represented by any local
	// AtomGit remote, so the command must refuse to add a temp remote and
	// switch the working tree.
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/unrelated/project.git (fetch)\n"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error when current repo has no matching AtomGit remote, got nil")
	}
	if !strings.Contains(err.Error(), "no AtomGit remote matching") {
		t.Fatalf("error = %q, want containing 'no AtomGit remote matching'", err.Error())
	}

	// Verify no destructive git operations happened.
	for _, call := range *calls {
		if len(call.Args) < 2 {
			continue
		}
		switch call.Args[1] {
		case "fetch", "checkout":
			t.Fatalf("unexpected %s in unrelated repo: %v", call.Args[1], call.Args)
		case "remote":
			if len(call.Args) >= 3 && (call.Args[2] == "add" || call.Args[2] == "remove") {
				t.Fatalf("unexpected remote %s in unrelated repo: %v", call.Args[2], call.Args)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// B6c: Fork PR checkout failure — keep the remote (tracking refs already exist)
// ---------------------------------------------------------------------------

func TestRunCheckout_ForkPR_CheckoutFailure_KeepsRemote(t *testing.T) {
	// Fork PR where fetch succeeds but "git checkout -b" fails. The temp
	// remote must be kept because fetch already wrote refs/remotes/pr-42/*
	// that the user may want to recover from manually.
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add":                     {exitCode: 0, stdout: ""},
		"fetch":                          {exitCode: 0, stdout: ""},
		"checkout":                       {exitCode: 128, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected error for checkout failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to checkout") {
		t.Fatalf("error = %q, want containing 'failed to checkout'", err.Error())
	}

	for _, call := range *calls {
		if len(call.Args) >= 3 && call.Args[1] == "remote" && call.Args[2] == "remove" {
			t.Fatalf("unexpected remote remove after checkout failure: %v", call.Args)
		}
	}
}

// ---------------------------------------------------------------------------
// B6d: Fork PR submodule failure — keep the remote (working tree already switched)
// ---------------------------------------------------------------------------

func TestRunCheckout_ForkPR_SubmoduleFailure_KeepsRemote(t *testing.T) {
	// Fork PR where fetch + checkout succeed but submodule sync fails. The
	// temp remote must be kept.
	results := map[string]mockResult{
		"status":                         {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/patch-fix": {exitCode: 128, stdout: ""},
		"remote:-v":                      {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"remote:add":                     {exitCode: 0, stdout: ""},
		"fetch":                          {exitCode: 0, stdout: ""},
		"checkout":                       {exitCode: 0, stdout: ""},
		"submodule":                      {exitCode: 128, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, forkPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{RecurseSubmodules: true}, io.Discard)
	if err == nil {
		t.Fatal("expected error for submodule failure, got nil")
	}
	if !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("error = %q, want containing 'submodule'", err.Error())
	}

	for _, call := range *calls {
		if len(call.Args) >= 3 && call.Args[1] == "remote" && call.Args[2] == "remove" {
			t.Fatalf("unexpected remote remove after submodule failure: %v", call.Args)
		}
	}
}

// ---------------------------------------------------------------------------
// B6e: Defensive validation of upstream-provided identifiers
// ---------------------------------------------------------------------------

func TestRunCheckout_UpstreamIdentifierValidation(t *testing.T) {
	tests := []struct {
		name        string
		prJSON      string
		wantErrPart string
	}{
		{
			name: "malformed base full_name",
			prJSON: `{
				"head": {"ref": "patch-fix", "sha": "def456abc123def456abc123def456abc123def4", "repo": {"full_name": "forker/repo"}},
				"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "../evil/x"}},
				"html_url": "https://atomgit.com/owner/repo/pull/42"
			}`,
			wantErrPart: "PR base repository",
		},
		{
			name: "malformed head full_name",
			prJSON: `{
				"head": {"ref": "patch-fix", "sha": "def456abc123def456abc123def456abc123def4", "repo": {"full_name": "forker/../evil"}},
				"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "owner/repo"}},
				"html_url": "https://atomgit.com/owner/repo/pull/42"
			}`,
			wantErrPart: "PR head repository",
		},
		{
			name: "malformed head ref",
			prJSON: `{
				"head": {"ref": "--evil", "sha": "def456abc123def456abc123def456abc123def4", "repo": {"full_name": "owner/repo"}},
				"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000", "repo": {"full_name": "owner/repo"}},
				"html_url": "https://atomgit.com/owner/repo/pull/42"
			}`,
			wantErrPart: "invalid PR head ref",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := map[string]mockResult{
				"status":                  {exitCode: 0, stdout: ""},
				"check-ref-format:--evil": {exitCode: 1, stdout: ""},
			}
			gitClient, calls := newTestGitClient(t, results)

			body := tc.prJSON
			apiClient := api.NewClientWithHTTPClient("token", &http.Client{
				Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return prResponse(200, body), nil
				}),
			})

			err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42", checkoutOptions{}, io.Discard)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tc.wantErrPart)
			}

			for _, call := range *calls {
				if len(call.Args) < 2 {
					continue
				}
				switch call.Args[1] {
				case "fetch", "checkout":
					t.Fatalf("unexpected %s after validation failure: %v", call.Args[1], call.Args)
				case "remote":
					if len(call.Args) >= 3 && (call.Args[2] == "add" || call.Args[2] == "remove") {
						t.Fatalf("unexpected remote %s after validation failure: %v", call.Args[2], call.Args)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// B7: Detach + Force
// ---------------------------------------------------------------------------

func TestRunCheckout_DetachForce(t *testing.T) {
	results := map[string]mockResult{
		"status":    {exitCode: 0, stdout: ""},
		"fetch":     {exitCode: 0, stdout: ""},
		"checkout":  {exitCode: 0, stdout: ""},
		"remote:-v": {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Detach: true, Force: true}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() detach+force error = %v", err)
	}

	// Verify checkout --detach --force FETCH_HEAD.
	found := false
	for _, call := range *calls {
		if len(call.Args) == 5 && call.Args[1] == "checkout" && call.Args[2] == "--detach" && call.Args[3] == "--force" && call.Args[4] == "FETCH_HEAD" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected checkout --detach --force FETCH_HEAD, not found")
	}
}

// ---------------------------------------------------------------------------
// B8: Explicit --branch should not be renamed on collision
// ---------------------------------------------------------------------------

func TestRunCheckout_ExplicitBranchNotRenamed(t *testing.T) {
	// User explicitly sets --branch main when base ref is also "main".
	// The branch name should NOT be prefixed with owner (respect explicit --branch).
	body := `{
		"head": {"ref": "main", "sha": "abc123def456abc123def456abc123def456abc1",
			"repo": {"full_name": "forker/repo"}, "user": {"login": "forker"}},
		"base": {"ref": "main", "sha": "0000000000000000000000000000000000000000",
			"repo": {"full_name": "owner/repo"}},
		"html_url": "https://atomgit.com/owner/repo/pull/42"
	}`

	results := map[string]mockResult{
		"status":                    {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/main": {exitCode: 128, stdout: ""},
		"remote:-v":                 {exitCode: 0, stdout: "origin\thttps://atomgit.com/owner/repo.git (fetch)\n"},
		"fetch":                     {exitCode: 0, stdout: ""},
		"checkout":                  {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, body), nil
		}),
	})

	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{Branch: "main"}, io.Discard)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Verify checkout used "main" (not "forker/main").
	for _, call := range *calls {
		if call.Args[1] == "checkout" && call.Args[3] == "forker/main" {
			t.Fatal("explicit --branch should not be renamed to forker/main")
		}
	}

	foundMain := false
	for _, call := range *calls {
		if call.Args[1] == "checkout" && call.Args[3] == "main" {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Fatal("expected checkout with branch 'main' (not renamed)")
	}
}

// ---------------------------------------------------------------------------
// B9: SHA comparison case-insensitive
// ---------------------------------------------------------------------------

func TestRunCheckout_SHAEqualFold(t *testing.T) {
	// Local branch exists at same SHA but API returns uppercase SHA.
	// Should detect the match case-insensitively and reuse the branch.
	results := map[string]mockResult{
		"status":                       {exitCode: 0, stdout: ""},
		"rev-parse:refs/heads/feature": {exitCode: 0, stdout: "ABC123DEF456ABC123DEF456ABC123DEF456ABC1"},
		"checkout":                     {exitCode: 0, stdout: ""},
	}
	gitClient, calls := newTestGitClient(t, results)

	apiClient := api.NewClientWithHTTPClient("token", &http.Client{
		Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return prResponse(200, sameRepoPR()), nil // SHA is lowercase
		}),
	})

	var stdout strings.Builder
	err := runCheckout(context.Background(), gitClient, apiClient, "owner", "repo", "42",
		checkoutOptions{}, &stdout)
	if err != nil {
		t.Fatalf("runCheckout() error = %v", err)
	}

	// Should print "already exists" (case-insensitive SHA match).
	if !strings.Contains(stdout.String(), "already exists") {
		t.Fatalf("expected 'already exists' for case-insensitive SHA match, got: %q", stdout.String())
	}

	// Should NOT call fetch (reuses branch).
	for _, call := range *calls {
		if call.Args[1] == "fetch" {
			t.Fatal("unexpected fetch for existing branch at matching SHA (case-insensitive)")
		}
	}
}

// ---------------------------------------------------------------------------
// B10: uniqueRemoteName
// ---------------------------------------------------------------------------

func TestUniqueRemoteName(t *testing.T) {
	tests := []struct {
		name     string
		remotes  []git.Remote
		number   string
		wantName string
	}{
		{
			name:     "no existing remotes",
			remotes:  nil,
			number:   "42",
			wantName: "pr-42",
		},
		{
			name: "no conflict",
			remotes: []git.Remote{
				{Name: "origin", URL: "https://atomgit.com/owner/repo.git"},
			},
			number:   "42",
			wantName: "pr-42",
		},
		{
			name: "name conflict",
			remotes: []git.Remote{
				{Name: "pr-42", URL: "https://atomgit.com/unrelated/repo.git"},
			},
			number:   "42",
			wantName: "pr-42-1",
		},
		{
			name: "multiple conflicts",
			remotes: []git.Remote{
				{Name: "pr-42", URL: "https://atomgit.com/a/b.git"},
				{Name: "pr-42-1", URL: "https://atomgit.com/c/d.git"},
			},
			number:   "42",
			wantName: "pr-42-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueRemoteName(tt.remotes, tt.number)
			if got != tt.wantName {
				t.Fatalf("uniqueRemoteName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// B11: parseRemoteURL was removed in favour of cmdutil.ResolveRepository.
// Host/port/case coverage is preserved by TestFindRemoteByURL_* above.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// B12: parseMirrorRemoteURL / inferRepoFromMirrorRemote — gitcode.com
// mirror fallback for cmdutil's atomgit-only resolver (upstream PR #68).
// ---------------------------------------------------------------------------

func TestParseMirrorRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{
			name:      "https atomgit",
			url:       "https://atomgit.com/hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "https gitcode mirror",
			url:       "https://gitcode.com/hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "https gitcode mirror without .git",
			url:       "https://gitcode.com/hust/atomgit-cli",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "scp atomgit",
			url:       "git@atomgit.com:hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "scp gitcode mirror",
			url:       "git@gitcode.com:hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "https gitcode with port",
			url:       "https://gitcode.com:443/hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:      "case-insensitive host",
			url:       "https://GitCode.COM/hust/atomgit-cli.git",
			wantOwner: "hust", wantRepo: "atomgit-cli", wantOK: true,
		},
		{
			name:   "unsupported host github",
			url:    "https://github.com/hust/atomgit-cli.git",
			wantOK: false,
		},
		{
			name:   "malformed url",
			url:    "not-a-url",
			wantOK: false,
		},
		{
			name:   "atomgit but missing repo segment",
			url:    "https://atomgit.com/hust",
			wantOK: false,
		},
		{
			name:   "atomgit but empty owner",
			url:    "https://atomgit.com//repo.git",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := parseMirrorRemoteURL(tt.url)
			if ok != tt.wantOK {
				t.Fatalf("parseMirrorRemoteURL(%q) ok = %v, want %v", tt.url, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("parseMirrorRemoteURL(%q) = %q/%q, want %q/%q",
					tt.url, owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestInferRepoFromMirrorRemote(t *testing.T) {
	tests := []struct {
		name       string
		results    map[string]mockResult
		wantOwner  string
		wantRepo   string
		wantErrSub string
	}{
		{
			name: "only gitcode remote via step 4 unique",
			results: map[string]mockResult{
				"remote:-v": {stdout: "origin\thttps://gitcode.com/hust/atomgit-cli.git (fetch)\n"},
			},
			wantOwner: "hust", wantRepo: "atomgit-cli",
		},
		{
			name: "gitcode scp form via step 4 unique",
			results: map[string]mockResult{
				"remote:-v": {stdout: "origin\tgit@gitcode.com:hust/atomgit-cli.git (fetch)\n"},
			},
			wantOwner: "hust", wantRepo: "atomgit-cli",
		},
		{
			name: "github coexists with gitcode picks gitcode via step 3 origin",
			results: map[string]mockResult{
				"remote:-v": {stdout: "github\thttps://github.com/hust/atomgit-cli.git (fetch)\n" +
					"origin\thttps://gitcode.com/hust/atomgit-cli.git (fetch)\n"},
			},
			wantOwner: "hust", wantRepo: "atomgit-cli",
		},
		{
			name: "step 1: remote.pushDefault selects mirror",
			results: map[string]mockResult{
				"remote:-v": {stdout: "alpha\thttps://gitcode.com/owner/one.git (fetch)\n" +
					"beta\thttps://gitcode.com/owner/two.git (fetch)\n"},
				"config:remote.pushDefault": {stdout: "beta\n"},
			},
			wantOwner: "owner", wantRepo: "two",
		},
		{
			name: "step 2: branch upstream selects mirror",
			results: map[string]mockResult{
				"remote:-v": {stdout: "alpha\thttps://gitcode.com/owner/one.git (fetch)\n" +
					"beta\thttps://gitcode.com/owner/two.git (fetch)\n"},
				"config:remote.pushDefault": {exitCode: 1}, // unset
				"symbolic-ref":              {stdout: "main\n"},
				"config:branch.main.remote": {stdout: "alpha\n"},
			},
			wantOwner: "owner", wantRepo: "one",
		},
		{
			name: "step 3: origin wins when it is a mirror",
			results: map[string]mockResult{
				"remote:-v": {stdout: "alpha\thttps://gitcode.com/owner/one.git (fetch)\n" +
					"origin\thttps://gitcode.com/owner/two.git (fetch)\n"},
				"config:remote.pushDefault": {exitCode: 1},
				"symbolic-ref":              {exitCode: 1},
			},
			wantOwner: "owner", wantRepo: "two",
		},
		{
			name: "step 4 unique: two remotes same owner/repo",
			results: map[string]mockResult{
				"remote:-v": {stdout: "alpha\thttps://gitcode.com/hust/atomgit-cli.git (fetch)\n" +
					"beta\tgit@gitcode.com:hust/atomgit-cli.git (fetch)\n"},
				"config:remote.pushDefault": {exitCode: 1},
				"symbolic-ref":              {exitCode: 1},
			},
			wantOwner: "hust", wantRepo: "atomgit-cli",
		},
		{
			name: "conflict: two mirror remotes different owner/repo",
			results: map[string]mockResult{
				"remote:-v": {stdout: "alpha\thttps://gitcode.com/owner/one.git (fetch)\n" +
					"beta\thttps://gitcode.com/owner/two.git (fetch)\n"},
				"config:remote.pushDefault": {exitCode: 1},
				"symbolic-ref":              {exitCode: 1},
			},
			wantErrSub: "AtomGit mirror remotes conflict",
		},
		{
			name: "no supported host",
			results: map[string]mockResult{
				"remote:-v": {stdout: "origin\thttps://github.com/hust/atomgit-cli.git (fetch)\n"},
			},
			wantErrSub: "no AtomGit remote found",
		},
		{
			name: "no remotes at all",
			results: map[string]mockResult{
				"remote:-v": {stdout: ""},
			},
			wantErrSub: "no AtomGit remote found",
		},
		{
			name: "pushDefault names a non-mirror remote falls through",
			results: map[string]mockResult{
				"remote:-v": {stdout: "github\thttps://github.com/hust/atomgit-cli.git (fetch)\n" +
					"origin\thttps://gitcode.com/hust/atomgit-cli.git (fetch)\n"},
				"config:remote.pushDefault": {stdout: "github\n"},
				"symbolic-ref":              {exitCode: 1},
			},
			wantOwner: "hust", wantRepo: "atomgit-cli",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitClient, _ := newTestGitClient(t, tt.results)

			owner, repo, err := inferRepoFromMirrorRemote(context.Background(), gitClient)
			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("got %q/%q, want %q/%q", owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestIsNoAtomGitRemoteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{
			name: "exact sentinel from cmdutil",
			err:  fmt.Errorf("unable to determine repository: no AtomGit remote found; pass owner/repo explicitly"),
			want: true,
		},
		{
			name: "conflict must not trigger fallback",
			err:  fmt.Errorf("unable to determine repository: AtomGit remotes conflict (alpha, beta); pass owner/repo explicitly or configure remote.pushDefault"),
			want: false,
		},
		{
			name: "not-a-repo must not trigger fallback",
			err:  fmt.Errorf("unable to determine repository: current directory is not a Git repository; pass owner/repo explicitly"),
			want: false,
		},
		{
			name: "invalid URL must not trigger fallback",
			err:  fmt.Errorf("unable to determine repository: invalid AtomGit remote URL for origin; pass owner/repo explicitly"),
			want: false,
		},
		{
			name: "unrelated error must not trigger fallback",
			err:  fmt.Errorf("expected 1 or 2 arguments, got 3"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNoAtomGitRemoteError(tt.err); got != tt.want {
				t.Fatalf("isNoAtomGitRemoteError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
