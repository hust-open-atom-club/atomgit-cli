// Package docsgen renders a deterministic Markdown reference from a Cobra tree.
package docsgen

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Generate writes a stable command reference for root to w.
//
// The output is derived only from Cobra metadata. It intentionally excludes
// runtime state such as command execution results, timestamps, and versions.
func Generate(dst io.Writer, root *cobra.Command) error {
	if root == nil {
		return fmt.Errorf("root command is nil")
	}

	var output strings.Builder
	w := &output
	commands := collectCommands(root)
	if _, err := io.WriteString(w, "# AtomGit CLI command reference\n\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "> This file is generated from the Cobra command tree. Do not edit it manually.\n> Regenerate with `go run ./scripts/generate-command-reference`.\n\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "> Only commands registered in the root command tree are emitted. The optional Cobra completion helper is not registered by this CLI and is intentionally omitted.\n\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Command index\n\n"); err != nil {
		return err
	}
	for _, command := range commands {
		if _, err := fmt.Fprintf(w, "- [%s](#%s) — %s\n", command.path, anchor(command.path), escapeInline(command.short)); err != nil {
			return err
		}
	}

	for _, command := range commands {
		if _, err := fmt.Fprintf(w, "\n## %s\n\n", command.path); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Usage: `%s`\n\n", command.use); err != nil {
			return err
		}
		if command.short != "" {
			if _, err := fmt.Fprintf(w, "%s\n\n", command.short); err != nil {
				return err
			}
		}
		if command.long != "" && command.long != command.short {
			if _, err := fmt.Fprintf(w, "%s\n\n", command.long); err != nil {
				return err
			}
		}
		if len(command.aliases) > 0 {
			if _, err := fmt.Fprintf(w, "Aliases: `%s`\n\n", strings.Join(command.aliases, "`, `")); err != nil {
				return err
			}
		}
		if command.deprecated != "" {
			if _, err := fmt.Fprintf(w, "> Deprecated: %s\n\n", command.deprecated); err != nil {
				return err
			}
		}
		if command.hidden {
			if _, err := io.WriteString(w, "> Hidden command.\n\n"); err != nil {
				return err
			}
		}
		if len(command.flags) > 0 {
			if _, err := io.WriteString(w, "### Flags\n\n| Flag | Description | Default | Scope |\n| --- | --- | --- | --- |\n"); err != nil {
				return err
			}
			for _, flag := range command.flags {
				if _, err := fmt.Fprintf(w, "| `%s` | %s | `%s` | %s |\n", flag.name, escapeCell(flag.usage), escapeCell(flag.defaultValue), flag.scope); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		if command.example != "" {
			if _, err := fmt.Fprintf(w, "### Example\n\n```bash\n%s\n```\n\n", command.example); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(dst, strings.TrimRight(output.String(), "\n")+"\n")
	return err
}

type commandInfo struct {
	path       string
	use        string
	short      string
	long       string
	aliases    []string
	deprecated string
	hidden     bool
	example    string
	flags      []flagInfo
}

type flagInfo struct {
	name         string
	usage        string
	defaultValue string
	scope        string
}

func collectCommands(root *cobra.Command) []commandInfo {
	var result []commandInfo
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		result = append(result, commandInfo{
			path:       command.CommandPath(),
			use:        command.UseLine(),
			short:      strings.TrimSpace(command.Short),
			long:       strings.TrimSpace(command.Long),
			aliases:    append([]string(nil), command.Aliases...),
			deprecated: strings.TrimSpace(command.Deprecated),
			hidden:     command.Hidden,
			example:    normalizeExample(command.Example),
			flags:      collectFlags(command),
		})

		children := command.Commands()
		sort.SliceStable(children, func(i, j int) bool {
			return children[i].Name() < children[j].Name()
		})
		for _, child := range children {
			visit(child)
		}
	}
	visit(root)
	return result
}

// normalizeExample removes shared help-text indentation while preserving
// relative indentation in shell continuations and nested examples.
func normalizeExample(example string) string {
	lines := strings.Split(strings.ReplaceAll(example, "\r\n", "\n"), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	prefix := lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " \t"))]
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		for !strings.HasPrefix(line, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		} else {
			lines[i] = strings.TrimPrefix(line, prefix)
		}
	}
	return strings.Join(lines, "\n")
}

func collectFlags(command *cobra.Command) []flagInfo {
	flags := make(map[string]flagInfo)
	command.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		flags[flag.Name] = makeFlagInfo(flag, "inherited")
	})
	command.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		flags[flag.Name] = makeFlagInfo(flag, "local")
	})

	result := make([]flagInfo, 0, len(flags))
	for _, flag := range flags {
		result = append(result, flag)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result
}

func makeFlagInfo(flag *pflag.Flag, scope string) flagInfo {
	name := "--" + flag.Name
	if flag.Shorthand != "" {
		name = "-" + flag.Shorthand + ", " + name
	}
	return flagInfo{
		name:         name,
		usage:        strings.TrimSpace(flag.Usage),
		defaultValue: flag.DefValue,
		scope:        scope,
	}
}

func anchor(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func escapeInline(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"[", "\\[",
		"]", "\\]",
	).Replace(value)
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.Join(strings.Fields(value), " ")
}
