package schema_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type forbiddenConfig struct{ t *testing.T }

func (f forbiddenConfig) GetToken() (string, error) { f.t.Fatal("read token"); return "", nil }
func (f forbiddenConfig) GetUser() (string, error)  { f.t.Fatal("read user"); return "", nil }
func (f forbiddenConfig) GetHost() string           { f.t.Fatal("read host"); return "" }

func TestRegisteredCommands(t *testing.T) {
	for _, tc := range []struct {
		path            []string
		min, max        int
		output, effects string
	}{
		{[]string{"run", "cancel"}, 1, 2, "text", "write"},
		{[]string{"run", "stop"}, 1, 2, "text", "write"},
		{[]string{"run", "rerun"}, 1, 2, "text", "write"},
		{[]string{"run", "retry"}, 1, 2, "text", "write"},
		{[]string{"pr", "create"}, 0, 1, "text", "write"},
		{[]string{"api"}, 1, 1, "raw", "conditional"},
		{[]string{"pr", "comment", "create"}, 1, 2, "text", "write"},
		{[]string{"pr", "comment", "refresh-position"}, 1, 2, "text,json", "write"},
		{[]string{"pr", "comment", "resolve"}, 2, 3, "text", "write"},
		{[]string{"pr", "comment", "unresolve"}, 2, 3, "text", "write"},
		{[]string{"discussion", "edit"}, 1, 2, "text", "write"},
		{[]string{"discussion", "create"}, 0, 1, "text", "write"},
		{[]string{"discussion", "delete"}, 1, 2, "text", "write"},
		{[]string{"org", "create"}, 1, 1, "text", "write"},
		{[]string{"user", "followers"}, 1, 1, "text,json", "read"},
		{[]string{"org", "edit"}, 1, 1, "text,json", "write"},
		{[]string{"org", "member"}, 0, 0, "text", "none"},
		{[]string{"org", "member", "invite"}, 2, 2, "text", "write"},
		{[]string{"org", "member", "remove"}, 2, 2, "text", "write"},
		{[]string{"org", "followers"}, 1, 1, "text,json", "read"},
		{[]string{"user", "following"}, 1, 1, "text,json", "read"},
		{[]string{"repo", "archive"}, 0, 1, "text", "write"},
		{[]string{"repo", "unarchive"}, 0, 1, "text", "write"},
		{[]string{"repo", "permissions"}, 0, 1, "text,json", "read"},
		{[]string{"repo", "settings"}, 0, 0, "text", "none"},
		{[]string{"repo", "settings", "view"}, 0, 1, "text,json", "read"},
		{[]string{"repo", "settings", "edit"}, 0, 1, "text,json", "write"},
		{[]string{"repo", "module", "edit"}, 0, 1, "text,json", "write"},
		{[]string{"repo", "cla", "list"}, 0, 1, "text,json", "read"},
		{[]string{"repo", "cla", "set"}, 0, 1, "text", "write"},
		{[]string{"repo", "cla", "disable"}, 0, 1, "text", "write"},
		{[]string{"org", "discussion", "list"}, 1, 1, "text,json", "read"},
		{[]string{"org", "discussion", "view"}, 2, 2, "text,json", "read"},
		{[]string{"org", "discussion", "create"}, 1, 1, "text", "write"},
		{[]string{"org", "discussion", "edit"}, 2, 2, "text", "write"},
		{[]string{"org", "discussion", "delete"}, 2, 2, "text", "write"},
		{[]string{"org", "discussion", "comment", "create"}, 2, 2, "text", "write"},
		{[]string{"org", "discussion", "comment", "reply"}, 3, 3, "text", "write"},
		{[]string{"org", "discussion", "comment", "edit"}, 3, 3, "text", "write"},
		{[]string{"org", "discussion", "comment", "delete"}, 3, 3, "text", "write"},
	} {
		t.Run(strings.Join(tc.path, "/"), func(t *testing.T) {
			f := &cmdutil.Factory{
				Config:             forbiddenConfig{t},
				HttpClient:         func() (*http.Client, error) { t.Fatal("created HTTP client"); return nil, nil },
				RepositoryResolver: func() (cmdutil.Repository, error) { t.Fatal("resolved repository"); return cmdutil.Repository{}, nil },
			}
			cmd, err := root.NewCmdRoot(f)
			if err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SetArgs(append([]string{"schema"}, tc.path...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var doc commandschema.Document
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatalf("%v: %s", err, out.String())
			}
			if doc.FormatVersion != 1 || doc.CLIVersion.Version == "" || doc.Command == nil || stderr.Len() != 0 {
				t.Fatalf("bad envelope: %+v / %s", doc, stderr.String())
			}
			detail := doc.Command
			if detail.Positionals == nil || detail.Positionals.MinCount != tc.min || detail.Positionals.MaxCount != tc.max {
				t.Fatalf("positionals: %+v", detail.Positionals)
			}
			if detail.Capabilities.Output != tc.output || detail.Capabilities.Effects != tc.effects {
				t.Fatalf("capabilities: %+v", detail.Capabilities)
			}
			target, _, err := cmd.Find(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			if detail.Path == "ag-cli repo module edit" {
				// Argument validation also rejects an empty update before config loading.
				if err := target.Flags().Set("wiki", "false"); err != nil {
					t.Fatal(err)
				}
				want := []commandschema.FlagGroup{{Kind: "atLeastOneRequired", Flags: []string{"analysis", "discussion", "fork", "issue", "merge-request", "security", "wiki"}}}
				if !reflect.DeepEqual(detail.FlagGroups, want) {
					t.Fatalf("module constraints = %+v", detail.FlagGroups)
				}
			}
			for n := 0; n <= tc.max+1; n++ {
				args := make([]string, n)
				if len(tc.path) == 2 && tc.path[0] == "run" && n > 0 {
					args[n-1] = "run-1"
				}
				if valid := target.Args(target, args) == nil; valid != (n >= tc.min && n <= tc.max) {
					t.Fatalf("argument count metadata drift at %d", n)
				}
			}
			if detail.Path == "ag-cli pr create" {
				title := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == "title" })
				if title < 0 || !detail.Flags[title].Required || detail.Flags[title].RequiredSource != "annotation" {
					t.Fatal("missing manual required-title annotation")
				}
				if len(detail.FlagGroups) != 1 || detail.FlagGroups[0].Kind != "mutuallyExclusive" {
					t.Fatal("missing body/body-file constraint")
				}
			}
			if detail.Path == "ag-cli org create" {
				for name, defaultValue := range map[string]string{"name": "", "description": "", "visibility": "public"} {
					i := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == name })
					if i < 0 || detail.Flags[i].Required || detail.Flags[i].Default != defaultValue || detail.Flags[i].Type != "string" {
						t.Fatalf("incorrect optional flag metadata for %s", name)
					}
				}
			}
			if detail.Path == "ag-cli repo settings edit" {
				want := []commandschema.FlagGroup{{Kind: "atLeastOneRequired", Flags: []string{
					"action-pipeline-enabled", "block-cross-repo-pr-trigger", "branch-name-regex", "disable-fork",
					"forbidden-committer-create-branch", "forbidden-developer-create-branch", "forbidden-developer-create-branch-user-ids",
					"forbidden-developer-create-tag", "generate-pre-merge-ref", "include-lfs-objects", "open-gpg-verified",
					"rebase-disable-trigger-webhook", "show-branch-creator", "tag-name-regex",
				}}}
				if !reflect.DeepEqual(detail.FlagGroups, want) {
					t.Fatalf("repository settings constraints = %+v", detail.FlagGroups)
				}
			}
			if detail.Path == "ag-cli org edit" {
				want := []commandschema.FlagGroup{
					{Kind: "atLeastOneRequired", Flags: []string{"description", "email", "location", "name", "website"}},
				}
				if !reflect.DeepEqual(detail.FlagGroups, want) {
					t.Fatalf("organization edit constraints = %+v", detail.FlagGroups)
				}
			}
			if detail.Path == "ag-cli org member invite" {
				for name, defaultValue := range map[string]string{"permission": "push", "role-id": ""} {
					i := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == name })
					if i < 0 || detail.Flags[i].Required || detail.Flags[i].Default != defaultValue || detail.Flags[i].Type != "string" {
						t.Fatalf("incorrect conditional flag metadata for %s", name)
					}
				}
				if !strings.Contains(strings.Join(detail.Notes, " "), "required for customized and forbidden otherwise") {
					t.Fatal("missing conditional role constraint")
				}
			}
			if detail.Path == "ag-cli discussion create" || detail.Path == "ag-cli org discussion create" {
				for _, name := range []string{"title", "category"} {
					i := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == name })
					if i < 0 || !detail.Flags[i].Required {
						t.Fatalf("missing required flag %s", name)
					}
				}
				want := []commandschema.FlagGroup{
					{Kind: "atLeastOneRequired", Flags: []string{"body", "body-file"}},
					{Kind: "mutuallyExclusive", Flags: []string{"body", "body-file"}},
				}
				if !reflect.DeepEqual(detail.FlagGroups, want) {
					t.Fatalf("create constraints = %+v", detail.FlagGroups)
				}
			}
			if detail.Path == "ag-cli discussion edit" || detail.Path == "ag-cli org discussion edit" {
				wantGroups := []commandschema.FlagGroup{
					{Kind: "atLeastOneRequired", Flags: []string{"body", "body-file", "category", "title"}},
					{Kind: "mutuallyExclusive", Flags: []string{"body", "body-file"}},
				}
				if !reflect.DeepEqual(detail.FlagGroups, wantGroups) {
					t.Fatalf("edit constraints = %+v", detail.FlagGroups)
				}
				if !strings.Contains(strings.Join(detail.Notes, " "), "stdin") {
					t.Fatal("missing body input metadata")
				}
			}
			if strings.HasPrefix(detail.Path, "ag-cli org discussion comment ") && !strings.HasSuffix(detail.Path, " delete") {
				want := []commandschema.FlagGroup{
					{Kind: "atLeastOneRequired", Flags: []string{"body", "body-file"}},
					{Kind: "mutuallyExclusive", Flags: []string{"body", "body-file"}},
				}
				if !reflect.DeepEqual(detail.FlagGroups, want) {
					t.Fatalf("comment body constraints = %+v", detail.FlagGroups)
				}
			}
			if detail.Path == "ag-cli org discussion list" || detail.Path == "ag-cli org discussion view" {
				jsonFlag := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == "json" })
				if jsonFlag < 0 || detail.Flags[jsonFlag].Type != "bool" || detail.Flags[jsonFlag].Default != "false" {
					t.Fatal("missing optional JSON output flag")
				}
				if !strings.Contains(strings.ToLower(detail.Positionals.Description), "organization") {
					t.Fatal("missing organization argument description")
				}
				if detail.Path == "ag-cli org discussion list" {
					limit := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == "limit" })
					if limit < 0 || detail.Flags[limit].Type != "int" || detail.Flags[limit].Default != "30" || detail.Flags[limit].Shorthand != "L" {
						t.Fatal("missing list limit metadata")
					}
				} else {
					comments := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == "comments" })
					if comments < 0 || detail.Flags[comments].Type != "bool" || detail.Flags[comments].Default != "false" {
						t.Fatal("missing optional comments metadata")
					}
				}
			}
			global := slices.IndexFunc(detail.Flags, func(f commandschema.Flag) bool { return f.Name == "raw-output" })
			if global < 0 || !detail.Flags[global].Inherited || detail.Flags[global].DefinedOn != "ag-cli" {
				t.Fatal("missing inherited flags")
			}
		})
	}
}

func TestAllPublicCommandsCanBeDescribed(t *testing.T) {
	cmd, err := root.NewCmdRoot(&cmdutil.Factory{Config: forbiddenConfig{t}})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := commandschema.Describe(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range directory.Commands {
		path := strings.Fields(summary.Path)
		detail, err := commandschema.Describe(cmd, path)
		if err != nil || detail.Command == nil || detail.Command.Path != summary.Path || len(detail.Commands) != 0 {
			t.Fatalf("%s: %v", summary.Path, err)
		}
		repeated, err := commandschema.Describe(cmd, path)
		if err != nil || !reflect.DeepEqual(detail, repeated) {
			t.Fatalf("%s: description changed on repeat query: %v", summary.Path, err)
		}
		if len(path) == 1 {
			continue
		}
		short, err := commandschema.Describe(cmd, path[1:])
		if err != nil || !reflect.DeepEqual(detail, short) {
			t.Fatalf("%s: prefixed and short paths differ: %v", summary.Path, err)
		}
		for _, alias := range summary.Aliases {
			aliasPath := append([]string{}, path...)
			aliasPath[len(aliasPath)-1] = alias
			aliasDoc, err := commandschema.Describe(cmd, aliasPath)
			if err != nil || !reflect.DeepEqual(detail, aliasDoc) {
				t.Fatalf("alias %s: %v", alias, err)
			}
			shortAlias, err := commandschema.Describe(cmd, aliasPath[1:])
			if err != nil || !reflect.DeepEqual(detail, shortAlias) {
				t.Fatalf("short alias %s: %v", alias, err)
			}
		}
	}
	for _, path := range [][]string{{"auth", "git-credential"}, {"missing"}, {"ag-cli", "auth", "git-credential"}, {"ag-cli", "missing"}, {"ag-cli", "ag-cli"}} {
		_, err := commandschema.Describe(cmd, path)
		if err == nil {
			t.Fatal(fmt.Sprint("accepted hidden/unknown path ", path))
		}
	}
}
