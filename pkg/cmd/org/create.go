package org

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type createOptions struct {
	Name, Description, Visibility string
}

func newCmdOrgCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use: "create <path>", Short: "Create an organization",
		Long: `Create an AtomGit organization using an explicit path.

The name defaults to the path and visibility defaults to public. Use
--visibility private to create a private organization. Omitted options are left
to the API defaults; --description="" explicitly sends an empty description.
Creation requests are never automatically retried or redirected. If the
response cannot be read or verified, check AtomGit before retrying.`,
		Example: "  ag-cli org create my-organization\n  ag-cli org create my-team --name 'My Team' --visibility private\n  ag-cli org create my-team --description 'Team projects'",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := parseDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			request := api.OrganizationCreateRequest{Path: path}
			if cmd.Flags().Changed("name") {
				if strings.TrimSpace(opts.Name) == "" {
					return fmt.Errorf("organization name must not be empty")
				}
				request.Name = &opts.Name
			}
			if opts.Visibility != "public" && opts.Visibility != "private" {
				return fmt.Errorf("invalid visibility: must be public or private")
			}
			if cmd.Flags().Changed("visibility") {
				request.Visibility = &opts.Visibility
			}
			if cmd.Flags().Changed("description") {
				request.Description = &opts.Description
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			result, err := api.CreateOrganization(client, request)
			if err != nil {
				return fmt.Errorf("failed to create organization %q: %w", path, err)
			}
			if result.WebURL != "" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created organization %s: %s\n", result.Path, result.WebURL)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created organization %s\n", result.Path)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "Organization name (defaults to path)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Organization description")
	cmd.Flags().StringVar(&opts.Visibility, "visibility", "public", "Organization visibility: public or private")
	cmdutil.RegisterValueCompletion(cmd, "visibility", []string{"public", "private"})
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Explicit organization path; Git repository context is not used."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication. Name defaults to path; visibility defaults to public.", "Only explicitly supplied optional fields are sent. Creation is never automatically retried or redirected. Verify remote state before retrying an uncertain request."},
	})
	return cmd
}
