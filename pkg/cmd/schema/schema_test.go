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
		{[]string{"pr", "create"}, 0, 1, "text", "write"},
		{[]string{"api"}, 1, 1, "raw", "conditional"},
		{[]string{"pr", "comment", "create"}, 1, 2, "text", "write"},
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
			for n := 0; n <= tc.max+1; n++ {
				args := make([]string, n)
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
