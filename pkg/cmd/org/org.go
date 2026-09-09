package org

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const defaultOrganizationListLimit = 30

type listOptions struct {
	Limit int
	JSON  bool
}

type organizationJSON struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func NewCmdOrg(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
		Long:  "View organizations associated with your AtomGit account.",
	}
	cmd.AddCommand(newCmdOrgList(f))
	return cmd
}

func newCmdOrgList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: defaultOrganizationListLimit}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations for the authenticated user",
		Example: `  ag org list
  ag org list --limit 100
  ag org list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOrgList(cmd.OutOrStdout(), f, opts)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultOrganizationListLimit, "Maximum number of organizations to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output organizations as JSON")
	return cmd
}

func runOrgList(out io.Writer, f *cmdutil.Factory, opts *listOptions) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}
	organizations, err := api.GetPaginated[api.Organization](client, opts.Limit, func(page, perPage int) string {
		return fmt.Sprintf("/users/orgs?page=%d&per_page=%d", page, perPage)
	})
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if opts.JSON {
		return cmdutil.WriteJSON(out, organizationsJSON(organizations, f.Config.GetHost()))
	}
	if len(organizations) == 0 {
		fmt.Fprintln(out, "No organizations found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tNAME\tURL")
	for _, organization := range organizations {
		fmt.Fprintf(w, "%s\t%s\t%s\n", organizationPath(organization), displayValue(organization.Name), organizationURL(organization, f.Config.GetHost()))
	}
	return w.Flush()
}

func organizationsJSON(organizations []api.Organization, host string) []organizationJSON {
	result := make([]organizationJSON, len(organizations))
	for index, organization := range organizations {
		result[index] = organizationJSON{
			ID:          organization.ID,
			Path:        organizationPath(organization),
			Name:        organization.Name,
			URL:         organizationURL(organization, host),
			Description: organization.Description,
		}
	}
	return result
}

func organizationPath(organization api.Organization) string {
	if path := strings.TrimSpace(organization.Path); path != "" {
		return path
	}
	return strings.TrimSpace(organization.Login)
}

func organizationURL(organization api.Organization, host string) string {
	if webURL := strings.TrimSpace(organization.HTMLURL); webURL != "" {
		return webURL
	}
	path := organizationPath(organization)
	if path == "" {
		return "-"
	}
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		host = "atomgit.com"
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return host + "/" + url.PathEscape(path)
}

func displayValue(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	return value
}
