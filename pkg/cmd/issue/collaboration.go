package issue

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type linkedPRJSON struct {
	ID        int64  `json:"id"`
	Number    string `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func newLinkedPRJSON(pr api.IssueLinkedPullRequest) linkedPRJSON {
	return linkedPRJSON{
		ID:        pr.ID,
		Number:    pr.GetNumber(),
		Title:     pr.Title,
		Body:      pr.Body,
		State:     pr.State,
		URL:       pr.HTMLURL,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
	}
}

func newCmdIssuePRS(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		JSON bool
	}

	cmd := &cobra.Command{
		Use:   "prs [<owner>/<repo>] <number>",
		Short: "List pull requests linked to an issue",
		Long: `List pull requests linked to an issue.

Concurrent updates to linked pull requests are not reflected until the next
request.`,
		Example: `  ag issue prs owner/repo 42
  ag issue prs owner/repo 42 --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number := remaining[0]
			if !isPositiveIssueNumber(number) {
				return fmt.Errorf("invalid issue number %q (must be a positive integer)", number)
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			prs, err := api.ListIssueLinkedPullRequests(client, repository.Owner, repository.Name, number)
			if err != nil {
				return err
			}

			if opts.JSON {
				result := make([]linkedPRJSON, len(prs))
				for i, pr := range prs {
					result[i] = newLinkedPRJSON(pr)
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, pr := range prs {
				fmt.Fprintf(out, "#%s %s [%s]\n", pr.GetNumber(), pr.Title, pr.State)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output linked pull requests as JSON")
	return cmd
}

func newCmdIssueBranches(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Add    []string
		Remove []string
		Yes    bool
		JSON   bool
	}

	cmd := &cobra.Command{
		Use:   "branches [<owner>/<repo>] <number>",
		Short: "List or update related branches for an issue",
		Long: `List or update related branches for an issue.

With neither --add nor --remove, the command lists current related branch names.

Concurrent branch-name updates from other clients can be overwritten by this
command because the AtomGit API uses whole-list replacement. Review the current
list before mutating branches that other tools may also manage.`,
		Example: `  ag issue branches owner/repo 42
  ag issue branches owner/repo 42 --json
  ag issue branches owner/repo 42 --add feature/x
  ag issue branches owner/repo 42 --remove main --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number := remaining[0]
			if !isPositiveIssueNumber(number) {
				return fmt.Errorf("invalid issue number %q (must be a positive integer)", number)
			}

			isMutation := len(opts.Add) > 0 || len(opts.Remove) > 0

			if isMutation {
				addNames, removeNames, validateErr := validateBranchNames(opts.Add, opts.Remove)
				if validateErr != nil {
					return validateErr
				}
				opts.Add = addNames
				opts.Remove = removeNames
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			if isMutation {
				hasRemoval := false
				for _, name := range opts.Remove {
					if name != "" {
						hasRemoval = true
						break
					}
				}
				if hasRemoval && !opts.Yes {
					action := "remove " + strings.Join(opts.Remove, ", ") + " from"
					if len(opts.Add) > 0 {
						action = "add " + strings.Join(opts.Add, ", ") + " and " + action
					}
					prompt := fmt.Sprintf("Change related branches on issue #%s in %s/%s: %s? (y/N) ", number, repository.Owner, repository.Name, action)
					confirmed, promptErr := confirmPrompt(cmd, prompt)
					if promptErr != nil {
						return promptErr
					}
					if !confirmed {
						return nil
					}
				}
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			current, err := api.ListIssueRelatedBranches(client, repository.Owner, repository.Name, number)
			if err != nil {
				return err
			}

			if !isMutation {
				if opts.JSON {
					return cmdutil.WriteJSON(cmd.OutOrStdout(), current)
				}
				out := cmd.OutOrStdout()
				for _, branch := range current {
					fmt.Fprintln(out, branch)
				}
				return nil
			}

			desired := computeDesiredBranches(current, opts.Add, opts.Remove)

			if branchesEqual(current, desired) {
				if opts.JSON {
					return cmdutil.WriteJSON(cmd.OutOrStdout(), desired)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "No changes needed for issue #%s (already has %d branch(es))\n", number, len(desired))
				return nil
			}

			if err := api.UpdateIssueRelatedBranches(client, repository.Owner, repository.Name, number, desired); err != nil {
				return err
			}

			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), desired)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated issue #%s: %d branch(es)\n", number, len(desired))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&opts.Add, "add", nil, "Branch names to add (repeatable)")
	cmd.Flags().StringArrayVar(&opts.Remove, "remove", nil, "Branch names to remove (repeatable)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output branch names as JSON")
	return cmd
}

func isPositiveIssueNumber(number string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(number))
	return err == nil && n > 0
}

func validateBranchNames(addRaw, removeRaw []string) ([]string, []string, error) {
	add := make([]string, 0, len(addRaw))
	addSeen := make(map[string]struct{})
	for _, name := range addRaw {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return nil, nil, fmt.Errorf("branch name cannot be empty")
		}
		if _, exists := addSeen[trimmed]; exists {
			return nil, nil, fmt.Errorf("duplicate branch name in --add: %q", trimmed)
		}
		addSeen[trimmed] = struct{}{}
		add = append(add, trimmed)
	}

	remove := make([]string, 0, len(removeRaw))
	removeSeen := make(map[string]struct{})
	for _, name := range removeRaw {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return nil, nil, fmt.Errorf("branch name cannot be empty")
		}
		if _, exists := removeSeen[trimmed]; exists {
			return nil, nil, fmt.Errorf("duplicate branch name in --remove: %q", trimmed)
		}
		removeSeen[trimmed] = struct{}{}
		remove = append(remove, trimmed)
	}

	for _, name := range add {
		if _, exists := removeSeen[name]; exists {
			return nil, nil, fmt.Errorf("branch name %q cannot be in both --add and --remove", name)
		}
	}

	return add, remove, nil
}

// computeDesiredBranches preserves existing server order for unremoved branches
// and appends new additions in flag order.
func computeDesiredBranches(current, add, remove []string) []string {
	removeSet := make(map[string]struct{}, len(remove))
	for _, name := range remove {
		removeSet[name] = struct{}{}
	}

	desired := make([]string, 0, len(current)+len(add))
	for _, name := range current {
		if _, ok := removeSet[name]; !ok {
			desired = append(desired, name)
		}
	}

	addSet := make(map[string]struct{}, len(current))
	for _, name := range current {
		addSet[name] = struct{}{}
	}
	for _, name := range add {
		if _, exists := addSet[name]; !exists {
			desired = append(desired, name)
		}
	}

	return desired
}

func branchesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
