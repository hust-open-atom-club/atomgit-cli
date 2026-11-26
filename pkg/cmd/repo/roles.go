package repo

import (
	"fmt"
	"io"
	"strconv"
	"strings"

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
once, unchanged. A field the server omits is shown as "-" instead of a
substituted value, and unknown fields are preserved in --json output.
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
	rows := make([][]string, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, []string{
			roleTextField(role, "role_id"),
			roleTextField(role, "role_name"),
			roleIntegerField(role, "access_level"),
			roleIntegerField(role, "member_count"),
			roleTextField(role, "role_chinese_name"),
		})
	}
	return writeAlignedRows(out, []string{"ROLE ID", "NAME", "ACCESS LEVEL", "MEMBERS", "CHINESE NAME"}, rows)
}

// writeAlignedRows pads cells by terminal display width instead of rune count.
// text/tabwriter counts runes, so a Chinese name would occupy two cells per
// rune and shift every later column on that row; padding by display width
// keeps the numeric columns aligned for names of any script.
func writeAlignedRows(out io.Writer, header []string, rows [][]string) error {
	widths := make([]int, len(header))
	for _, row := range append([][]string{header}, rows...) {
		for i, cell := range row {
			if width := displayWidth(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}
	writeRow := func(row []string) error {
		var line strings.Builder
		for i, cell := range row {
			line.WriteString(cell)
			if i < len(row)-1 {
				line.WriteString(strings.Repeat(" ", widths[i]-displayWidth(cell)+2))
			}
		}
		line.WriteString("\n")
		_, err := io.WriteString(out, line.String())
		return err
	}
	if err := writeRow(header); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(row); err != nil {
			return err
		}
	}
	return nil
}

// displayWidth returns the number of terminal cells s occupies.
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if isWideRune(r) {
			width += 2
		} else {
			width++
		}
	}
	return width
}

// isWideRune reports whether r occupies two terminal cells, i.e. its East
// Asian Width property is W or F. The ranges cover the wide blocks an API
// response can contain: CJK ideographs, kana, Hangul, fullwidth forms and
// common emoji.
func isWideRune(r rune) bool {
	switch {
	case r < 0x1100:
		return false
	case r <= 0x115F, r == 0x2329, r == 0x232A:
		return true
	case r >= 0x2E80 && r <= 0x303E,
		r >= 0x3041 && r <= 0x33FF,
		r >= 0x3400 && r <= 0x4DBF,
		r >= 0x4E00 && r <= 0x9FFF,
		r >= 0xA000 && r <= 0xA4CF,
		r >= 0xA960 && r <= 0xA97F,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE10 && r <= 0xFE19,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F,
		r >= 0x1F900 && r <= 0x1F9FF,
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	}
	return false
}

// roleTextField escapes a string field so a tab, newline, carriage return or
// control character reported by the server cannot break the table or the
// terminal. An absent or blank field is shown as "-" rather than a substitute.
func roleTextField(role api.RepositoryCustomRole, field string) string {
	value, ok := role.StringField(field)
	if !ok {
		return "-"
	}
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return cmdutil.EscapeTSVField(value)
}

func roleIntegerField(role api.RepositoryCustomRole, field string) string {
	value, ok := role.IntegerField(field)
	if !ok {
		return "-"
	}
	return strconv.FormatInt(value, 10)
}
