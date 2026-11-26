package org

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type rolesOptions struct {
	JSON bool
}

func newCmdOrgRoles(f *cmdutil.Factory) *cobra.Command {
	opts := &rolesOptions{}
	cmd := &cobra.Command{
		Use:   "roles <org>",
		Short: "List customized roles of an organization",
		Long: `List the customized roles defined on an organization.

Requires authentication and permission to view the organization's customized
roles; anonymous access is never used. Role IDs are printed as strings and can
be passed unchanged to "org member invite" with the customized permission and
--role-id, so the ID is never replaced by a name or a display name. The
endpoint is documented without pagination parameters, so every returned role is
printed once, unchanged. A field the server omits is shown as "-" instead of a
substituted value, and unknown fields are preserved in --json output.
The organization response uses its own field names (id, name, name_cn, ...);
the repository customized-role field names are not applied to it.
This command only reads roles: it does not create or edit roles.`,
		Example: `  ag-cli org roles hust-open-atom-club
  ag-cli org roles hust-open-atom-club --json`,
		// Validate the organization in Args rather than RunE: the real entry
		// loads credential configuration in PersistentPreRunE, which runs after
		// argument validation, so a malformed path must fail before it.
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			_, err := parseOrganization(args[0])
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseOrganization(args[0])
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			roles, err := api.GetOrganizationCustomRoles(client.WithContext(cmd.Context()), organization)
			if err != nil {
				return fmt.Errorf("failed to list customized roles for organization %q: %w", organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), roles)
			}
			return writeOrganizationCustomRoles(cmd.OutOrStdout(), organization, roles)
		},
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output customized roles as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "One explicit organization path."},
		Output:      "text,json", Effects: "read",
		Notes: []string{"Requires authentication to view customized roles; never falls back to anonymous access.",
			"Role IDs are printed as strings and are suitable for org member invite --permission customized --role-id.",
			"No pagination parameters are documented for this endpoint, so the response is printed as received and no page is requested.",
			"JSON preserves unknown fields and the documented organization role fields the server returned; absent fields are not filled in. No role is created or assigned."},
	})
	return cmd
}

func writeOrganizationCustomRoles(out io.Writer, organization string, roles []api.OrganizationCustomRole) error {
	if len(roles) == 0 {
		_, err := fmt.Fprintf(out, "No customized roles found for organization %q.\n", organization)
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	// NAME is last because it is the only display column that can contain wide
	// characters, combining marks or emoji; ROLE ID, ACCESS LEVEL and MEMBERS
	// are ASCII under the API contract (role IDs are server-generated
	// identifiers and the other two are validated integers). text/tabwriter
	// pads by rune count, so no aligned column follows a cell whose rune count
	// can differ from its terminal width, and no width table is needed.
	if _, err := fmt.Fprintln(w, "ROLE ID\tACCESS LEVEL\tMEMBERS\tNAME"); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			organizationRoleTextField(role, "id"),
			organizationRoleIntegerField(role, "access_level"),
			organizationRoleIntegerField(role, "member_count"),
			organizationRoleName(role),
		); err != nil {
			return err
		}
	}
	return w.Flush()
}

// organizationRoleName renders the role name together with its Chinese name
// when the server returns both and they differ after trimming surrounding
// whitespace. An absent name is shown as "-" instead of a substituted value.
func organizationRoleName(role api.OrganizationCustomRole) string {
	name := strings.TrimSpace(organizationRoleField(role, "name"))
	chinese := strings.TrimSpace(organizationRoleField(role, "name_cn"))
	switch {
	case name != "" && chinese != "" && name != chinese:
		return organizationEscapeTableCell(name + "（" + chinese + "）")
	case name != "":
		return organizationEscapeTableCell(name)
	case chinese != "":
		return organizationEscapeTableCell(chinese)
	default:
		return "-"
	}
}

// organizationRoleField returns a documented string field, or "" when the
// server omitted it.
func organizationRoleField(role api.OrganizationCustomRole, field string) string {
	value, _ := role.StringField(field)
	return value
}

// organizationTableCellEscaper escapes the characters text/tabwriter treats as
// structure: a vertical tab ends a cell and a form feed ends a line block.
// EscapeTSVField covers the backslash, tab, newline and carriage return.
var organizationTableCellEscaper = strings.NewReplacer("\v", `\v`, "\f", `\f`)

// organizationEscapeTableCell escapes a value before it is written to a
// tabwriter, so a remote name or ID cannot add a cell or a line.
func organizationEscapeTableCell(value string) string {
	return organizationTableCellEscaper.Replace(cmdutil.EscapeTSVField(value))
}

// organizationRoleTextField escapes a string field so a tab, newline, carriage
// return or other control character reported by the server cannot break the
// table or the terminal. An absent or blank field is shown as "-" rather than a
// substitute.
func organizationRoleTextField(role api.OrganizationCustomRole, field string) string {
	value, ok := role.StringField(field)
	if !ok {
		return "-"
	}
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return organizationEscapeTableCell(value)
}

func organizationRoleIntegerField(role api.OrganizationCustomRole, field string) string {
	value, ok := role.IntegerField(field)
	if !ok {
		return "-"
	}
	return strconv.FormatInt(value, 10)
}
