package branch

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type protectionSetOptions struct {
	Push  string
	Merge string
	Yes   bool
}

func newCmdBranchProtection(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protection",
		Short: "Manage protected branch rules",
		Long: `List, view, create, update, and delete protected branch rules.

Rules may name an exact branch or contain an AtomGit wildcard pattern. Exact
rules take precedence over matching wildcard rules. AtomGit's API exposes only
push and merge allowlists; other web settings are not changed by this command.`,
	}
	cmd.AddCommand(newCmdProtectionList(f))
	cmd.AddCommand(newCmdProtectionView(f))
	cmd.AddCommand(newCmdProtectionSet(f))
	cmd.AddCommand(newCmdProtectionDelete(f))
	return cmd
}

func newCmdProtectionList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list <owner>/<repo>",
		Short: "List protected branch rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			rules, err := listProtectionRules(client, repository)
			if err != nil {
				return fmt.Errorf("failed to list protected branch rules for %s/%s: %w", repository.Owner, repository.Repo, err)
			}
			for _, rule := range rules {
				printProtectionSummary(cmd.OutOrStdout(), rule)
			}
			return nil
		},
	}
}

func newCmdProtectionView(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view <owner>/<repo> <branch-or-pattern>",
		Short: "View a protected branch rule",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}
			pattern, err := validateProtectionPattern(args[1])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			rules, err := listProtectionRules(client, repository)
			if err != nil {
				return fmt.Errorf("failed to read protected branch rules for %s/%s: %w", repository.Owner, repository.Repo, err)
			}
			rule, found := findProtectionRule(rules, pattern)
			if !found {
				return fmt.Errorf("protected branch rule %q was not found in %s/%s", pattern, repository.Owner, repository.Repo)
			}
			printProtectionDetail(cmd.OutOrStdout(), rule)
			return nil
		},
	}
}

func newCmdProtectionSet(f *cmdutil.Factory) *cobra.Command {
	opts := &protectionSetOptions{}
	cmd := &cobra.Command{
		Use:   "set <owner>/<repo> <branch-or-pattern>",
		Short: "Create or update a protected branch rule",
		Long: `Create or update a protected branch rule.

Permission values are semicolon-separated role names or usernames. Supported
roles are develop, admin, and maintainer. An explicitly empty value denies the
operation to everyone. Existing rules preserve any permission whose flag is
omitted; new rules require both --push and --merge. Updating an existing rule
requires confirmation unless --yes is supplied.`,
		Example: `  ag branch protection set owner/repo main --push admin --merge admin
  ag branch protection set owner/repo main --push maintainer --merge admin
  ag branch protection set owner/repo "release/*" --push "develop;alice" --merge admin
  ag branch protection set owner/repo main --push "" --yes`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}
			pattern, err := validateProtectionPattern(args[1])
			if err != nil {
				return err
			}
			pushChanged := cmd.Flags().Changed("push")
			mergeChanged := cmd.Flags().Changed("merge")
			if !pushChanged && !mergeChanged {
				return fmt.Errorf("at least one of --push or --merge must be provided")
			}
			if pushChanged {
				if err := validateProtectionPermission(opts.Push); err != nil {
					return fmt.Errorf("invalid --push value: %w", err)
				}
			}
			if mergeChanged {
				if err := validateProtectionPermission(opts.Merge); err != nil {
					return fmt.Errorf("invalid --merge value: %w", err)
				}
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			rules, err := listProtectionRules(client, repository)
			if err != nil {
				return fmt.Errorf("failed to read protected branch rules for %s/%s: %w", repository.Owner, repository.Repo, err)
			}
			existing, found := findProtectionRule(rules, pattern)
			request := api.ProtectedBranchRequest{Wildcard: pattern}
			if found {
				if !pushChanged {
					request.Pusher, err = protectionPermissionValue(existing, true)
					if err != nil {
						return fmt.Errorf("cannot preserve push permissions for %q: %w", pattern, err)
					}
				}
				if !mergeChanged {
					request.Merger, err = protectionPermissionValue(existing, false)
					if err != nil {
						return fmt.Errorf("cannot preserve merge permissions for %q: %w", pattern, err)
					}
				}
			} else if !pushChanged || !mergeChanged {
				return fmt.Errorf("new protected branch rules require both --push and --merge")
			}
			if pushChanged {
				request.Pusher = opts.Push
			}
			if mergeChanged {
				request.Merger = opts.Merge
			}

			if found && !opts.Yes {
				printProtectionDetail(cmd.OutOrStdout(), existing)
				fmt.Fprintf(cmd.OutOrStdout(), "New Push: %s\n", protectionValueDisplay(request.Pusher))
				fmt.Fprintf(cmd.OutOrStdout(), "New Merge: %s\n", protectionValueDisplay(request.Merger))
				confirmed, err := confirmProtectionChange(cmd.InOrStdin(), cmd.OutOrStdout(), "Update", pattern)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Protected branch update cancelled")
					return nil
				}
			}

			if found {
				request.Wildcard = ""
				if err := client.Put(protectionRulePath(repository, pattern), request, nil); err != nil {
					return fmt.Errorf("failed to update protected branch rule %q: %w", pattern, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Updated protected branch rule %s\n", pattern)
				return nil
			}
			if err := client.Put(protectionRulesPath(repository)+"/setting/new", request, nil); err != nil {
				return fmt.Errorf("failed to create protected branch rule %q: %w", pattern, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created protected branch rule %s\n", pattern)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Push, "push", "", "Push allowlist: develop, admin, maintainer, usernames, or empty")
	cmd.Flags().StringVar(&opts.Merge, "merge", "", "Merge allowlist: develop, admin, maintainer, usernames, or empty")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation when updating an existing rule")
	return cmd
}

func newCmdProtectionDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <owner>/<repo> <branch-or-pattern>",
		Short: "Delete a protected branch rule",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parseRepositoryArg(args[0])
			if err != nil {
				return err
			}
			pattern, err := validateProtectionPattern(args[1])
			if err != nil {
				return err
			}
			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			rules, err := listProtectionRules(client, repository)
			if err != nil {
				return fmt.Errorf("failed to read protected branch rules for %s/%s: %w", repository.Owner, repository.Repo, err)
			}
			rule, found := findProtectionRule(rules, pattern)
			if !found {
				return fmt.Errorf("protected branch rule %q was not found in %s/%s", pattern, repository.Owner, repository.Repo)
			}
			if !yes {
				printProtectionDetail(cmd.OutOrStdout(), rule)
				confirmed, err := confirmProtectionChange(cmd.InOrStdin(), cmd.OutOrStdout(), "Delete", pattern)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Protected branch deletion cancelled")
					return nil
				}
			}
			if err := client.Delete(protectionRulePath(repository, pattern)); err != nil {
				return fmt.Errorf("failed to delete protected branch rule %q: %w", pattern, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted protected branch rule %s\n", pattern)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip deletion confirmation")
	return cmd
}

func protectionRulesPath(repository repositoryRef) string {
	return fmt.Sprintf("/repos/%s/%s/branches", url.PathEscape(repository.Owner), url.PathEscape(repository.Repo))
}

func protectionRulePath(repository repositoryRef, pattern string) string {
	return protectionRulesPath(repository) + "/" + url.PathEscape(pattern) + "/setting"
}

func listProtectionRules(client *api.Client, repository repositoryRef) ([]api.ProtectedBranchRule, error) {
	var rules []api.ProtectedBranchRule
	path := fmt.Sprintf("/repos/%s/%s/protect_branches", url.PathEscape(repository.Owner), url.PathEscape(repository.Repo))
	if err := client.Get(path, &rules); err != nil {
		return nil, err
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules, nil
}

func findProtectionRule(rules []api.ProtectedBranchRule, pattern string) (api.ProtectedBranchRule, bool) {
	for _, rule := range rules {
		if rule.Name == pattern {
			return rule, true
		}
	}
	return api.ProtectedBranchRule{}, false
}

func validateProtectionPattern(value string) (string, error) {
	pattern := strings.TrimSpace(value)
	if pattern == "" {
		return "", fmt.Errorf("branch or wildcard pattern is required")
	}
	if pattern != value || strings.HasPrefix(pattern, "/") || strings.HasSuffix(pattern, "/") || strings.Contains(pattern, "//") {
		return "", fmt.Errorf("invalid branch or wildcard pattern %q", value)
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("invalid branch or wildcard pattern %q", value)
		}
	}
	if strings.Contains(pattern, "..") || strings.Contains(pattern, "@{") || strings.HasSuffix(pattern, ".") {
		return "", fmt.Errorf("invalid branch or wildcard pattern %q", value)
	}
	for _, r := range pattern {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '\\' || r == ':' || r == '~' || r == '^' {
			return "", fmt.Errorf("invalid branch or wildcard pattern %q", value)
		}
	}
	if _, err := path.Match(pattern, pattern); err != nil {
		return "", fmt.Errorf("invalid branch or wildcard pattern %q: %w", value, err)
	}
	return pattern, nil
}

func validateProtectionPermission(value string) error {
	if value == "" {
		return nil
	}
	seen := make(map[string]struct{})
	for _, token := range strings.Split(value, ";") {
		if token == "" || token != strings.TrimSpace(token) {
			return fmt.Errorf("permissions must be non-empty semicolon-separated values without spaces")
		}
		for _, r := range token {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.') {
				return fmt.Errorf("unsupported role or username %q", token)
			}
		}
		if _, exists := seen[token]; exists {
			return fmt.Errorf("duplicate role or username %q", token)
		}
		seen[token] = struct{}{}
	}
	return nil
}

func protectionPermissionValue(rule api.ProtectedBranchRule, push bool) (string, error) {
	var noOne, developers, committer, master, maintainer, owner bool
	users := rule.MergeUsers
	if push {
		noOne = rule.NoOneCanPush.Bool()
		developers = rule.DevelopersCanPush.Bool()
		committer = rule.CommitterCanPush.Bool()
		master = rule.MasterCanPush.Bool()
		maintainer = rule.MaintainerCanPush.Bool()
		owner = rule.OwnerCanPush.Bool()
		users = rule.PushUsers
	} else {
		noOne = rule.NoOneCanMerge.Bool()
		developers = rule.DevelopersCanMerge.Bool()
		committer = rule.CommitterCanMerge.Bool()
		master = rule.MasterCanMerge.Bool()
		maintainer = rule.MaintainerCanMerge.Bool()
		owner = rule.OwnerCanMerge.Bool()
	}
	if noOne {
		return "", nil
	}

	values := make([]string, 0, len(users)+1)
	switch {
	case developers:
		values = append(values, "develop")
	case master:
		values = append(values, "admin")
	case maintainer:
		values = append(values, "maintainer")
	case committer:
		return "", fmt.Errorf("the API response contains unsupported committer-only access; specify this permission explicitly")
	}
	for _, user := range users {
		name := strings.TrimSpace(user.Username)
		if name == "" {
			name = strings.TrimSpace(user.Login)
		}
		if name == "" {
			return "", fmt.Errorf("the API response contains a user without a username")
		}
		values = append(values, name)
	}
	if len(values) == 0 && owner {
		return "", fmt.Errorf("the API response contains unsupported owner-only access; specify this permission explicitly")
	}
	return strings.Join(values, ";"), nil
}

func protectionRuleKind(name string) string {
	if strings.ContainsAny(name, "*?[") {
		return "wildcard"
	}
	return "exact"
}

func permissionDisplay(rule api.ProtectedBranchRule, push bool) string {
	value, err := protectionPermissionValue(rule, push)
	if err != nil {
		return "unsupported"
	}
	return protectionValueDisplay(value)
}

func protectionValueDisplay(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

func printProtectionSummary(out io.Writer, rule api.ProtectedBranchRule) {
	fmt.Fprintf(out, "%s type:%s push:%s merge:%s", rule.Name, protectionRuleKind(rule.Name), permissionDisplay(rule, true), permissionDisplay(rule, false))
	if rule.UpdatedAt != "" {
		fmt.Fprintf(out, " updated:%s", rule.UpdatedAt)
	}
	fmt.Fprintln(out)
}

func printProtectionDetail(out io.Writer, rule api.ProtectedBranchRule) {
	fmt.Fprintf(out, "Rule: %s\n", rule.Name)
	fmt.Fprintf(out, "Type: %s\n", protectionRuleKind(rule.Name))
	fmt.Fprintf(out, "Push: %s\n", permissionDisplay(rule, true))
	fmt.Fprintf(out, "Merge: %s\n", permissionDisplay(rule, false))
	if rule.UpdatedAt != "" {
		fmt.Fprintf(out, "Updated: %s\n", rule.UpdatedAt)
	}
}

func confirmProtectionChange(in io.Reader, out io.Writer, action, pattern string) (bool, error) {
	fmt.Fprintf(out, "%s protected branch rule %s? [y/N] ", action, pattern)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
