package org

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	defaultRunnerGroupLimit = 30
	maxRunnerGroupsPerPage  = 100
)

type runnerGroupListOptions struct {
	Limit int
	JSON  bool
}

func newCmdOrgRunnerGroup(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runner-group",
		Short: "Inspect organization Actions runner groups",
		Long:  "Inspect organization-level AtomGit Actions runner groups and their read-only associations.",
		Example: `  ag org runner-group list my-organization
  ag org runner-group view my-organization group-id
  ag org runner-group runners my-organization group-id --json`,
	}
	cmd.AddCommand(newCmdRunnerGroupList(f))
	cmd.AddCommand(newCmdRunnerGroupView(f))
	cmd.AddCommand(newCmdRunnerGroupRunners(f))
	cmd.AddCommand(newCmdRunnerGroupRunnerSets(f))
	cmd.AddCommand(newCmdRunnerGroupNamespaces(f))
	return cmd
}

func newCmdRunnerGroupList(f *cmdutil.Factory) *cobra.Command {
	opts := &runnerGroupListOptions{Limit: defaultRunnerGroupLimit}
	cmd := &cobra.Command{
		Use:   "list <org>",
		Short: "List organization runner groups",
		Example: `  ag org runner-group list my-organization
  ag org runner-group list my-organization --limit 100
  ag org runner-group list my-organization --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseRunnerGroupOrganization(args[0])
			if err != nil {
				return err
			}
			if err := validateRunnerGroupLimit(opts.Limit); err != nil {
				return err
			}
			client, err := organizationActionsClient(f)
			if err != nil {
				return err
			}
			groups, err := collectRunnerGroupPages(opts.Limit, func(group actions.RunnerGroup) string { return group.ID }, func(page, perPage int) (int, []actions.RunnerGroup, error) {
				response, err := client.ListOrganizationRunnerGroups(organization, actions.ListRunnerGroupsOptions{Page: page, PerPage: perPage})
				return response.TotalCount, response.RunnerGroups, err
			})
			if err != nil {
				return fmt.Errorf("failed to list runner groups for organization %q: %w", organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), actions.RunnerGroupListResponse{TotalCount: len(groups), RunnerGroups: groups})
			}
			if len(groups) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No runner groups found for organization %q.\n", organization)
				return nil
			}
			return writeRunnerGroups(cmd, organization, groups)
		},
	}
	addRunnerGroupListFlags(cmd, opts, "runner groups")
	return cmd
}

func newCmdRunnerGroupView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "view <org> <group-id>",
		Short: "View an organization runner group",
		Example: `  ag org runner-group view my-organization group-id
  ag org runner-group view my-organization group-id --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, groupID, err := parseRunnerGroupArgs(args)
			if err != nil {
				return err
			}
			client, err := organizationActionsClient(f)
			if err != nil {
				return err
			}
			group, err := client.GetOrganizationRunnerGroup(organization, groupID)
			if err != nil {
				return fmt.Errorf("failed to view runner group %q for organization %q: %w", groupID, organization, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), group)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Organization: %s\n", displayValue(organization))
			fmt.Fprintf(cmd.OutOrStdout(), "Group ID: %s\n", displayValue(group.RunnerGroupID))
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", displayValue(group.RunnerGroupName))
			fmt.Fprintf(cmd.OutOrStdout(), "Share all repositories: %t\n", group.ShareAll)
			fmt.Fprintf(cmd.OutOrStdout(), "Share all public repositories: %t\n", group.ShareAllPublicRepos)
			fmt.Fprintf(cmd.OutOrStdout(), "Explicit shared repositories: %d\n", group.ExplicitSharedRepoCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", formatActionsTimestamp(group.CreatedAt))
			fmt.Fprintf(cmd.OutOrStdout(), "Updated: %s\n", formatActionsTimestamp(group.UpdatedAt))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output runner group as JSON")
	return cmd
}

func newCmdRunnerGroupRunners(f *cmdutil.Factory) *cobra.Command {
	opts := &runnerGroupListOptions{Limit: defaultRunnerGroupLimit}
	cmd := &cobra.Command{
		Use:   "runners <org> <group-id>",
		Short: "List host runners in an organization runner group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, groupID, client, err := prepareRunnerGroupList(f, args, opts.Limit)
			if err != nil {
				return err
			}
			runners, err := collectRunnerGroupPages(opts.Limit, func(runner actions.OrganizationRunner) string { return runner.ID }, func(page, perPage int) (int, []actions.OrganizationRunner, error) {
				response, err := client.ListOrganizationRunnerGroupRunners(organization, groupID, actions.ListRunnerGroupsOptions{Page: page, PerPage: perPage})
				return response.TotalCount, response.Runners, err
			})
			if err != nil {
				return fmt.Errorf("failed to list runners for runner group %q in organization %q: %w", groupID, organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), actions.OrganizationRunnerListResponse{TotalCount: len(runners), Runners: runners})
			}
			if len(runners) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No runners found for runner group %q in organization %q.\n", groupID, organization)
				return nil
			}
			return writeOrganizationRunners(cmd, organization, groupID, runners)
		},
	}
	addRunnerGroupListFlags(cmd, opts, "runners")
	return cmd
}

func newCmdRunnerGroupRunnerSets(f *cmdutil.Factory) *cobra.Command {
	opts := &runnerGroupListOptions{Limit: defaultRunnerGroupLimit}
	cmd := &cobra.Command{
		Use:   "runner-sets <org> <group-id>",
		Short: "List Kubernetes runner sets in an organization runner group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, groupID, client, err := prepareRunnerGroupList(f, args, opts.Limit)
			if err != nil {
				return err
			}
			sets, err := collectRunnerGroupPages(opts.Limit, func(set actions.RunnerSet) string { return set.ID }, func(page, perPage int) (int, []actions.RunnerSet, error) {
				response, err := client.ListOrganizationRunnerGroupRunnerSets(organization, groupID, actions.ListRunnerGroupsOptions{Page: page, PerPage: perPage})
				return response.TotalCount, response.RunnerSets, err
			})
			if err != nil {
				return fmt.Errorf("failed to list runner sets for runner group %q in organization %q: %w", groupID, organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), actions.RunnerSetListResponse{TotalCount: len(sets), RunnerSets: sets})
			}
			if len(sets) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No runner sets found for runner group %q in organization %q.\n", groupID, organization)
				return nil
			}
			return writeRunnerSets(cmd, organization, groupID, sets)
		},
	}
	addRunnerGroupListFlags(cmd, opts, "runner sets")
	return cmd
}

func newCmdRunnerGroupNamespaces(f *cmdutil.Factory) *cobra.Command {
	opts := &runnerGroupListOptions{Limit: defaultRunnerGroupLimit}
	cmd := &cobra.Command{
		Use:   "namespaces <org> <group-id>",
		Short: "List repositories that can use an organization runner group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, groupID, client, err := prepareRunnerGroupList(f, args, opts.Limit)
			if err != nil {
				return err
			}
			namespaces, err := collectRunnerGroupPages(opts.Limit, func(namespace actions.SharedNamespace) string { return namespace.ID }, func(page, perPage int) (int, []actions.SharedNamespace, error) {
				response, err := client.ListOrganizationRunnerGroupSharedNamespaces(organization, groupID, actions.ListRunnerGroupsOptions{Page: page, PerPage: perPage})
				return response.TotalCount, response.SharedNamespaces, err
			})
			if err != nil {
				return fmt.Errorf("failed to list shared namespaces for runner group %q in organization %q: %w", groupID, organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), actions.SharedNamespaceListResponse{TotalCount: len(namespaces), SharedNamespaces: namespaces})
			}
			if len(namespaces) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No shared namespaces found for runner group %q in organization %q.\n", groupID, organization)
				return nil
			}
			return writeSharedNamespaces(cmd, organization, groupID, namespaces)
		},
	}
	addRunnerGroupListFlags(cmd, opts, "shared namespaces")
	return cmd
}

func addRunnerGroupListFlags(cmd *cobra.Command, opts *runnerGroupListOptions, noun string) {
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultRunnerGroupLimit, "Maximum number of "+noun+" to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output "+noun+" as JSON")
}

func prepareRunnerGroupList(f *cmdutil.Factory, args []string, limit int) (string, string, *actions.Client, error) {
	organization, groupID, err := parseRunnerGroupArgs(args)
	if err != nil {
		return "", "", nil, err
	}
	if err := validateRunnerGroupLimit(limit); err != nil {
		return "", "", nil, err
	}
	client, err := organizationActionsClient(f)
	if err != nil {
		return "", "", nil, err
	}
	return organization, groupID, client, nil
}

func parseRunnerGroupArgs(args []string) (string, string, error) {
	organization, err := parseRunnerGroupOrganization(args[0])
	if err != nil {
		return "", "", err
	}
	groupID := strings.TrimSpace(args[1])
	if groupID == "" || strings.Contains(groupID, "/") || strings.IndexFunc(groupID, unicode.IsControl) >= 0 {
		return "", "", fmt.Errorf("invalid runner group ID: %q", args[1])
	}
	return organization, groupID, nil
}

func parseRunnerGroupOrganization(value string) (string, error) {
	organization, err := parseOrganization(value)
	if err != nil {
		return "", err
	}
	if strings.IndexFunc(organization, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("invalid organization: %q", value)
	}
	return organization, nil
}

func validateRunnerGroupLimit(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	return nil
}

func organizationActionsClient(f *cmdutil.Factory) (*actions.Client, error) {
	if f == nil || f.Config == nil {
		return nil, fmt.Errorf("configuration is unavailable")
	}
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	if f.HttpClient == nil {
		return actions.NewClient(token), nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return actions.NewClientWithHTTPClient(token, httpClient), nil
}

func collectRunnerGroupPages[T any](limit int, identity func(T) string, fetch func(page, perPage int) (int, []T, error)) ([]T, error) {
	if err := validateRunnerGroupLimit(limit); err != nil {
		return nil, err
	}
	items := make([]T, 0, min(limit, maxRunnerGroupsPerPage))
	seenItems := make(map[string]struct{})
	seenPages := make(map[string]struct{})
	expectedTotal := 0
	for page := 1; ; page++ {
		total, current, err := fetch(page, maxRunnerGroupsPerPage)
		if err != nil {
			return nil, err
		}
		if total > 0 {
			if expectedTotal == 0 {
				expectedTotal = total
			} else if total != expectedTotal {
				return nil, fmt.Errorf("inconsistent pagination: API reported total_count %d after %d", total, expectedTotal)
			}
		} else if expectedTotal > 0 {
			return nil, fmt.Errorf("inconsistent pagination: API reported total_count 0 after %d", expectedTotal)
		}
		if len(current) == 0 {
			if expectedTotal > len(items) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
			}
			break
		}
		fingerprint, err := json.Marshal(current)
		if err != nil {
			return nil, fmt.Errorf("fingerprint page: %w", err)
		}
		if _, exists := seenPages[string(fingerprint)]; exists {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		seenPages[string(fingerprint)] = struct{}{}
		newItems := 0
		for _, item := range current {
			key := strings.TrimSpace(identity(item))
			if key == "" {
				fingerprint, err := json.Marshal(item)
				if err != nil {
					return nil, fmt.Errorf("fingerprint item: %w", err)
				}
				key = "fallback:" + string(fingerprint)
			} else {
				key = "id:" + key
			}
			if _, exists := seenItems[key]; exists {
				continue
			}
			seenItems[key] = struct{}{}
			newItems++
			items = append(items, item)
			if expectedTotal > 0 && len(items) > expectedTotal {
				return nil, fmt.Errorf("inconsistent pagination: API reports %d items but %d were returned", expectedTotal, len(items))
			}
			if len(items) == limit {
				return items, nil
			}
		}
		if newItems == 0 {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		if expectedTotal > 0 && len(items) == expectedTotal {
			break
		}
		if len(current) < maxRunnerGroupsPerPage {
			if expectedTotal > len(items) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
			}
			break
		}
	}
	return items, nil
}

func writeRunnerGroups(cmd *cobra.Command, organization string, groups []actions.RunnerGroup) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ORGANIZATION\tGROUP ID\tNAME\tRUNNERS\tSHARE ALL\tNAMESPACE TYPE\tCREATED")
	for _, group := range groups {
		name := group.RunnerGroupName
		if strings.TrimSpace(name) == "" {
			name = group.Name
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%t\t%s\t%s\n", displayValue(organization), displayValue(group.ID), displayValue(name), group.RunnerCount, group.ShareAll, displayValue(group.NamespaceType), formatActionsTimestamp(group.CreateTime))
	}
	return w.Flush()
}

func writeOrganizationRunners(cmd *cobra.Command, organization, groupID string, runners []actions.OrganizationRunner) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ORGANIZATION\tGROUP ID\tRUNNER ID\tNAME\tSTATUS\tWORK DIR\tMEMORY %\tDISK GB\tLABELS")
	for _, runner := range runners {
		name := runner.RunnerName
		if strings.TrimSpace(name) == "" {
			name = runner.Name
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%g\t%g\t%s\n", displayValue(organization), displayValue(groupID), displayValue(runner.ID), displayValue(name), displayValue(runner.Status), displayValue(runner.WorkDir), runner.Memory, runner.Disk, displayRunnerLabels(runner.Labels))
	}
	return w.Flush()
}

func writeRunnerSets(cmd *cobra.Command, organization, groupID string, sets []actions.RunnerSet) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ORGANIZATION\tGROUP ID\tSET ID\tNAME\tSTATUS\tSIZE\tCPU\tMEMORY\tIMAGE\tCLUSTER\tNAMESPACE\tLABELS")
	for _, set := range sets {
		size := fmt.Sprintf("%d-%d", set.MinRunnerSize, set.MaxRunnerSize)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\n", displayValue(organization), displayValue(groupID), displayValue(set.ID), displayValue(set.Name), displayValue(set.Status), size, set.LimitCPU, set.LimitMemory, displayValue(set.ImageName), displayValue(set.UserK8SClusterName), displayValue(set.UserK8SClusterNamespace), displayRunnerLabels(set.RequiredLabels))
	}
	return w.Flush()
}

func writeSharedNamespaces(cmd *cobra.Command, organization, groupID string, namespaces []actions.SharedNamespace) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ORGANIZATION\tGROUP ID\tSHARE ID\tPATH\tVISIBILITY\tTYPE")
	for _, namespace := range namespaces {
		path := namespace.PathWithNamespace
		if strings.TrimSpace(path) == "" {
			path = namespace.Path
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", displayValue(organization), displayValue(groupID), displayValue(namespace.ID), displayValue(path), displayValue(namespace.Visibility), displayValue(namespace.Type))
	}
	return w.Flush()
}

func displayRunnerLabels(labels []actions.OrganizationRunnerLabel) string {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		name := strings.Join(strings.Fields(label.Name), " ")
		value := strings.Join(strings.Fields(label.Value), " ")
		switch {
		case name != "" && value != "":
			values = append(values, name+"="+value)
		case name != "":
			values = append(values, name)
		case value != "":
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

func formatActionsTimestamp(value actions.Timestamp) string {
	timestamp := value.Time()
	if timestamp.IsZero() {
		return "-"
	}
	return timestamp.UTC().Format(time.RFC3339)
}
