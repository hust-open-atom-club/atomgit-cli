package root

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestContentCompatibilityOutput(t *testing.T) {
	// Warning-like file bytes must not be mistaken for a diagnostic and filtered.
	const content = "Command \"read-file\" is deprecated, use something else\n\x1b[31mfile\x1b[0m\n"
	for _, raw := range []bool{false, true} {
		for _, entry := range []struct {
			name              string
			args              []string
			directory, legacy bool
		}{
			{"legacy file", []string{"read-file", "owner/repo", "file"}, false, true},
			{"legacy directory", []string{"read-dir", "owner/repo", "."}, true, true},
			{"modern file", []string{"content", "view", "owner/repo", "file"}, false, false},
			{"modern directory", []string{"content", "list", "owner/repo", "."}, true, false},
		} {
			for _, asJSON := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/raw=%t/json=%t", entry.name, raw, asJSON), func(t *testing.T) {
					var stdout, stderr bytes.Buffer
					factory := &cmdutil.Factory{Config: rootTestConfig{}, HttpClient: func() (*http.Client, error) {
						return &http.Client{Transport: rootRoundTripFunc(func(req *http.Request) (*http.Response, error) {
							body := fmt.Sprintf(`{"type":"file","name":"file","path":"file","sha":"abc","size":%d,"encoding":"base64","content":%q}`, len(content), base64.StdEncoding.EncodeToString([]byte(content)))
							if entry.directory {
								body = `[{"type":"file","name":"file","path":"file","sha":"abc","size":4}]`
							}
							return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
						})}, nil
					}}
					root, err := newCmdRootWithWriters(factory, &stdout, &stderr)
					if err != nil {
						t.Fatal(err)
					}
					args := append([]string{"repo"}, entry.args...)
					if asJSON {
						args = append(args, "--json")
					}
					if raw {
						args = append(args, "--raw-output")
					}
					root.SetArgs(args)
					if err := root.Execute(); err != nil {
						t.Fatal(err)
					}
					if entry.legacy {
						replacement := "view"
						old := "read-file"
						if entry.directory {
							replacement, old = "list", "read-dir"
						}
						want := fmt.Sprintf("Command %q is deprecated, use 'ag-cli repo content %s' instead\n", old, replacement)
						if stderr.String() != want {
							t.Fatalf("stderr = %q, want %q", stderr.String(), want)
						}
					} else if stderr.Len() != 0 {
						t.Fatalf("modern stderr = %q", stderr.String())
					}
					if asJSON {
						if !json.Valid(stdout.Bytes()) {
							t.Fatalf("invalid JSON stdout: %q", stdout.String())
						}
						if entry.directory {
							var entries []map[string]any
							if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil || len(entries) != 1 || entries[0]["path"] != "file" {
								t.Fatalf("directory JSON = %s, error = %v", stdout.String(), err)
							}
						} else {
							var file map[string]any
							if err := json.Unmarshal(stdout.Bytes(), &file); err != nil || file["content"] != base64.StdEncoding.EncodeToString([]byte(content)) {
								t.Fatalf("file JSON = %s, error = %v", stdout.String(), err)
							}
						}
					} else {
						want := content
						if !raw {
							want = strings.ReplaceAll(want, "\x1b", `\x1b`)
						}
						if entry.directory {
							want = "file\t4\tfile\n"
							if !entry.legacy {
								want = "file\tfile\tabc\n"
							}
						}
						if stdout.String() != want {
							t.Fatalf("stdout = %q, want %q", stdout.String(), want)
						}
					}
				})
			}
		}
	}
}

func TestContentCompatibilityMetadata(t *testing.T) {
	root, err := newCmdRootWithWriters(&cmdutil.Factory{}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"read-file", "read-dir"} {
		doc, err := commandschema.Describe(root, []string{"repo", name})
		if err != nil || doc.Command.Deprecated == "" {
			t.Fatalf("schema %s = %#v, error = %v", name, doc.Command, err)
		}
		cmd, _, err := root.Find([]string{"repo", name})
		if err != nil || cmd.IsAvailableCommand() {
			t.Fatalf("%s should remain excluded from regular help", name)
		}
	}
	doc, err := commandschema.Describe(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, cmd := range doc.Commands {
		if cmd.Path == "ag-cli repo read-file" || cmd.Path == "ag-cli repo read-dir" {
			found++
			if cmd.Deprecated == "" {
				t.Fatalf("missing deprecation: %#v", cmd)
			}
		}
	}
	if found != 2 {
		t.Fatalf("compatibility entries = %d, want 2", found)
	}
}

type unavailableDeprecationWriter struct{}

func (unavailableDeprecationWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestContentCompatibilityWarningFailureDoesNotBlockOutput(t *testing.T) {
	for _, old := range []string{"read-file", "read-dir"} {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/raw=%t", old, raw), func(t *testing.T) {
				calls := 0
				f := &cmdutil.Factory{Config: rootTestConfig{}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: rootRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						body := `{"type":"file","name":"file","path":"file","encoding":"base64","content":"b2sK"}`
						if old == "read-dir" {
							body = `[{"type":"file","name":"file","path":"file","size":3}]`
						}
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
					})}, nil
				}}
				var out bytes.Buffer
				root, err := newCmdRootWithWriters(f, &out, unavailableDeprecationWriter{})
				if err != nil {
					t.Fatal(err)
				}
				path := "file"
				if old == "read-dir" {
					path = "."
				}
				args := []string{"repo", old, "owner/repo", path, "--json"}
				if raw {
					args = append(args, "--raw-output")
				}
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				for _, w := range []io.Writer{root.OutOrStdout(), root.ErrOrStderr()} {
					if err := cmdutil.FlushWriter(w); err != nil {
						t.Fatalf("flush after optional warning: %v", err)
					}
				}
				if calls != 1 || !json.Valid(out.Bytes()) {
					t.Fatalf("calls=%d stdout=%q", calls, out.String())
				}
			})
		}
	}
}

func TestContentCompatibilityHelpIncludesMigration(t *testing.T) {
	for _, entry := range []struct{ old, replacement string }{{"read-file", "view"}, {"read-dir", "list"}} {
		for _, viaHelp := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/help=%t", entry.old, viaHelp), func(t *testing.T) {
				var out, stderr bytes.Buffer
				root, err := newCmdRootWithWriters(&cmdutil.Factory{}, &out, &stderr)
				if err != nil {
					t.Fatal(err)
				}
				args := []string{"repo", entry.old, "--help"}
				if viaHelp {
					args = []string{"help", "repo", entry.old}
				}
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				if err := cmdutil.FlushWriter(root.OutOrStdout()); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), "Deprecated:") || !strings.Contains(out.String(), "ag-cli repo content "+entry.replacement) || stderr.Len() != 0 {
					t.Fatalf("stdout=%q stderr=%q", out.String(), stderr.String())
				}
			})
		}
	}
}
