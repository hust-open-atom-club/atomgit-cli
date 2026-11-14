package org

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type memberInviteOptions struct {
	Permission string
	RoleID     string
	Yes        bool
}

func newCmdOrgMember(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use: "member", Short: "Invite or remove organization members",
		Long: `Invite users to an organization or remove organization members.

Use "ag-cli org members <org>" to list members. These commands manage
organization membership, not repository collaborators or enterprise members.
They do not provide an interface to edit an existing member's role.`,
	}
	cmd.AddCommand(newCmdOrgMemberInvite(f), newCmdOrgMemberRemove(f))
	return cmd
}

func newCmdOrgMemberInvite(f *cmdutil.Factory) *cobra.Command {
	opts := &memberInviteOptions{}
	cmd := &cobra.Command{
		Use: "invite <org> <username>", Short: "Invite a user to an organization",
		Long: `Submit an invitation with pull, push, admin, or customized permission.

Permission defaults to push. Customized permission requires a non-empty
--role-id; --role-id is not accepted with other permissions.
Confirm the organization, username and requested permission with y or yes,
or pass --yes. Empty input, EOF or another answer cancels without a request.
Success means the invitation request was sent, not that the user accepted it.
Writes are not automatically retried or redirected; check AtomGit before
retrying an uncertain result.`,
		Example: `  ag-cli org member invite my-org alice
  ag-cli org member invite my-org alice --permission pull --yes
  ag-cli org member invite my-org alice --permission customized --role-id role-123`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, username, err := parseOrganizationMember(args)
			if err != nil {
				return err
			}
			switch opts.Permission {
			case "pull", "push", "admin", "customized":
			default:
				return fmt.Errorf("invalid permission: must be pull, push, admin, or customized")
			}
			roleID := strings.TrimSpace(opts.RoleID)
			if opts.Permission == "customized" {
				if roleID == "" {
					return fmt.Errorf("--role-id must be non-empty with customized permission")
				}
			} else if cmd.Flags().Changed("role-id") {
				return fmt.Errorf("--role-id requires --permission customized")
			}
			permission := opts.Permission
			if opts.Permission == "customized" {
				permission += fmt.Sprintf(" (role ID %q)", roleID)
			}
			if !opts.Yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(),
					fmt.Sprintf("Invite %s to organization %s with %s permission? [y/N] ", username, organization, permission))
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Invitation cancelled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			request := api.OrganizationInvitationRequest{Permission: opts.Permission, RoleID: roleID}
			if err := api.InviteOrganizationMember(client, organization, username, request); err != nil {
				return fmt.Errorf("failed to invite %s to organization %s (not retried; check AtomGit before retrying): %w", username, organization, err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Invitation request sent for %s to organization %s (requested permission: %s).\nAcceptance by the user is not verified.\n", username, organization, permission)
			return err
		},
	}
	cmd.Flags().StringVar(&opts.Permission, "permission", "push", "Requested permission: pull, push, admin, or customized")
	cmd.Flags().StringVar(&opts.RoleID, "role-id", "", "Custom role ID (required only for customized permission)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip invitation confirmation")
	cmdutil.RegisterValueCompletion(cmd, "permission", []string{"pull", "push", "admin", "customized"})
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "Explicit organization path and username; no Git repository inference."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and confirmation (y/yes or --yes). Cancellation sends no request.",
			"Permission defaults to push. --role-id is required for customized and forbidden otherwise.",
			"HTTP 200 acknowledges the invitation request, not acceptance by the user. Writes are not retried or redirected."},
	})
	return cmd
}

func newCmdOrgMemberRemove(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "remove <org> <username>", Short: "Remove a member from an organization",
		Long: `Remove a user from the explicitly named organization.

Confirm with y or yes, or pass --yes. Empty input, EOF or another answer
cancels without a request. This affects organization membership, not a
repository collaborator or an enterprise member. Writes are not automatically
retried or redirected; check AtomGit before retrying an uncertain result.`,
		Example: "  ag-cli org member remove my-org alice\n  ag-cli org member remove my-org alice --yes",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, username, err := parseOrganizationMember(args)
			if err != nil {
				return err
			}
			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(),
					fmt.Sprintf("Remove member %s from organization %s? [y/N] ", username, organization))
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Removal cancelled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if err := api.RemoveOrganizationMember(client, organization, username); err != nil {
				return fmt.Errorf("failed to remove %s from organization %s (not retried; check AtomGit before retrying): %w", username, organization, err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed member %s from organization %s\n", username, organization)
			return err
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip removal confirmation")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "Explicit organization path and username; no Git repository inference."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and confirmation (y/yes or --yes). Cancellation sends no request.",
			"Removes organization membership only. Accepts an empty HTTP 200 response; writes are not retried or redirected."},
	})
	return cmd
}

func parseOrganizationMember(args []string) (string, string, error) {
	organization, err := parseDiscussionOrganization(args[0])
	if err != nil {
		return "", "", err
	}
	if !validDiscussionPathID(args[1]) {
		return "", "", fmt.Errorf("invalid username: %q", args[1])
	}
	return organization, args[1], nil
}
