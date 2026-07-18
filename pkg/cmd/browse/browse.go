package browse

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdBrowse(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		branch    string
		commit    string
		repo      string
		releases  bool
		noBrowser bool
	}

	cmd := &cobra.Command{
		Use:   "browse [<number> | <path> | <commit-sha>]",
		Short: "Open repositories, issues, pull requests, and more in the browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := resolveRepo(f, opts.repo)
			if err != nil {
				return err
			}

			var targetURL string

			if opts.releases {
				targetURL = browser.BuildReleasesURL(owner, repo)
			} else if opts.commit != "" && len(args) == 0 {
				targetURL = browser.BuildCommitURL(owner, repo, opts.commit)
			} else if len(args) == 0 {
				targetURL = browser.BuildRepoURL(owner, repo)
			} else {
				arg := args[0]
				switch classifyArg(arg) {
				case argTypeNumber:
					num, err := strconv.Atoi(arg)
					if err != nil {
						return fmt.Errorf("invalid number: %s", arg)
					}
					token, err := f.Config.GetToken()
					if err != nil {
						return fmt.Errorf("not authenticated: %w", err)
					}
					var httpClient *http.Client
					if f.HttpClient != nil {
						httpClient, err = f.HttpClient()
						if err != nil {
							return fmt.Errorf("failed to create HTTP client: %w", err)
						}
					}
					client := api.NewClientWithHTTPClient(token, httpClient)
					targetURL, err = resolveNumber(client, owner, repo, num)
					if err != nil {
						return err
					}
				case argTypeCommit:
					targetURL = browser.BuildCommitURL(owner, repo, arg)
				default:
					filePath, lineStart, lineEnd := parseFilePathArg(arg)

					// Resolve branch lazily — only needed for file path without -c
					commitRef := opts.branch
					if commitRef == "" && opts.commit == "" {
						if b, err := resolveDefaultBranch(f, owner, repo); err == nil {
							commitRef = b
						} else {
							commitRef = "main"
						}
					}
					if opts.commit != "" {
						commitRef = opts.commit
					}
					targetURL = browser.BuildFileURL(owner, repo, commitRef, filePath, lineStart, lineEnd)
				}
			}

			if opts.noBrowser {
				fmt.Fprintln(cmd.OutOrStdout(), targetURL)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", targetURL)

			if f.BrowserOpener == nil {
				return nil
			}

			if err := f.BrowserOpener(targetURL); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Select another branch by passing in the branch name")
	cmd.Flags().StringVarP(&opts.commit, "commit", "c", "", "Select another commit by passing in the commit SHA, default is the last commit")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Select another repository using the OWNER/REPO format")
	cmd.Flags().BoolVarP(&opts.releases, "releases", "r", false, "Open repository releases")
	cmd.Flags().BoolVarP(&opts.noBrowser, "no-browser", "n", false, "Print destination URL instead of opening the browser")

	return cmd
}

type argType int

const (
	argTypePath argType = iota
	argTypeNumber
	argTypeCommit
)

var digitsRe = regexp.MustCompile(`^\d+$`)
var hexHashRe = regexp.MustCompile(`^[0-9a-fA-F]{6,40}$`)

func classifyArg(arg string) argType {
	if digitsRe.MatchString(arg) {
		return argTypeNumber
	}
	if hexHashRe.MatchString(arg) {
		return argTypeCommit
	}
	return argTypePath
}

func parseFilePathArg(arg string) (path string, lineStart, lineEnd int) {
	idx := strings.LastIndex(arg, ":")
	if idx < 0 {
		return arg, 0, 0
	}
	path = arg[:idx]
	linePart := arg[idx+1:]

	linePart = strings.Replace(linePart, "..", "-", 1)
	parts := strings.SplitN(linePart, "-", 2)

	if start, err := strconv.Atoi(parts[0]); err == nil {
		lineStart = start
	}
	if len(parts) == 2 {
		if end, err := strconv.Atoi(parts[1]); err == nil {
			lineEnd = end
		}
	}
	return path, lineStart, lineEnd
}

func resolveRepo(f *cmdutil.Factory, repoFlag string) (owner, repo string, err error) {
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid --repo format: %s (expected OWNER/REPO)", repoFlag)
		}
		return parts[0], parts[1], nil
	}

	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("no repository specified; run inside a git repo or use --repo")
	}

	return browser.ParseRemoteURL(string(remote))
}

func resolveDefaultBranch(f *cmdutil.Factory, owner, repo string) (string, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return "", err
	}
	var httpClient *http.Client
	if f.HttpClient != nil {
		httpClient, err = f.HttpClient()
		if err != nil {
			return "", err
		}
	}
	client := api.NewClientWithHTTPClient(token, httpClient)
	var repoInfo api.Repository
	if err := client.Get(fmt.Sprintf("/repos/%s/%s", owner, repo), &repoInfo); err != nil {
		return "", err
	}
	if repoInfo.DefaultBranch == "" {
		return "", fmt.Errorf("empty default branch for %s/%s", owner, repo)
	}
	return repoInfo.DefaultBranch, nil
}

func resolveNumber(client *api.Client, owner, repo string, num int) (string, error) {
	issuePath := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, num)
	resp, err := client.DoRequestRaw(http.MethodGet, issuePath)
	if err != nil {
		return "", fmt.Errorf("failed to check issue #%d: %w", num, err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return browser.BuildIssueURL(owner, repo, num), nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return "", fmt.Errorf("unexpected status checking issue #%d: %s", num, resp.Status)
	}

	prPath := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, num)
	resp, err = client.DoRequestRaw(http.MethodGet, prPath)
	if err != nil {
		return "", fmt.Errorf("failed to check PR #%d: %w", num, err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return browser.BuildPRURL(owner, repo, num), nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return "", fmt.Errorf("unexpected status checking PR #%d: %s", num, resp.Status)
	}

	return "", fmt.Errorf("no issue or pull request with number %d found in %s/%s", num, owner, repo)
}
