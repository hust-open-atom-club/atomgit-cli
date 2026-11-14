package org

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type editOptions struct {
	Name, Description, Email, Location, Website string
	Yes, JSON                                   bool
}

type editOrganizationJSON struct {
	Organization    api.EditedOrganization      `json:"organization"`
	SubmittedFields api.OrganizationEditRequest `json:"submitted_fields"`
	Verification    string                      `json:"verification"`
}

func newCmdOrgEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use: "edit <org>", Short: "Edit an organization profile",
		Long: `Edit the display name, description, public email, location, or website of an organization.
Only explicitly supplied fields are sent. Name cannot be empty; pass an empty
string to clear description, email, location, or website. This does not rename
the organization path. --website maps to html_url, the profile's website field.
All updates require confirmation unless --yes is supplied. PATCH requests are
never automatically retried or redirected. The response identity is checked;
submitted fields are shown without claiming an independent profile readback.
If the result cannot be confirmed, check AtomGit before retrying.`,
		Example: "  ag-cli org edit my-team --name 'My Team' --description 'Team projects'\n  ag-cli org edit my-team --email '' --location Wuhan --yes\n  ag-cli org edit my-team --website www.example.com --json --yes",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			request, err := buildOrgEditRequest(cmd, opts)
			if err != nil {
				return err
			}
			if !opts.Yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Organization: %s\nSubmitted fields:\n", organization); err != nil {
					return err
				}
				if err := cmdutil.WriteJSON(cmd.ErrOrStderr(), request); err != nil {
					return err
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply these organization profile changes? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "Organization update cancelled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			updated, err := api.EditOrganization(client.WithContext(cmd.Context()), organization, request)
			if err != nil {
				return fmt.Errorf("failed to edit organization %q: %w", organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), editOrganizationJSON{updated, request, "identity_only"})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated organization %s (ID: %d)\nSubmitted fields (not independently verified):\n", updated.Path, updated.ID); err != nil {
				return err
			}
			return cmdutil.WriteJSON(cmd.OutOrStdout(), request)
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "Organization display name (non-empty; does not change path)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Organization description (empty clears)")
	cmd.Flags().StringVar(&opts.Email, "email", "", "Public email address (empty clears)")
	cmd.Flags().StringVar(&opts.Location, "location", "", "Organization location (empty clears)")
	cmd.Flags().StringVar(&opts.Website, "website", "", "Organization website, sent as html_url (empty clears)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip update confirmation")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output acknowledged identity and submitted fields as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Explicit organization path; Git repository context is not used."},
		Output:      "text,json", Effects: "write", Notes: []string{"Only explicitly supplied fields are sent; name is a display name, not the path.", "Requires confirmation unless --yes is supplied. No automatic PATCH retries or redirects.", "Output reports the response identity and submitted fields, not an independently verified complete profile."},
	})
	return cmd
}

func buildOrgEditRequest(cmd *cobra.Command, opts *editOptions) (api.OrganizationEditRequest, error) {
	request := api.OrganizationEditRequest{}
	for _, field := range []struct {
		flag   string
		value  *string
		target **string
	}{
		{"name", &opts.Name, &request.Name}, {"description", &opts.Description, &request.Description},
		{"email", &opts.Email, &request.Email}, {"location", &opts.Location, &request.Location}, {"website", &opts.Website, &request.Website},
	} {
		if cmd.Flags().Changed(field.flag) {
			*field.target = field.value
		}
	}
	return request, request.Validate()
}
