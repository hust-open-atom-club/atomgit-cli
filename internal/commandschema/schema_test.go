package commandschema

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestStaticDescriptionAndFlagGroups(t *testing.T) {
	root := &cobra.Command{Use: "ag"}
	root.PersistentFlags().Bool("global", false, "Inherited switch")
	root.PersistentFlags().String("shadow", "parent", "Parent value")
	leaf := &cobra.Command{
		Use: "create [repo]", Short: "Create something", Aliases: []string{"z", "c"}, Deprecated: "use another command",
		Example: "ag create example", Args: func(*cobra.Command, []string) error { t.Fatal("ran validator"); return nil },
		RunE:    func(*cobra.Command, []string) error { t.Fatal("ran business logic"); return nil },
		PreRunE: func(*cobra.Command, []string) error { t.Fatal("ran pre-hook"); return nil },
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			t.Fatal("ran completion")
			return nil, 0
		},
	}
	root.AddCommand(leaf)
	leaf.Flags().StringP("title", "t", "", "Title")
	leaf.Flags().Int("count", 5, "Count")
	leaf.Flags().Bool("json", false, "JSON")
	leaf.Flags().StringSlice("label", []string{"one", "two"}, "Labels")
	leaf.Flags().String("shadow", "child", "Local value")
	leaf.Flags().String("internal", "hidden-secret", "Internal")
	if err := leaf.Flags().MarkHidden("internal"); err != nil {
		t.Fatal(err)
	}
	if err := leaf.MarkFlagRequired("title"); err != nil {
		t.Fatal(err)
	}
	leaf.MarkFlagsMutuallyExclusive("count", "json")
	leaf.MarkFlagsOneRequired("title", "label")
	leaf.MarkFlagsRequiredTogether("global", "label")
	leaf.MarkFlagsMutuallyExclusive("internal", "title")
	if err := leaf.Flags().Set("title", "runtime-secret"); err != nil {
		t.Fatal(err)
	}
	if err := leaf.Flags().Set("count", "99"); err != nil {
		t.Fatal(err)
	}
	Annotate(leaf, Metadata{Positionals: &Positionals{MinCount: 0, MaxCount: 1, Description: "Optional repository"}})

	doc, err := Describe(root, []string{"c"})
	if err != nil {
		t.Fatal(err)
	}
	got := doc.Command
	if got.Path != "ag create" || !reflect.DeepEqual(got.Aliases, []string{"c", "z"}) || got.Deprecated == "" {
		t.Fatalf("summary: %+v", got.Summary)
	}
	if got.Capabilities.Output != "undescribed" || got.Capabilities.Effects != "undescribed" || got.Validation != "partial" {
		t.Fatalf("invented capabilities: %+v", got)
	}
	if got.Positionals.MinCount != 0 || got.Positionals.MaxCount != 1 {
		t.Fatalf("positionals: %+v", got.Positionals)
	}
	for _, f := range got.Flags {
		switch f.Name {
		case "title":
			if !f.Required || f.RequiredSource != "cobra" || f.Default != "" || f.Shorthand != "t" {
				t.Fatalf("title: %+v", f)
			}
		case "count":
			if f.Type != "int" || f.Default != "5" {
				t.Fatalf("count: %+v", f)
			}
		case "global":
			if !f.Inherited || f.DefinedOn != "ag" || f.Type != "bool" || f.NoOptDefault != "true" {
				t.Fatalf("global: %+v", f)
			}
		case "shadow":
			if f.Inherited || f.DefinedOn != "ag create" || f.Default != "child" {
				t.Fatalf("shadow: %+v", f)
			}
		case "label":
			if f.Type != "stringSlice" || f.Default != "[one,two]" {
				t.Fatalf("label: %+v", f)
			}
		case "internal":
			t.Fatal("hidden flag leaked")
		}
	}
	wantGroups := []FlagGroup{
		{Kind: "atLeastOneRequired", Flags: []string{"label", "title"}},
		{Kind: "mutuallyExclusive", Flags: []string{"count", "json"}},
		{Kind: "requiredTogether", Flags: []string{"global", "label"}},
	}
	if !reflect.DeepEqual(got.FlagGroups, wantGroups) {
		t.Fatalf("groups = %+v", got.FlagGroups)
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "internal") {
		t.Fatal("runtime value or hidden metadata leaked")
	}
	for range 20 {
		next, err := Describe(root, []string{"create"})
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(next)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != string(encoded) {
			t.Fatal("unstable output or alias-dependent metadata")
		}
	}
}

func TestCatalogueTracksTreeAndHidesSubtrees(t *testing.T) {
	root := &cobra.Command{Use: "ag"}
	public := &cobra.Command{Use: "zebra", Aliases: []string{"z"}}
	hidden := &cobra.Command{Use: "credential-helper", Aliases: []string{"secret"}, Hidden: true}
	hidden.AddCommand(&cobra.Command{Use: "public-looking"})
	root.AddCommand(public, hidden)
	paths := func() []string {
		t.Helper()
		doc, err := Describe(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		result := []string{}
		for _, c := range doc.Commands {
			result = append(result, c.Path)
		}
		return result
	}
	if got := paths(); !reflect.DeepEqual(got, []string{"ag", "ag zebra"}) {
		t.Fatal(got)
	}
	public.Use = "renamed"
	added := &cobra.Command{Use: "added", Deprecated: "old but public"}
	root.AddCommand(added)
	if got := paths(); !reflect.DeepEqual(got, []string{"ag", "ag added", "ag renamed"}) {
		t.Fatal(got)
	}
	root.RemoveCommand(added)
	if got := paths(); !reflect.DeepEqual(got, []string{"ag", "ag renamed"}) {
		t.Fatal(got)
	}
	for _, path := range [][]string{{"credential-helper"}, {"secret"}, {"credential-helper", "public-looking"}, {"missing"}, {"ren"}, {"renamed", "extra"}, {"renamed", "--help"}} {
		if _, err := Describe(root, path); err == nil {
			t.Fatalf("accepted invalid path %v", path)
		}
	}
}

func TestInheritedOnlyUsageIsStable(t *testing.T) {
	root := &cobra.Command{Use: "ag"}
	root.PersistentFlags().Bool("global", false, "Inherited switch")
	leaf := &cobra.Command{Use: "diff <number>"}
	root.AddCommand(leaf)
	first, err := Describe(root, []string{"diff"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Describe(root, []string{"diff"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Command.Usage != "ag diff <number> [flags]" || !reflect.DeepEqual(first, second) {
		t.Fatalf("inherited-only usage changes across queries: first=%q second=%q", first.Command.Usage, second.Command.Usage)
	}
}

func TestRootPathDescription(t *testing.T) {
	root := &cobra.Command{
		Use: "tool <command>", Short: "Root command",
		Args: func(*cobra.Command, []string) error { t.Fatal("ran root validator"); return nil },
		RunE: func(*cobra.Command, []string) error { t.Fatal("ran root command"); return nil },
	}
	root.PersistentFlags().Bool("global", false, "Global switch")
	root.AddCommand(&cobra.Command{Use: "child", Aliases: []string{"c"}})
	for _, tc := range []struct {
		name string
		path []string
		want string
	}{
		{"root", []string{"tool"}, "tool"},
		{"short", []string{"child"}, "tool child"},
		{"prefixed", []string{"tool", "child"}, "tool child"},
		{"prefixed alias", []string{"tool", "c"}, "tool child"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := Describe(root, tc.path)
			if err != nil || doc.Command == nil || doc.Command.Path != tc.want || len(doc.Commands) != 0 {
				t.Fatalf("Describe(%v) = %+v, %v", tc.path, doc, err)
			}
			if tc.name == "root" {
				if len(doc.Command.Flags) != 1 || doc.Command.Flags[0].Name != "global" || doc.Command.Flags[0].Inherited || len(doc.Command.Subcommands) != 1 {
					t.Fatalf("root metadata: %+v", doc.Command)
				}
			}
		})
	}
	directory, err := Describe(root, nil)
	if err != nil || directory.Command != nil || len(directory.Commands) != 2 {
		t.Fatalf("empty path no longer returns catalogue: %+v, %v", directory, err)
	}
}
