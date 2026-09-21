package commandschema

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FormatVersion versions this command description format, not JSON Schema.
const FormatVersion = 1

type Document struct {
	FormatVersion int          `json:"formatVersion"`
	CLIVersion    version.Info `json:"cliVersion"`
	Commands      []Summary    `json:"commands,omitempty"`
	Command       *Command     `json:"command,omitempty"`
}

type Summary struct {
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Deprecated  string   `json:"deprecated"`
	Runnable    bool     `json:"runnable"`
}

type Command struct {
	Summary
	Usage           string       `json:"usage"`
	LongDescription string       `json:"longDescription"`
	Example         string       `json:"example"`
	Subcommands     []Summary    `json:"subcommands"`
	Flags           []Flag       `json:"flags"`
	FlagGroups      []FlagGroup  `json:"flagGroups"`
	Positionals     *Positionals `json:"positionals"`
	Capabilities    Capabilities `json:"capabilities"`
	Validation      string       `json:"validation"`
	Undescribed     []string     `json:"undescribed"`
	Notes           []string     `json:"notes"`
}

type Flag struct {
	Name                string `json:"name"`
	Shorthand           string `json:"shorthand"`
	Description         string `json:"description"`
	Type                string `json:"type"`
	Default             string `json:"default"`
	NoOptDefault        string `json:"noOptDefault"`
	Required            bool   `json:"required"`
	RequiredSource      string `json:"requiredSource"`
	Inherited           bool   `json:"inherited"`
	DefinedOn           string `json:"definedOn"`
	Deprecated          string `json:"deprecated"`
	ShorthandDeprecated string `json:"shorthandDeprecated"`
}

type FlagGroup struct {
	Kind  string   `json:"kind"`
	Flags []string `json:"flags"`
}

type Capabilities struct {
	Output         string `json:"output"`
	Effects        string `json:"effects"`
	Permissions    string `json:"permissions"`
	ResponseSchema string `json:"responseSchema"`
}

// Describe reads metadata only. Exact canonical names and built-in aliases are
// accepted; no prefix matching, flag parsing, local aliases, or command hooks.
func Describe(root *cobra.Command, path []string) (Document, error) {
	if root == nil {
		return Document{}, fmt.Errorf("command tree is unavailable")
	}
	doc := Document{FormatVersion: FormatVersion, CLIVersion: version.Get()}
	if len(path) == 0 {
		var visit func(*cobra.Command)
		visit = func(cmd *cobra.Command) {
			if cmd.Hidden {
				return
			}
			doc.Commands = append(doc.Commands, summarize(cmd))
			for _, child := range cmd.Commands() {
				visit(child)
			}
		}
		visit(root)
		sort.Slice(doc.Commands, func(i, j int) bool { return doc.Commands[i].Path < doc.Commands[j].Path })
		return doc, nil
	}
	cmd := root
	for _, part := range path {
		var next *cobra.Command
		for _, child := range cmd.Commands() {
			if !child.Hidden && child.Name() == part {
				next = child
				break
			}
		}
		if next == nil {
			for _, child := range cmd.Commands() {
				if !child.Hidden && slices.Contains(child.Aliases, part) {
					next = child
					break
				}
			}
		}
		if next == nil {
			return Document{}, fmt.Errorf("unknown or hidden command path; run 'ag schema' to list public commands")
		}
		cmd = next
	}
	detail, err := describeCommand(cmd)
	if err != nil {
		return Document{}, err
	}
	doc.Command = &detail
	return doc, nil
}

func summarize(cmd *cobra.Command) Summary {
	aliases := append([]string{}, cmd.Aliases...)
	sort.Strings(aliases)
	return Summary{Path: cmd.CommandPath(), Description: strings.TrimSpace(cmd.Short),
		Aliases: aliases, Deprecated: cmd.Deprecated, Runnable: cmd.Runnable()}
}

func describeCommand(cmd *cobra.Command) (Command, error) {
	metadata := Metadata{}
	if data := cmd.Annotations[annotationKey]; data != "" {
		if err := json.Unmarshal([]byte(data), &metadata); err != nil {
			return Command{}, fmt.Errorf("invalid command metadata for %s: %w", cmd.CommandPath(), err)
		}
	}
	// Cobra merges persistent flags lazily. Merge them before UseLine, or a
	// command with only inherited flags omits [flags] on its first query.
	flags := make(map[string]*pflag.Flag)
	inherited := cmd.InheritedFlags()
	inherited.VisitAll(func(f *pflag.Flag) { flags[f.Name] = f })
	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) { flags[f.Name] = f })
	result := Command{
		Summary: summarize(cmd), Usage: cmd.UseLine(), LongDescription: strings.TrimSpace(cmd.Long),
		Example: strings.TrimSpace(cmd.Example), Subcommands: []Summary{}, Flags: []Flag{}, FlagGroups: []FlagGroup{},
		Positionals: metadata.Positionals, Validation: "partial",
		Capabilities: Capabilities{Output: "undescribed", Effects: "undescribed", Permissions: "undescribed", ResponseSchema: "undescribed"},
		Undescribed:  []string{"businessValidation", "permissions", "responseSchema"}, Notes: append([]string{}, metadata.Notes...),
	}
	if metadata.Positionals == nil {
		result.Undescribed = append(result.Undescribed, "positionals")
	}
	if metadata.Output != "" {
		result.Capabilities.Output = metadata.Output
	} else {
		result.Undescribed = append(result.Undescribed, "output")
	}
	if metadata.Effects != "" {
		result.Capabilities.Effects = metadata.Effects
	} else {
		result.Undescribed = append(result.Undescribed, "effects")
	}
	sort.Strings(result.Undescribed)
	for _, child := range cmd.Commands() {
		if !child.Hidden {
			result.Subcommands = append(result.Subcommands, summarize(child))
		}
	}
	sort.Slice(result.Subcommands, func(i, j int) bool { return result.Subcommands[i].Path < result.Subcommands[j].Path })
	for _, f := range flags {
		if f.Hidden {
			continue
		}
		requiredSource := "none"
		if values := f.Annotations[cobra.BashCompOneRequiredFlag]; len(values) > 0 && values[0] == "true" {
			requiredSource = "cobra"
		} else if slices.Contains(metadata.RequiredFlags, f.Name) {
			requiredSource = "annotation"
		}
		isInherited := inherited.Lookup(f.Name) == f
		definedOn := cmd.CommandPath()
		if isInherited {
			for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
				if parent.PersistentFlags().Lookup(f.Name) == f {
					definedOn = parent.CommandPath()
					break
				}
			}
		}
		result.Flags = append(result.Flags, Flag{
			Name: f.Name, Shorthand: f.Shorthand, Description: f.Usage, Type: f.Value.Type(),
			Default: f.DefValue, NoOptDefault: f.NoOptDefVal, Required: requiredSource != "none", RequiredSource: requiredSource,
			Inherited: isInherited, DefinedOn: definedOn, Deprecated: f.Deprecated, ShorthandDeprecated: f.ShorthandDeprecated,
		})
	}
	sort.Slice(result.Flags, func(i, j int) bool { return result.Flags[i].Name < result.Flags[j].Name })
	result.FlagGroups = flagGroups(flags)
	return result, nil
}

// Cobra v1.10 stores flag groups in these annotations (not exported constants).
// Compatibility tests use the public MarkFlags* methods to detect drift.
func flagGroups(flags map[string]*pflag.Flag) []FlagGroup {
	kinds := map[string]string{
		"cobra_annotation_required_if_others_set": "requiredTogether",
		"cobra_annotation_one_required":           "atLeastOneRequired",
		"cobra_annotation_mutually_exclusive":     "mutuallyExclusive",
	}
	groups := make(map[string]FlagGroup)
	for _, flag := range flags {
		if flag.Hidden {
			continue
		}
		for annotation, kind := range kinds {
			for _, value := range flag.Annotations[annotation] {
				names := strings.Fields(value)
				visible := true
				for _, name := range names {
					if f := flags[name]; f == nil || f.Hidden {
						visible = false
					}
				}
				if !visible || len(names) == 0 {
					continue
				}
				sort.Strings(names)
				groups[kind+":"+strings.Join(names, " ")] = FlagGroup{Kind: kind, Flags: names}
			}
		}
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]FlagGroup, 0, len(keys))
	for _, key := range keys {
		result = append(result, groups[key])
	}
	return result
}
