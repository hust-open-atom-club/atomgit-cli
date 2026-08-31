package tag

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type tagProtectionSetOptions struct {
	CreateAccess string
	Yes          bool
}

type protectedTagJSON struct {
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	CreateAccess          string `json:"createAccess"`
	CreateAccessLevel     int    `json:"createAccessLevel"`
	CreateAccessLevelDesc string `json:"createAccessLevelDesc"`
}

func newCmdTagProtection(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protection",
		Short: "Manage protected tag rules",
		Long: `List, view, create, update, and delete protected tag rules.

Rules may name an exact tag or contain an AtomGit wildcard pattern. AtomGit's
API exposes only create/push access levels; other web settings are not changed
by this command.`,
	}
	cmd.AddCommand(newCmdTagProtectionList(f))
	cmd.AddCommand(newCmdTagProtectionView(f))
	cmd.AddCommand(newCmdTagProtectionSet(f))
	cmd.AddCommand(newCmdTagProtectionDelete(f))
	return cmd
}

func newCmdTagProtectionList(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	var limit int

	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List protected tag rules",
		Example: "  ag tag protection list owner/repo --limit 50",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := tagAPIClient(f)
			if err != nil {
				return err
			}
			rules, err := listProtectedTags(client, repository, limit)
			if err != nil {
				return fmt.Errorf("failed to list protected tag rules for %s: %w", repository, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), protectedTagsJSON(rules))
			}
			if len(rules) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No protected tag rules found")
				return nil
			}
			for _, rule := range rules {
				printProtectedTagSummary(cmd.OutOrStdout(), rule)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of protected tag rules to list")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output protected tag rules as JSON")
	return cmd
}

func newCmdTagProtectionView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <tag-or-pattern>",
		Short: "View a protected tag rule",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			name, err := validateProtectedTagName(remaining[0])
			if err != nil {
				return err
			}
			client, err := tagAPIClient(f)
			if err != nil {
				return err
			}
			rule, err := getProtectedTag(client, repository, name)
			if err != nil {
				if api.IsHTTPStatus(err, http.StatusNotFound) {
					return fmt.Errorf("protected tag rule %q was not found in %s", name, repository)
				}
				return fmt.Errorf("failed to view protected tag rule %q for %s: %w", name, repository, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newProtectedTagJSON(rule))
			}
			printProtectedTagDetail(cmd.OutOrStdout(), repository, rule)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the protected tag rule as JSON")
	return cmd
}

func newCmdTagProtectionSet(f *cmdutil.Factory) *cobra.Command {
	opts := &tagProtectionSetOptions{}
	cmd := &cobra.Command{
		Use:   "set [<owner>/<repo>] <tag-or-pattern>",
		Short: "Create or update a protected tag rule",
		Long: `Create or update a protected tag rule.

--create-access accepts none, developer, or maintainer. These map to AtomGit
create_access_level values 0, 30, and 40: nobody; Developer/Maintainer/Admin;
and Maintainer/Admin. New rules require --create-access. Existing rules keep
the current access level when the flag is omitted. Updating an existing rule
requires confirmation unless --yes is supplied.`,
		Example: `  ag tag protection set owner/repo v1.0.0 --create-access maintainer
  ag tag protection set owner/repo "v*" --create-access developer
  ag tag protection set owner/repo v1.0.0 --create-access none --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			name, err := validateProtectedTagName(remaining[0])
			if err != nil {
				return err
			}
			accessChanged := cmd.Flags().Changed("create-access")
			var accessLevel int
			if accessChanged {
				accessLevel, err = parseCreateAccess(opts.CreateAccess)
				if err != nil {
					return fmt.Errorf("invalid --create-access value: %w", err)
				}
			}

			client, err := tagAPIClient(f)
			if err != nil {
				return err
			}
			existing, err := getProtectedTag(client, repository, name)
			found := err == nil
			if err != nil && !api.IsHTTPStatus(err, http.StatusNotFound) {
				return fmt.Errorf("failed to read protected tag rule %q for %s: %w", name, repository, err)
			}
			if !found && !accessChanged {
				return fmt.Errorf("new protected tag rules require --create-access")
			}
			if found && !accessChanged {
				accessLevel, err = requireCreateAccessLevel(existing)
				if err != nil {
					return fmt.Errorf("cannot preserve create access for %q: %w", name, err)
				}
			}

			request := api.ProtectedTagRequest{Name: name, CreateAccessLevel: accessLevel}
			if found && !opts.Yes {
				printProtectedTagDetail(cmd.OutOrStdout(), repository, existing)
				fmt.Fprintf(cmd.OutOrStdout(), "New create access: %s\n", mustCreateAccessName(accessLevel))
				confirmed, err := confirmProtectedTagChange(cmd.InOrStdin(), cmd.ErrOrStderr(), "Update", name, repository)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Protected tag update cancelled")
					return nil
				}
			}

			if found {
				if err := client.Put(protectedTagsPath(repository), request, nil); err != nil {
					return fmt.Errorf("failed to update protected tag rule %q: %w", name, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Updated protected tag rule %s\n", name)
				return nil
			}
			if err := client.Post(protectedTagsPath(repository), request, nil); err != nil {
				return fmt.Errorf("failed to create protected tag rule %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created protected tag rule %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.CreateAccess, "create-access", "", "Create access: none, developer, or maintainer")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation when updating an existing rule")
	return cmd
}

func newCmdTagProtectionDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <tag-or-pattern>",
		Short: "Delete a protected tag rule",
		Long: `Delete a protected tag rule.

By default, the current repository and rule are shown and you will be prompted
to confirm. Use --yes to skip the confirmation prompt.`,
		Example: `  ag tag protection delete owner/repo "v*"
  ag tag protection delete owner/repo "v*" --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			name, err := validateProtectedTagName(remaining[0])
			if err != nil {
				return err
			}
			client, err := tagAPIClient(f)
			if err != nil {
				return err
			}
			rule, err := getProtectedTag(client, repository, name)
			if err != nil {
				if api.IsHTTPStatus(err, http.StatusNotFound) {
					return fmt.Errorf("protected tag rule %q was not found in %s", name, repository)
				}
				return fmt.Errorf("failed to read protected tag rule %q for %s: %w", name, repository, err)
			}
			if !yes {
				printProtectedTagDetail(cmd.OutOrStdout(), repository, rule)
				confirmed, err := confirmProtectedTagChange(cmd.InOrStdin(), cmd.ErrOrStderr(), "Delete", name, repository)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Protected tag deletion cancelled")
					return nil
				}
			}
			if err := client.Delete(protectedTagPath(repository, name)); err != nil {
				return fmt.Errorf("failed to delete protected tag rule %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted protected tag rule %s\n", name)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip deletion confirmation")
	return cmd
}

func tagAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return f.NewAPIClient(token)
}

func protectedTagsPath(repository cmdutil.Repository) string {
	return fmt.Sprintf("/repos/%s/%s/protected_tags", url.PathEscape(repository.Owner), url.PathEscape(repository.Name))
}

func protectedTagPath(repository cmdutil.Repository, name string) string {
	return protectedTagsPath(repository) + "/" + url.PathEscape(name)
}

func listProtectedTags(client *api.Client, repository cmdutil.Repository, limit int) ([]api.ProtectedTag, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	endpoint := protectedTagsPath(repository)
	rules, err := api.GetPaginated[api.ProtectedTag](client, limit, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", endpoint, page, perPage)
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules, nil
}

func getProtectedTag(client *api.Client, repository cmdutil.Repository, name string) (api.ProtectedTag, error) {
	var rule api.ProtectedTag
	if err := client.Get(protectedTagPath(repository, name), &rule); err != nil {
		return api.ProtectedTag{}, err
	}
	return rule, nil
}

func validateProtectedTagName(value string) (string, error) {
	pattern := strings.TrimSpace(value)
	if pattern == "" {
		return "", fmt.Errorf("tag or wildcard pattern is required")
	}
	if pattern != value || strings.HasPrefix(pattern, "/") || strings.HasSuffix(pattern, "/") || strings.Contains(pattern, "//") {
		return "", fmt.Errorf("invalid tag or wildcard pattern %q", value)
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("invalid tag or wildcard pattern %q", value)
		}
	}
	if strings.Contains(pattern, "..") || strings.Contains(pattern, "@{") || strings.HasSuffix(pattern, ".") {
		return "", fmt.Errorf("invalid tag or wildcard pattern %q", value)
	}
	for _, r := range pattern {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '\\' || r == ':' || r == '~' || r == '^' {
			return "", fmt.Errorf("invalid tag or wildcard pattern %q", value)
		}
	}
	if _, err := path.Match(pattern, pattern); err != nil {
		return "", fmt.Errorf("invalid tag or wildcard pattern %q: %w", value, err)
	}
	return pattern, nil
}

func parseCreateAccess(value string) (int, error) {
	switch value {
	case "none":
		return api.ProtectedTagCreateAccessNone, nil
	case "developer":
		return api.ProtectedTagCreateAccessDeveloper, nil
	case "maintainer":
		return api.ProtectedTagCreateAccessMaintainer, nil
	default:
		return 0, fmt.Errorf("must be none, developer, or maintainer")
	}
}

func createAccessName(level int) (string, error) {
	switch level {
	case api.ProtectedTagCreateAccessNone:
		return "none", nil
	case api.ProtectedTagCreateAccessDeveloper:
		return "developer", nil
	case api.ProtectedTagCreateAccessMaintainer:
		return "maintainer", nil
	default:
		return "", fmt.Errorf("unsupported create access level %d", level)
	}
}

func requireCreateAccessLevel(rule api.ProtectedTag) (int, error) {
	if _, err := createAccessName(rule.CreateAccessLevel); err != nil {
		return 0, err
	}
	return rule.CreateAccessLevel, nil
}

func mustCreateAccessName(level int) string {
	name, err := createAccessName(level)
	if err != nil {
		return "unsupported"
	}
	return name
}

func protectedTagKind(name string) string {
	if strings.ContainsAny(name, "*?[") {
		return "wildcard"
	}
	return "exact"
}

func newProtectedTagJSON(rule api.ProtectedTag) protectedTagJSON {
	return protectedTagJSON{
		Name:                  rule.Name,
		Type:                  protectedTagKind(rule.Name),
		CreateAccess:          mustCreateAccessName(rule.CreateAccessLevel),
		CreateAccessLevel:     rule.CreateAccessLevel,
		CreateAccessLevelDesc: rule.CreateAccessLevelDesc,
	}
}

func protectedTagsJSON(rules []api.ProtectedTag) []protectedTagJSON {
	result := make([]protectedTagJSON, len(rules))
	for i, rule := range rules {
		result[i] = newProtectedTagJSON(rule)
	}
	return result
}

func printProtectedTagSummary(out io.Writer, rule api.ProtectedTag) {
	fmt.Fprintf(out, "%s type:%s create-access:%s\n", rule.Name, protectedTagKind(rule.Name), mustCreateAccessName(rule.CreateAccessLevel))
}

func printProtectedTagDetail(out io.Writer, repository cmdutil.Repository, rule api.ProtectedTag) {
	fmt.Fprintf(out, "Repository: %s\n", repository)
	fmt.Fprintf(out, "Rule: %s\n", rule.Name)
	fmt.Fprintf(out, "Type: %s\n", protectedTagKind(rule.Name))
	fmt.Fprintf(out, "Create access: %s\n", mustCreateAccessName(rule.CreateAccessLevel))
}

func confirmProtectedTagChange(in io.Reader, out io.Writer, action, name string, repository cmdutil.Repository) (bool, error) {
	return cmdutil.Confirm(in, out, fmt.Sprintf("%s protected tag rule %s in %s? [y/N] ", action, name, repository))
}
