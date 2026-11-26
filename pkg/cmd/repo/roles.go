package repo

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

func newCmdRepoRoles(f *cmdutil.Factory) *cobra.Command {
	opts := &rolesOptions{}
	cmd := &cobra.Command{
		Use:   "roles [<owner>/<repo>]",
		Short: "List customized roles of a repository",
		Long: `List the customized roles defined on a repository.

Requires authentication and permission to view the repository's customized
roles; anonymous access is never used. Role IDs are printed as strings, so they
are never replaced by a username or a role display name. The endpoint is
documented without pagination parameters, so every returned role is printed
once. The NAME column shows the role name together with its Chinese name when
the server returns both and they differ after trimming surrounding whitespace;
a "field the server omitted" replacement is not applied inside NAME. Every
other field the server omits is shown as "-" instead of a substituted value,
and unknown fields are preserved in --json output.
This command only reads roles: it does not create roles, and it does not change
an existing collaborator's permission or role.`,
		Example: `  ag-cli repo roles owner/repo
  ag-cli repo roles --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			roles, err := api.GetRepositoryCustomRoles(client.WithContext(cmd.Context()), repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("failed to list customized roles for %s: %w", repository, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), roles)
			}
			return writeRepositoryCustomRoles(cmd.OutOrStdout(), roles)
		},
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output customized roles as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; explicit repository takes precedence over Git context."},
		Output:      "text,json", Effects: "read",
		Notes: []string{"Requires authentication to view customized roles; never falls back to anonymous access.",
			"Role IDs are printed as strings and are not replaced by usernames or display names.",
			"No pagination parameters are documented for this endpoint, so the response is printed as received and no page is requested.",
			"JSON preserves unknown fields and the documented fields the server returned; absent fields are not filled in. No role is created or assigned."},
	})
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func writeRepositoryCustomRoles(out io.Writer, roles []api.RepositoryCustomRole) error {
	if len(roles) == 0 {
		_, err := fmt.Fprintln(out, "No customized roles found.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	// NAME is last because it is the only display column that can contain wide
	// characters, combining marks or emoji; ROLE ID, ACCESS LEVEL and MEMBERS
	// hold printable ASCII under the API contract (role IDs are server-generated
	// identifiers and the other two are validated integers). text/tabwriter
	// pads by rune count, so no aligned column follows a cell whose rune count
	// can differ from its terminal width, and no width table is needed.
	if _, err := fmt.Fprintln(w, "ROLE ID\tACCESS LEVEL\tMEMBERS\tNAME"); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			roleTextField(role, "role_id"),
			roleIntegerField(role, "access_level"),
			roleIntegerField(role, "member_count"),
			roleName(role),
		); err != nil {
			return err
		}
	}
	return w.Flush()
}

// roleName renders the role name together with its Chinese name when the
// server returns both and they differ after trimming surrounding whitespace.
// An absent name is shown as "-" instead of a substituted value.
func roleName(role api.RepositoryCustomRole) string {
	name := strings.TrimSpace(roleField(role, "role_name"))
	chinese := strings.TrimSpace(roleField(role, "role_chinese_name"))
	switch {
	case name != "" && chinese != "" && name != chinese:
		return escapeTableCell(name + "（" + chinese + "）")
	case name != "":
		return escapeTableCell(name)
	case chinese != "":
		return escapeTableCell(chinese)
	default:
		return "-"
	}
}

// roleField returns a documented string field, or "" when the server omitted
// it.
func roleField(role api.RepositoryCustomRole, field string) string {
	value, _ := role.StringField(field)
	return value
}

// escapeTableCell escapes a value before it is written to a tabwriter. The
// terminal sanitizer runs here as well as in the command writer, so control
// characters - including a vertical tab, which tabwriter treats as a cell
// terminator, and a form feed, which ends a line block - are expanded before
// column widths are computed. A field containing them therefore cannot add a
// cell, add a line, or shift a later column.
func escapeTableCell(value string) string {
	return cmdutil.SanitizeTerminal(cmdutil.EscapeTSVField(value))
}

// roleTextField escapes a string field so a tab, newline, carriage return or
// other control character reported by the server cannot break the table or the
// terminal. An absent or blank field is shown as "-" rather than a substitute.
func roleTextField(role api.RepositoryCustomRole, field string) string {
	value, ok := role.StringField(field)
	if !ok {
		return "-"
	}
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return escapeTableCell(value)
}

func roleIntegerField(role api.RepositoryCustomRole, field string) string {
	value, ok := role.IntegerField(field)
	if !ok {
		return "-"
	}
	return strconv.FormatInt(value, 10)
}
