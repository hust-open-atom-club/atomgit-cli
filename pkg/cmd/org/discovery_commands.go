package org

import (
	"fmt"
	"io"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "view <org>",
		Short: "View an organization",
		Example: `  ag org view my-organization
  ag org view my-organization --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			return runOrgView(cmd.OutOrStdout(), f, organization, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output organization as JSON")
	return cmd
}

func runOrgView(out io.Writer, f *cmdutil.Factory, organization string, jsonOutput bool) error {
	client, err := organizationAPIClient(f)
	if err != nil {
		return err
	}

	var details api.Organization
	if err := client.Get(organizationEndpoint(organization, ""), &details); err != nil {
		return fmt.Errorf("failed to view organization %q: %w", organization, err)
	}
	result := organizationViewJSON{
		ID:          details.ID,
		Path:        organizationPath(details),
		Name:        details.Name,
		Visibility:  organizationVisibility(details),
		Description: details.Description,
		URL:         organizationURL(details, f.Config.GetHost()),
	}
	if jsonOutput {
		return cmdutil.WriteJSON(out, result)
	}

	fmt.Fprintf(out, "Path: %s\n", displayValue(result.Path))
	fmt.Fprintf(out, "Name: %s\n", displayValue(result.Name))
	fmt.Fprintf(out, "Visibility: %s\n", result.Visibility)
	fmt.Fprintf(out, "Description: %s\n", displayValue(result.Description))
	fmt.Fprintf(out, "URL: %s\n", displayValue(result.URL))
	return nil
}

func newCmdOrgMembers(f *cmdutil.Factory) *cobra.Command {
	opts := &collectionOptions{Limit: defaultOrganizationCollectionLimit}
	cmd := &cobra.Command{
		Use:   "members <org>",
		Short: "List organization members",
		Example: `  ag org members my-organization
  ag org members my-organization --limit 100
  ag org members my-organization --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			return runOrgMembers(cmd.OutOrStdout(), f, organization, opts)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultOrganizationCollectionLimit, "Maximum number of members to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output members as JSON")
	return cmd
}

func runOrgMembers(out io.Writer, f *cmdutil.Factory, organization string, opts *collectionOptions) error {
	client, err := organizationAPIClient(f)
	if err != nil {
		return err
	}
	members, err := api.GetPaginated[api.OrganizationMember](client, opts.Limit, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", organizationEndpoint(organization, "/members"), page, perPage)
	})
	if err != nil {
		return fmt.Errorf("failed to list members for organization %q: %w", organization, err)
	}

	result := make([]organizationMemberJSON, len(members))
	for index, member := range members {
		result[index] = organizationMemberJSON{Login: member.Login, Name: member.Name, Role: member.MemberRole}
	}
	if opts.JSON {
		return cmdutil.WriteJSON(out, result)
	}
	if len(result) == 0 {
		fmt.Fprintf(out, "No members found for organization %q.\n", organization)
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tROLE")
	for _, member := range result {
		fmt.Fprintf(w, "%s\t%s\t%s\n", displayValue(member.Login), displayValue(member.Name), displayValue(member.Role))
	}
	return w.Flush()
}

func newCmdOrgRepos(f *cmdutil.Factory) *cobra.Command {
	opts := &collectionOptions{Limit: defaultOrganizationCollectionLimit}
	cmd := &cobra.Command{
		Use:   "repos <org>",
		Short: "List organization repositories",
		Example: `  ag org repos my-organization
  ag org repos my-organization --limit 100
  ag org repos my-organization --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			return runOrgRepos(cmd.OutOrStdout(), f, organization, opts)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultOrganizationCollectionLimit, "Maximum number of repositories to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output repositories as JSON")
	return cmd
}

func runOrgRepos(out io.Writer, f *cmdutil.Factory, organization string, opts *collectionOptions) error {
	client, err := organizationAPIClient(f)
	if err != nil {
		return err
	}
	repositories, err := api.GetPaginated[api.Repository](client, opts.Limit, func(page, perPage int) string {
		return fmt.Sprintf("%s?page=%d&per_page=%d", organizationEndpoint(organization, "/repos"), page, perPage)
	})
	if err != nil {
		return fmt.Errorf("failed to list repositories for organization %q: %w", organization, err)
	}

	result := make([]organizationRepositoryJSON, len(repositories))
	for index, repository := range repositories {
		result[index] = organizationRepositoryJSON{
			ID: repository.ID, Name: repository.Name, Path: repositoryPath(repository),
			Visibility: organizationRepositoryVisibility(repository), Description: repository.Description,
			DefaultBranch: repository.DefaultBranch, Language: repository.Language,
			Stars: repository.StarsCount, Forks: repository.ForksCount, UpdatedAt: repository.UpdatedAt,
			URL: repositoryURL(repository, organization, f.Config.GetHost()),
		}
	}
	if opts.JSON {
		return cmdutil.WriteJSON(out, result)
	}
	if len(result) == 0 {
		fmt.Fprintf(out, "No repositories found for organization %q.\n", organization)
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tVISIBILITY\tDESCRIPTION\tDEFAULT BRANCH\tLANGUAGE\tSTARS\tFORKS\tUPDATED\tURL")
	for _, repository := range result {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\n",
			displayValue(repository.Path), repository.Visibility, displayValue(repository.Description),
			displayValue(repository.DefaultBranch), displayValue(repository.Language), repository.Stars,
			repository.Forks, displayValue(repository.UpdatedAt), displayValue(repository.URL))
	}
	return w.Flush()
}

func organizationAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create AtomGit API client: %w", err)
	}
	return client, nil
}
