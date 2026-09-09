package alias

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdAlias(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Create command shortcuts",
		Long:  `Create, list, and delete command shortcuts (aliases) for "ag" commands.`,
	}

	cmd.AddCommand(newCmdAliasSet(f))
	cmd.AddCommand(newCmdAliasList(f))
	cmd.AddCommand(newCmdAliasDelete(f))
	return cmd
}

func newCmdAliasSet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <alias> <expansion>...",
		Short: "Create a shortcut for an ag command",
		Long: `Create a shortcut for an ag command.

Aliases are expanded at invocation time: the first non-flag argument of an ag
invocation is looked up and replaced with the expansion. Aliases never
override built-in commands, so names that conflict with a built-in command
are rejected, and the expansion must start with a known built-in command.

To include a literal space inside an expansion argument (for example a
Windows path), escape it with a backslash: C:\Program\ Files.`,
		Example: `  ag alias set pl "pr list"
  ag alias set rv repo view`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			expansion := strings.TrimSpace(strings.Join(args[1:], " "))
			if err := validateAliasName(name); err != nil {
				return err
			}
			if err := validateAliasExpansion(expansion); err != nil {
				return err
			}
			// Built-in commands always take precedence at expansion time, so
			// an alias that shadows one would never run; reject it up front.
			for _, c := range cmd.Root().Commands() {
				if c.Name() == name {
					return fmt.Errorf("alias name %q conflicts with the built-in command %q; built-in commands always take precedence", name, c.Name())
				}
			}
			// Expansion only ever fires through a real built-in command, so
			// reject typos and alias chains at set time instead of failing on
			// every invocation.
			if err := validateExpansionTarget(cmd.Root(), expansion); err != nil {
				return err
			}
			if err := config.SaveAlias(name, expansion); err != nil {
				return fmt.Errorf("failed to save alias: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added alias %s: %s\n", name, expansion)
			return nil
		},
	}
	return cmd
}

func newCmdAliasList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List aliases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			aliases, err := config.LoadAliases()
			if err != nil {
				return fmt.Errorf("failed to load aliases: %w", err)
			}
			out := cmd.OutOrStdout()
			if len(aliases) == 0 {
				fmt.Fprintln(out, "No aliases configured.")
				return nil
			}
			names := make([]string, 0, len(aliases))
			for name := range aliases {
				names = append(names, name)
			}
			sort.Strings(names)
			w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			for _, name := range names {
				fmt.Fprintf(w, "%s\t%s\n", name, aliases[name])
			}
			return w.Flush()
		},
	}
	return cmd
}

func newCmdAliasDelete(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <alias>",
		Short: "Delete an alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			deleted, err := config.DeleteAlias(name)
			if err != nil {
				return fmt.Errorf("failed to delete alias: %w", err)
			}
			if !deleted {
				return fmt.Errorf("alias %q not found", name)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted alias %s\n", name)
			return nil
		},
	}
	return cmd
}

// validateAliasName checks that an alias name is a single, flag-free token.
func validateAliasName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("alias name %q cannot start with '-'", name)
	}
	if strings.ContainsAny(name, " \t\n") {
		return fmt.Errorf("alias name %q cannot contain whitespace", name)
	}
	return nil
}

// validateAliasExpansion checks that an expansion is a non-empty command
// fragment and not a shell-style alias.
func validateAliasExpansion(expansion string) error {
	if expansion == "" {
		return fmt.Errorf("alias expansion cannot be empty")
	}
	if strings.HasPrefix(expansion, "!") {
		return fmt.Errorf("shell-style aliases (starting with '!') are not supported")
	}
	return nil
}

// validateExpansionTarget ensures the expansion begins with a real built-in
// command, so a typo in the command word or an alias chain is rejected at
// set time rather than failing on every invocation. Aliases expand only
// once, so an expansion that refers to another alias would never resolve.
func validateExpansionTarget(root *cobra.Command, expansion string) error {
	fields := SplitExpansion(expansion)
	if len(fields) == 0 {
		return nil
	}
	// cobra's Find/Traverse are lenient about unknown deeper tokens, so
	// validate the command word itself against the top-level commands.
	c, _, err := root.Find([]string{fields[0]})
	if err != nil || c == nil || c == root {
		return fmt.Errorf("expansion %q does not start with a known ag command", expansion)
	}
	return nil
}

// SplitExpansion tokenizes an alias expansion on whitespace while honoring
// backslash-escaped spaces and tabs, so Windows paths such as
// "C:\Program\ Files" survive as a single token. Any other backslash
// sequence is kept verbatim.
func SplitExpansion(s string) []string {
	var fields []string
	var cur strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\t') {
			cur.WriteRune(runes[i+1])
			i++
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}
