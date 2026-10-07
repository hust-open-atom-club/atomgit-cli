// Package mangen renders deterministic section 1 manuals from Cobra metadata.
// It never executes commands, initializers, or authentication hooks.
package mangen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Header contains explicit build metadata, never inferred from the environment.
// Empty values omit the date and version from the manual header.
type Header struct {
	Version string
	Date    string // YYYY-MM-DD
}

// Page is one complete, uncompressed section 1 manual.
type Page struct {
	Name string
	Data []byte
}

var safeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// Generate includes canonical, visible, non-deprecated commands and excludes
// hidden/deprecated subtrees. Aliases are documented on their canonical page.
// Cobra's runtime help/completion helpers are not initialized here.
func Generate(root *cobra.Command, header Header) ([]Page, error) {
	if root == nil || root.Hidden || root.Deprecated != "" {
		return nil, fmt.Errorf("a public root command is required")
	}
	if header.Date != "" {
		if _, err := time.Parse("2006-01-02", header.Date); err != nil {
			return nil, fmt.Errorf("invalid manual date: %w", err)
		}
	}
	if strings.ContainsAny(header.Version, "\r\n\x00") {
		return nil, fmt.Errorf("manual version must be a single line")
	}
	var commands []*cobra.Command
	names := make(map[*cobra.Command]string)
	used := make(map[string]bool)
	var visit func(*cobra.Command, string) error
	visit = func(cmd *cobra.Command, prefix string) error {
		if cmd.Hidden || cmd.Deprecated != "" {
			return nil
		}
		if !safeName.MatchString(cmd.Name()) {
			return fmt.Errorf("unsafe manual command name %q", cmd.Name())
		}
		name := prefix + cmd.Name()
		if used[name] {
			return fmt.Errorf("duplicate manual name %q", name)
		}
		used[name] = true
		names[cmd] = name
		commands = append(commands, cmd)
		for _, child := range cmd.Commands() {
			if err := visit(child, name+"-"); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root, ""); err != nil {
		return nil, err
	}
	pages := make([]Page, 0, len(commands))
	for _, cmd := range commands {
		pages = append(pages, Page{Name: names[cmd] + ".1", Data: render(cmd, names, header)})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })
	return pages, nil
}

func render(cmd *cobra.Command, names map[*cobra.Command]string, header Header) []byte {
	// Cobra lazily merges parent persistent flags. Resolve inheritance before
	// local flags and UseLine so the first render agrees with subsequent renders.
	inherited := cmd.InheritedFlags()
	local := cmd.NonInheritedFlags()
	var b strings.Builder
	fmt.Fprintf(&b, ".\\\" Generated from Cobra metadata; do not edit.\n.TH \"%s\" \"1\" \"%s\" \"%s\" \"AtomGit CLI Manual\"\n",
		escape(strings.ToUpper(names[cmd])), escape(header.Date), escape(strings.TrimSpace("AtomGit CLI "+header.Version)))
	b.WriteString(".SH NAME\n")
	text(&b, names[cmd]+" - "+cmd.Short)
	b.WriteString(".SH SYNOPSIS\n.nf\n")
	text(&b, cmd.UseLine())
	b.WriteString(".fi\n.SH DESCRIPTION\n")
	description := cmd.Long
	if description == "" {
		description = cmd.Short
	}
	text(&b, description)
	if len(cmd.Aliases) > 0 {
		aliases := append([]string(nil), cmd.Aliases...)
		sort.Strings(aliases)
		b.WriteString(".SH ALIASES\n")
		text(&b, strings.Join(aliases, ", "))
	}
	writeFlags(&b, "OPTIONS", local)
	writeFlags(&b, "INHERITED OPTIONS", inherited)
	if example := strings.TrimSpace(cmd.Example); example != "" {
		b.WriteString(".SH EXAMPLES\n.nf\n")
		text(&b, example)
		b.WriteString(".fi\n")
	}
	var references []string
	if parent, ok := names[cmd.Parent()]; ok {
		references = append(references, parent)
	}
	for _, child := range cmd.Commands() {
		if name, ok := names[child]; ok {
			references = append(references, name)
		}
	}
	sort.Strings(references)
	if len(references) > 0 {
		b.WriteString(".SH SEE ALSO\n")
		for i, reference := range references {
			if i > 0 {
				b.WriteString(",\n")
			}
			fmt.Fprintf(&b, "\\fB%s\\fR(1)", escape(reference))
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func writeFlags(b *strings.Builder, title string, flags *pflag.FlagSet) {
	var visible []*pflag.Flag
	flags.VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden {
			visible = append(visible, flag)
		}
	})
	if len(visible) == 0 {
		return
	}
	sort.Slice(visible, func(i, j int) bool { return visible[i].Name < visible[j].Name })
	fmt.Fprintf(b, ".SH %s\n", title)
	for _, flag := range visible {
		name := "--" + flag.Name
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			name = "-" + flag.Shorthand + ", " + name
		}
		if flag.Value.Type() != "bool" {
			name += " <" + flag.Value.Type() + ">"
		}
		b.WriteString(".TP\n")
		fmt.Fprintf(b, "\\fB%s\\fR\n", escape(name))
		text(b, flag.Usage)
		text(b, "Default: \""+flag.DefValue+"\".")
		if flag.NoOptDefVal != "" && flag.Value.Type() != "bool" {
			text(b, "Value when specified without an argument: \""+flag.NoOptDefVal+"\".")
		}
		if flag.Deprecated != "" {
			text(b, "Deprecated: "+flag.Deprecated)
		}
		if flag.ShorthandDeprecated != "" {
			text(b, "Deprecated shorthand -"+flag.Shorthand+": "+flag.ShorthandDeprecated)
		}
	}
}

// escape treats all metadata as literal text, not roff requests or escapes.
func escape(s string) string {
	return strings.NewReplacer("\\", `\e`, "-", `\-`, "\"", `\(dq`).Replace(s)
}

func text(b *strings.Builder, s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' || r == 127 {
			return -1
		}
		return r
	}, s)
	for line := range strings.SplitSeq(s, "\n") {
		if line == "" {
			b.WriteString(".PP\n")
		} else {
			// Also protects lines beginning with '.' or '\'' and preserves spacing.
			b.WriteString("\\&" + escape(line) + "\n")
		}
	}
}

// Write writes or checks generated pages. Unexpected .1 files are errors, never
// silently deleted; use an empty output directory after removing a command.
func Write(directory string, pages []Page, check bool) error {
	if !check {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create manual directory: %w", err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read manual directory: %w", err)
	}
	want := make(map[string]bool, len(pages))
	for _, page := range pages {
		if filepath.Base(page.Name) != page.Name || !strings.HasSuffix(page.Name, ".1") {
			return fmt.Errorf("invalid manual filename %q", page.Name)
		}
		want[page.Name] = true
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".1") && (!want[entry.Name()] || entry.IsDir() || entry.Type()&os.ModeSymlink != 0) {
			return fmt.Errorf("unexpected manual %s; use an empty output directory", entry.Name())
		}
	}
	for _, page := range pages {
		path := filepath.Join(directory, page.Name)
		if check {
			current, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read manual %s: %w", page.Name, err)
			}
			if !bytes.Equal(current, page.Data) {
				return fmt.Errorf("manual %s is out of date", page.Name)
			}
		} else if err := os.WriteFile(path, page.Data, 0o644); err != nil {
			return fmt.Errorf("write manual %s: %w", page.Name, err)
		}
	}
	return nil
}
